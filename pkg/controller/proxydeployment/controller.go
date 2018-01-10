// Copyright 2017 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package proxydeployment

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	corev1listers "k8s.io/client-go/listers/core/v1"
	v1beta1listers "k8s.io/client-go/listers/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"github.com/ianlewis/controllerutil/logging"

	ianlewisorgclientset "github.com/ianlewis/memcached-operator/pkg/client/clientset/versioned"
	ianlewisorglisters "github.com/ianlewis/memcached-operator/pkg/client/listers/ianlewis/v1alpha1"
	"github.com/ianlewis/memcached-operator/pkg/controller"
)

var (
	KeyFunc = cache.DeletionHandlingMetaNamespaceKeyFunc
)

// Controller represents a memcached proxy service controller which watches MemcachedProxy objects and creates associated Service objects.
type Controller struct {
	client            clientset.Interface
	ianlewisorgClient ianlewisorgclientset.Interface

	pLister  ianlewisorglisters.MemcachedProxyLister
	dLister  v1beta1listers.DeploymentLister
	cmLister corev1listers.ConfigMapLister

	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	queue   workqueue.RateLimitingInterface
	workers int

	l *logging.Logger
}

func New(
	name string,
	client clientset.Interface,
	ianlewisorgClient ianlewisorgclientset.Interface,
	proxyInformer cache.SharedIndexInformer,
	deploymentInformer cache.SharedIndexInformer,
	configmapInformer cache.SharedIndexInformer,
	recorder record.EventRecorder,
	logger *logging.Logger,
	workers int,
) *Controller {
	c := &Controller{
		client:            client,
		ianlewisorgClient: ianlewisorgClient,

		pLister:  ianlewisorglisters.NewMemcachedProxyLister(proxyInformer.GetIndexer()),
		dLister:  v1beta1listers.NewDeploymentLister(deploymentInformer.GetIndexer()),
		cmLister: corev1listers.NewConfigMapLister(configmapInformer.GetIndexer()),

		recorder: recorder,

		l: logger,

		queue:   workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), name),
		workers: workers,
	}

	proxyInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueue,
		UpdateFunc: func(old, new interface{}) {
			c.enqueue(new)
		},
		DeleteFunc: c.enqueue,
	})

	// TODO: watch deployments for changes and self-heal

	return c
}

func (c *Controller) enqueue(obj interface{}) {
	key, err := KeyFunc(obj)
	if err != nil {
		c.l.Error.Printf("%v", err)
		return
	}
	c.queue.Add(key)
}

func (c *Controller) Run(ctx context.Context) error {
	// Shutdown the queue when the context is canceled. Workers watch the queue to see if it's
	// shut down so this allows workers to know when to stop.
	defer c.queue.ShutDown()

	for i := 0; i < c.workers; i++ {
		go c.worker()
	}

	<-ctx.Done()

	return nil
}

func (c *Controller) worker() {
	for {
		obj, shutdown := c.queue.Get()
		if shutdown {
			return
		}

		if err := c.processWorkItem(obj); err != nil {
			c.l.Error.Printf("%v", err)
			c.queue.AddRateLimited(obj)
		}
	}
}

func (c *Controller) processWorkItem(obj interface{}) error {
	// We call Done here so the workqueue knows we have finished
	// processing this item. We also must remember to call Forget if we
	// do not want this work item being re-queued. For example, we do
	// not call Forget if a transient error occurs, instead the item is
	// put back on the workqueue and attempted again after a back-off
	// period.
	defer c.queue.Done(obj)

	key, ok := obj.(string)
	if !ok {
		// As the item in the workqueue is actually invalid, we call
		// Forget here else we'd go into a loop of attempting to
		// process a work item that is invalid.
		c.queue.Forget(obj)
		return fmt.Errorf("expected string in workqueue but got %#v", obj)
	}

	// Run the syncHandler, passing it the namespace/name string of the
	// Foo resource to be synced.
	if err := c.syncHandler(key); err != nil {
		return fmt.Errorf("error syncing %q: %s", key, err.Error())
	}

	c.queue.Forget(obj)

	return nil
}

func (c *Controller) syncHandler(key string) error {
	startTime := time.Now()
	c.l.Info.V(4).Printf("Started syncing %q (%v)", key, startTime)
	defer func() {
		c.l.Info.V(4).Printf("Finished syncing %q (%v)", key, time.Now().Sub(startTime))
	}()

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return fmt.Errorf("invalid resource key: %s", key)
	}

	// Get the proxy with this namespace/name
	p, err := c.pLister.MemcachedProxies(ns).Get(name)
	if err != nil {
		// The resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			c.l.Info.V(5).Printf("memcached proxy %q was deleted", key)
			return nil
		}

		return err
	}

	// Check if the current version has been observed by the proxy controller
	hash, err := p.Spec.GetHash()
	if err != nil {
		return fmt.Errorf("failed to get hash for %q: %v", key, err)
	}
	if hash != p.Status.ObservedSpecHash {
		// Requeue
		c.l.Info.V(5).Printf("memcached proxy %q status not updated. requeueing", key)
		c.queue.AddRateLimited(key)
		return nil
	}

	// Get the service for this proxy
	dList, err := controller.GetDeploymentsForProxy(c.dLister, p)
	if err != nil {
		return fmt.Errorf("failed to get deployments for %q: %v", key, err)
	}
	if len(dList) == 0 {
		c.l.Info.V(4).Printf("creating deployment for %q", key)

		cm, err := controller.GetConfigMapForProxy(c.cmLister, p)
		if err != nil {
			return err
		}

		// Create the deployment
		replicas := int32(1)
		d := &v1beta1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "",
				GenerateName:    fmt.Sprintf("%s-mcrouter-", p.Name),
				Namespace:       p.Namespace,
				OwnerReferences: []metav1.OwnerReference{*controller.NewProxyOwnerRef(p)},
			},
			Spec: v1beta1.DeploymentSpec{
				Replicas: &replicas,
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: controller.GetProxyServiceSelector(p),
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:    "mcrouter",
								Image:   p.Spec.McRouter.Image,
								Command: []string{"mcrouter"},
								Args: []string{
									"-p", fmt.Sprint(*p.Spec.McRouter.Port),
									"--config-file=/etc/mcrouter/config.json",
								},
								Ports: []corev1.ContainerPort{
									{
										Name:          "mcrouter",
										ContainerPort: *p.Spec.McRouter.Port,
									},
								},
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      "config",
										MountPath: "/etc/mcrouter",
									},
								},
							},
						},
						Volumes: []corev1.Volume{
							{
								Name: "config",
								VolumeSource: corev1.VolumeSource{
									ConfigMap: &corev1.ConfigMapVolumeSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: cm.Name,
										},
									},
								},
							},
						},
					},
				},
			},
		}

		result, err := c.client.ExtensionsV1beta1().Deployments(ns).Create(d)
		if err != nil {
			msg := fmt.Sprintf("failed to create depoyment for %q: %v", key, err)
			logging.PrintMulti(c.l.Error, map[logging.Level]string{
				4: msg,
				9: fmt.Sprintf("failed to create deployment for %q: %v: %#v", key, err, d),
			})
			c.recorder.Event(p, corev1.EventTypeWarning, FailedDeploymentCreateReason, msg)
			return err
		}

		msg := fmt.Sprintf("deployment %q created", result.Namespace+"/"+result.Name)
		logging.PrintMulti(c.l.Info, map[logging.Level]string{
			4: msg,
			9: fmt.Sprintf("deployment %q created: %#v", result.Namespace+"/"+result.Name, result),
		})
		c.recorder.Event(p, corev1.EventTypeNormal, DeploymentCreateReason, msg)

		return nil
	}

	var d *v1beta1.Deployment
	if len(dList) == 1 {
		d = dList[0]
	} else {
		// Multiple deployments were found so clean up unnecessary deployments
		c.l.Info.V(4).Printf("found multiple deployments for %q", key)
		toDelete := dList[1:]
		for _, m := range toDelete {
			err := c.client.ExtensionsV1beta1().Deployments(ns).Delete(m.Name, nil)
			if err != nil {
				return err
			}
			msg := fmt.Sprintf("deleting deployment %q", m.Namespace+"/"+m.Name)
			c.l.Info.V(4).Print(msg)
			c.recorder.Event(p, corev1.EventTypeWarning, DeleteDeploymentReason, msg)
		}

		d = dList[0]
	}

	// TODO: compare deployment to desired deployment self-heal
	c.l.Info.V(4).Printf("found existing deployment%q", d.Namespace+"/"+d.Name)

	return nil
}
