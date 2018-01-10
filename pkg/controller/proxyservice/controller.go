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

package proxyservice

import (
	"context"
	"fmt"
	"reflect"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	clientset "k8s.io/client-go/kubernetes"
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

	pLister ianlewisorglisters.MemcachedProxyLister
	sLister corev1listers.ServiceLister

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
	serviceInformer cache.SharedIndexInformer,
	recorder record.EventRecorder,
	logger *logging.Logger,
	workers int,
) *Controller {
	c := &Controller{
		client:            client,
		ianlewisorgClient: ianlewisorgClient,

		pLister: ianlewisorglisters.NewMemcachedProxyLister(proxyInformer.GetIndexer()),
		sLister: corev1listers.NewServiceLister(serviceInformer.GetIndexer()),

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

	// Watch services for changes and self-heal.
	serviceInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
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

// enqueue enqueues MemcachedProxy objects in the workqueue when it changes
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

func serviceSpecForProxy(p *v1alpha1.MemcachedProxy) corev1.ServiceSpec {
	return corev1.ServiceSpec{
		Selector: controller.GetProxyServiceSelector(p),
		Type:     "ClusterIP",
		Ports: []corev1.ServicePort{
			{
				Name:     "memcached",
				Protocol: "TCP",
				Port:     *p.Spec.McRouter.Port,
				TargetPort: intstr.IntOrString{
					IntVal: *p.Spec.McRouter.Port,
				},
			},
		},
	}
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
	sName := controller.GetProxyServiceName(p)
	s, err := c.sLister.Services(ns).Get(sName)
	if err != nil {
		// if the resource doesn't exist we need to create it.
		if errors.IsNotFound(err) {
			c.l.Info.V(4).Printf("creating service %q", ns+"/"+sName)

			// Create the service
			s := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:            sName,
					Namespace:       ns,
					OwnerReferences: []metav1.OwnerReference{*controller.NewProxyOwnerRef(p)},
				},
				Spec: serviceSpecForProxy(p),
			}

			_, err := c.client.CoreV1().Services(ns).Create(s)
			if err != nil {
				c.recordServiceEvent("create", p, s, err)
				return err
			}

			c.recordServiceEvent("create", p, s, nil)
		}

		return err
	}

	// compare service to desired service and self-heal
	// update service if changed
	c.l.Info.V(4).Printf("found existing service %q", s.Namespace+"/"+s.Name)
	serviceSpec := serviceSpecForProxy(p)
	if !reflect.DeepEqual(s.Spec, serviceSpec) {
		s.Spec = serviceSpec
		_, err = c.client.CoreV1().Services(ns).Update(s)
		if err != nil {
			c.recordServiceEvent("update", p, s, err)
			return err
		}

		c.recordServiceEvent("update", p, s, nil)
	}

	return nil
}
