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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	appsv1listers "k8s.io/client-go/listers/apps/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"github.com/ianlewis/controllerutil/logging"

	"github.com/ianlewis/memcached-operator/pkg/apis/ianlewis.org/v1alpha1"
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
	dLister  appsv1listers.DeploymentLister
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
		dLister:  appsv1listers.NewDeploymentLister(deploymentInformer.GetIndexer()),
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

	// Watch for configmap changes so that the deployment is created properly
	// once the required configmap is created.
	configmapInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueueOwned,
		UpdateFunc: func(old, new interface{}) {
			// Enqueue the old owner of the service as well as the new owner
			c.enqueueOwned(old)
			c.enqueueOwned(new)
		},
		DeleteFunc: c.enqueueOwned,
	})

	// Watch deployments for changes and self-heal
	deploymentInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueueOwned,
		UpdateFunc: func(old, new interface{}) {
			// Enqueue the old owner of the service as well as the new owner
			c.enqueueOwned(old)
			c.enqueueOwned(new)
		},
		DeleteFunc: c.enqueueOwned,
	})

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

// enqueueOwned enqueues MemcachedProxy objects in the workqueue when one of
// one of its owned objects changes
func (c *Controller) enqueueOwned(obj interface{}) {
	if o, ok := obj.(metav1.Object); ok {
		owner := metav1.GetControllerOf(o)
		if owner != nil {
			if owner.APIVersion == v1alpha1.SchemeGroupVersion.String() && owner.Kind == "MemcachedProxy" {
				// Enqueue the MemcachedProxy that owns this object
				// KeyFunc only accepts metav1.Objects. OwnerReference doesn't implement metav1.Object so
				// here we use the fact that KeyFunc creates keys of the form <namespace>/<name>
				c.queue.Add(o.GetNamespace() + "/" + owner.Name)
			}
		}
	}
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
	d, dList, err := controller.GetDeploymentsForProxy(c.dLister, p)
	if err != nil {
		return fmt.Errorf("failed to get deployments for %q: %v", key, err)
	}
	if d == nil {
		c.l.Info.V(4).Printf("creating deployment for %q", key)
		err = c.createDeployment(p)
	} else {
		c.l.Info.V(4).Printf("updating deployment for %q", key)
		err = c.updateDeployment(p, d)
	}
	if err != nil {
		return err
	}

	if len(dList) > 0 {
		// More than one Deployment found. This is an error. Try to recover by removing the older ones.
		c.l.Info.V(4).Printf("found multiple deployments for %q", key)

		// Multiple deployments were found so clean up unnecessary deployments.
		// Deployments are returned sorted by the creation timestamp (in reverse order)
		// Here we delete all but the newest
		err := c.deleteDeployments(p, dList[1:])
		if err != nil {
			return err
		}

		return fmt.Errorf("Found multiple deployments for %q. Requeueing...", key)
	}

	return nil
}

// deploymentForProxy creates a deployment object for the given proxy
func (c *Controller) deploymentForProxy(p *v1alpha1.MemcachedProxy) (*appsv1.Deployment, error) {
	cm, err := controller.GetConfigMapForProxy(c.cmLister, p)
	if err != nil {
		return nil, err
	}
	if cm == nil {
		return nil, fmt.Errorf("no configmap for %q", p.Namespace+"/"+p.Name)
	}

	// Create the deployment
	// TODO: Allow control of replica count
	replicas := int32(1)
	d := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "",
			GenerateName:    fmt.Sprintf("%s-mcrouter-", p.Name),
			Namespace:       p.Namespace,
			OwnerReferences: []metav1.OwnerReference{*controller.NewProxyOwnerRef(p)},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: controller.GetProxyServiceSelector(p),
			},
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
								// Disable the async delete log because we won't persist state
								"--asynclog-disable",
								fmt.Sprintf("--stats-root=%s", p.Spec.McRouter.StatsRoot),
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
								{
									Name:      "stats",
									MountPath: p.Spec.McRouter.StatsRoot,
								},
							},
							SecurityContext: p.Spec.McRouter.SecurityContext,
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
						{
							Name: "stats",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}

	return d, nil
}

// createDeployment creates a mcrouter Deployment for the given MemcachedProxy
func (c *Controller) createDeployment(p *v1alpha1.MemcachedProxy) error {
	d, err := c.deploymentForProxy(p)
	if err != nil {
		return err
	}

	result, err := c.client.AppsV1().Deployments(p.Namespace).Create(d)
	if err != nil {
		msg := fmt.Sprintf("failed to create deployment for %q: %v", p.Namespace+"/"+p.Name, err)
		c.recorder.Event(p, corev1.EventTypeWarning, FailedDeploymentCreateReason, msg)
		return fmt.Errorf(msg)
	}

	msg := fmt.Sprintf("deployment %q created", result.Namespace+"/"+result.Name)
	c.recorder.Event(p, corev1.EventTypeNormal, DeploymentCreateReason, msg)

	return nil
}

// updateDeployment updates the existing deployment to reflect the desired state
func (c *Controller) updateDeployment(p *v1alpha1.MemcachedProxy, d *appsv1.Deployment) error {
	cm, err := controller.GetConfigMapForProxy(c.cmLister, p)
	if err != nil {
		return err
	}
	// Updated the deployment with the name of the newest configmap
	for _, v := range d.Spec.Template.Spec.Volumes {
		if v.Name == "config" {
			v.VolumeSource.ConfigMap.LocalObjectReference.Name = cm.Name
		}
	}

	result, err := c.client.AppsV1().Deployments(d.Namespace).Update(d)
	if err != nil {
		msg := fmt.Sprintf("failed to update deployment for %q: %v", p.Namespace+"/"+p.Name, err)
		c.recorder.Event(p, corev1.EventTypeWarning, FailedDeploymentUpdateReason, msg)
		return fmt.Errorf(msg)
	}

	msg := fmt.Sprintf("deployment %q updated", result.Namespace+"/"+result.Name)
	c.recorder.Event(p, corev1.EventTypeNormal, DeploymentUpdateReason, msg)

	return nil
}

// deleteDeployments deletes all deployments in the given list
func (c *Controller) deleteDeployments(p *v1alpha1.MemcachedProxy, dList []*appsv1.Deployment) error {
	for _, d := range dList {
		err := c.client.AppsV1().Deployments(d.Namespace).Delete(d.Name, nil)
		if err != nil {
			return err
		}
		msg := fmt.Sprintf("deleted deployment %q", d.Namespace+"/"+d.Name)
		c.l.Info.V(4).Print(msg)
		c.recorder.Event(p, corev1.EventTypeWarning, DeleteDeploymentReason, msg)
	}
	return nil
}
