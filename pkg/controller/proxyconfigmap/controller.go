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

package proxyconfigmap

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// Controller represents a memcached proxy config map controller that
// watches MemcachedProxy objects and creates a ConfigMap used to
// configure mcrouter.
type Controller struct {
	client            clientset.Interface
	ianlewisorgClient ianlewisorgclientset.Interface

	// Watches are limited to this namespace
	namespace string

	pLister  ianlewisorglisters.MemcachedProxyLister
	cmLister corev1listers.ConfigMapLister
	sLister  corev1listers.ServiceLister
	epLister corev1listers.EndpointsLister

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

// New creates a new memcached proxy configmap controller
func New(
	name string,
	client clientset.Interface,
	ianlewisorgClient ianlewisorgclientset.Interface,
	namespace string,
	proxyInformer cache.SharedIndexInformer,
	configMapInformer cache.SharedIndexInformer,
	serviceInformer cache.SharedIndexInformer,
	endpointsInformer cache.SharedIndexInformer,
	recorder record.EventRecorder,
	logger *logging.Logger,
	workers int,
) *Controller {
	c := &Controller{
		client:            client,
		ianlewisorgClient: ianlewisorgClient,

		pLister:  ianlewisorglisters.NewMemcachedProxyLister(proxyInformer.GetIndexer()),
		cmLister: corev1listers.NewConfigMapLister(configMapInformer.GetIndexer()),
		sLister:  corev1listers.NewServiceLister(serviceInformer.GetIndexer()),
		epLister: corev1listers.NewEndpointsLister(endpointsInformer.GetIndexer()),

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

	// Add an event handler for endpoints so that we can update mcrouter's configuration
	// when pods in memcached clusters change.
	endpointsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addEndpoints,
		UpdateFunc: c.updateEndpoints,
		DeleteFunc: c.deleteEndpoints,
	})

	// TODO: Watch configmap for changes and self-heal

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

// configMapDataForProxy renders the mcrouter configmap data
func (c *Controller) configMapDataForProxy(p *v1alpha1.MemcachedProxy) (map[string]string, error) {
	// Render config file
	config, err := c.configForProxy(p)
	if err != nil {
		return nil, fmt.Errorf("failed to create mcrouter config: %v", err)
	}
	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create render config.json: %v", err)
	}

	data := map[string]string{
		"config.json": string(configJSON),
	}

	return data, nil
}

func (c *Controller) syncHandler(key string) error {
	startTime := time.Now()
	c.l.Info.V(4).Printf("started syncing %q (%v)", key, startTime)
	defer func() {
		c.l.Info.V(4).Printf("finished syncing %q (%v)", key, time.Now().Sub(startTime))
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

	cmList, err := controller.GetConfigMapsForProxy(c.cmLister, p)
	if err != nil {
		return fmt.Errorf("failed to get configmaps for %q: %v", key, err)
	}
	if len(cmList) == 0 {
		c.l.Info.V(4).Printf("Creating configmap for %q", key)

		// Create the ConfigMap based on the proxy configuration.
		cmData, err := c.configMapDataForProxy(p)
		if err != nil {
			return err
		}

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				// Generate a new unique name on the API server side
				Name:            "",
				GenerateName:    fmt.Sprintf("%s-config-", p.Name),
				Namespace:       p.Namespace,
				OwnerReferences: []metav1.OwnerReference{*controller.NewProxyOwnerRef(p)},
			},
			Data: cmData,
		}

		result, err := c.client.CoreV1().ConfigMaps(ns).Create(cm)
		if err != nil {
			msg := fmt.Sprintf("failed to create configmap for %q: %v", key, err)
			logging.PrintMulti(c.l.Error, map[logging.Level]string{
				4: msg,
				9: fmt.Sprintf("failed to create configmap for %q: %v: %#v", key, err, cm),
			})
			c.recorder.Event(p, corev1.EventTypeWarning, FailedProxyConfigMapCreateReason, msg)
			return err
		}

		msg := fmt.Sprintf("configmap %q created", result.Namespace+"/"+result.Name)
		logging.PrintMulti(c.l.Info, map[logging.Level]string{
			4: msg,
			9: fmt.Sprintf("configmap %q created: %#v", result.Namespace+"/"+result.Name, result),
		})
		c.recorder.Event(p, corev1.EventTypeNormal, ProxyConfigMapCreateReason, msg)

		return nil
	}

	// Existing configmaps were found. Get one configmap and delete any others.
	var cm *corev1.ConfigMap
	if len(cmList) == 1 {
		cm = cmList[0]
	} else {
		// Multiple configmaps were found so clean up unnecessary configmaps
		c.l.Info.V(4).Printf("found multiple configmaps for %q", key)
		toDelete := cmList[1:]
		for _, m := range toDelete {
			err := c.client.CoreV1().ConfigMaps(ns).Delete(m.Name, nil)
			if err != nil {
				return err
			}
			msg := fmt.Sprintf("deleting configmap %q", m.Namespace+"/"+m.Name)
			c.l.Info.V(4).Print(msg)
			c.recorder.Event(p, corev1.EventTypeWarning, DeleteConfigMapReason, msg)
		}

		cm = cmList[0]
	}

	// Update configmap if changed
	c.l.Info.V(4).Printf("found existing configmap %q", cm.Namespace+"/"+cm.Name)
	// Create the ConfigMap based on the proxy configuration.
	newData, err := c.configMapDataForProxy(p)
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(cm.Data, newData) {
		cm.Data = newData
		result, err := c.client.CoreV1().ConfigMaps(ns).Update(cm)
		if err != nil {
			msg := fmt.Sprintf("failed to update configmap for %q: %v", key, err)
			logging.PrintMulti(c.l.Error, map[logging.Level]string{
				4: msg,
				9: fmt.Sprintf("failed to update configmap for %q: %v: %#v", key, err, cm),
			})
			c.recorder.Event(p, corev1.EventTypeWarning, FailedProxyConfigMapUpdateReason, msg)
			return err
		}

		logging.PrintMulti(c.l.Info, map[logging.Level]string{
			4: fmt.Sprintf("configmap %q updated", result.Namespace+"/"+result.Name),
			9: fmt.Sprintf("configmap %q updated: %#v", result.Namespace+"/"+result.Name, result),
		})

	}

	return nil
}
