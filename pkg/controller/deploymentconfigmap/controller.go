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

package deploymentconfigmap

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"github.com/ianlewis/controllerutil/logging"
)

var (
	KeyFunc = cache.DeletionHandlingMetaNamespaceKeyFunc
)

// Controller represents a memcached proxy config map controller that
// watches MemcachedProxy objects and creates a ConfigMap used to
// configure mcrouter.
type Controller struct {
	client clientset.Interface

	cmLister corev1listers.ConfigMapLister
	dLister  v1beta1listers.DeploymentLister
	rsLister v1beta1listers.ReplicaSetLister

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
	namespace string,
	configMapInformer cache.SharedIndexInformer,
	deploymentInformer cache.SharedIndexInformer,
	replicasetInformer cache.SharedIndexInformer,
	recorder record.EventRecorder,
	logger *logging.Logger,
	workers int,
) *Controller {
	c := &Controller{
		client:            client,
		ianlewisorgClient: ianlewisorgClient,

		namespace: namespace,

		cmLister: corev1listers.NewConfigMapLister(configMapInformer.GetIndexer()),
		dLister:  v1beta1listers.NewDeploymentLister(deploymentInformer.GetIndexer()),
		rsLister: v1beta1listers.NewReplicaSetLister(deploymentInformer.GetIndexer()),

		recorder: recorder,

		l: logger,

		queue:   workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), name),
		workers: workers,
	}

	deploymentInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueue,
		UpdateFunc: func(old, new interface{}) {
			c.enqueue(new)
		},
		DeleteFunc: c.enqueue,
	})

	replicasetInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueueOwned,
		UpdateFunc: func(old, new interface{}) {
			c.enqueueOwned(new)
		},
		DeleteFunc: c.enqueueOwned,
	})

	// We watch for new configmaps and enqueue them if they are owned
	// by a deployment. Any configmaps owned by the deployment are considered
	// new configmaps that will trigger a rolling update.
	configMapInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueueOwned,
		UpdateFunc: func(old, new interface{}) {
			c.enqueueOwned(new)
		},
		DeleteFunc: c.enqueueOwned,
	})

	// We don't need to watch for configmap changes because they are immutable.
	// We only update configmaps when new replicasets are created. We don't care
	// when configmaps themselves are updated.

	// Add an event handler for endpoints so that we can update mcrouter's configuration
	// when pods in memcached clusters change.
	endpointsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addEndpoints,
		UpdateFunc: c.updateEndpoints,
		DeleteFunc: c.deleteEndpoints,
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

// enqueueOwned enqueues deployments when one of one of its owned objects changes
func (c *Controller) enqueueOwned(obj interface{}) {
	if o, ok := obj.(metav1.Object); ok {
		owner := metav1.GetControllerOf(o)
		if owner != nil {
			if owner.APIVersion == v1beta1.SchemeGroupVersion.String() && owner.Kind == "Deployment" {
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

// configMapForProxy creates a ConfigMap object for proxy configuration based on the given MemcachedProxy
func (c *Controller) configMapForProxy(p *v1alpha1.MemcachedProxy, int64 version) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			// Generate a new unique name on the API server side
			Name:            "",
			GenerateName:    fmt.Sprintf("%s-config-", p.Name),
			Namespace:       p.Namespace,
			OwnerReferences: []metav1.OwnerReference{*controller.NewProxyOwnerRef(p)},
		},
	}

	// Create the ConfigMap data based on the proxy configuration.
	cmData, err := c.configMapDataForProxy(p)
	cm.Data = cmData

	return cm, nil
}

// replicaSetsForDeployment returns the ReplicaSets owned by the given Deployment
func replicaSetsForDeployment(d *v1beta1.Deployment) ([]*v1beta1.ReplicaSet, error) {
	// ReplicaSets are retrieved using the Deployments selector and ownership is verified

	// Generate the selector from the Deployment's selector
	selector, err := metav1.LabelSelectorAsSelector(d.Spec.Selector)
	if err != nil {
		return nil, err
	}

	// List all replicasets matching the selector
	options := metav1.ListOptions{LabelSelector: selector.String()}
	rsList, err := c.rsLister.ReplicaSets(d.Namespace).List(options)
	if err != nil {
		return nil, err
	}

	// Verify ownership of the ReplicaSets and return only the owned ReplicaSets
	owned := make([]*extensions.ReplicaSet, 0, len(all))
	for _, rs := range rsList {
		if metav1.IsControlledBy(rs, d) {
			owned := append(owned, rs)
		}
	}

	return owned, nil
}

// configMapsForProxy returns all configmaps owned by the proxy
func (c *Controller) configMapsForProxy(cmLister corev1listers.ConfigMapLister, p *v1alpha1.MemcachedProxy) ([]*corev1.ConfigMap, error) {
	cmList, err := c.cmLister.ConfigMaps(p.Namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}

	// Each ConfigMap has an owner reference to a ReplicaSet which, in turn, has an owner reference to a Deployment which, in turn, has an owner reference to the MemcachedProxy
	var result []*corev1.ConfigMap
	dList, err := controller.GetDeploymentsForProxy(c.dLister, p)
	if err != nil {
		return nil, err
	}
	for _, d := range dList {
		rsList, err := c.replicaSetsForDeployment(d)
		if err != nil {
			return nil, err
		}
		for _, rs := range rsList {
			for _, cm := range cmList {
				if metav1.IsControlledBy(rs, cm) {
					result := append(result, cm)
				}
			}
		}
	}
	return result, nil
}

// desiredConfigMapForProxy returns the ConfigMap that matches the current cluster state.
// If one already exists that matches the current desired state, it is returned and true
// is returned in the boolean desired flag. If one does not exist a ConfigMap object representing
// the desired state is returned and false is returned in the desired flag. It is the responsibility
// of the caller to actually create the object via the API if no ConfigMap matches the desired state.
func (c *Controller) desiredConfigMapForProxy(p *v1alpha1.MemcachedProxy) (*corev1.ConfigMap, bool, error) {
	// Get a ConfigMap representing the desired state
	desired, err := c.configMapForProxy(p)
	if err != nil {
		return nil, false, err
	}

	// Get all ConfigMaps owned by the proxy
	cmList, err := c.configMapsForProxy(p)
	if err != nil {
		return nil, false, err
	}

	for _, cm := range cmList {
		if apiequality.Semantic.DeepEqual(cm, desired) {
			// A ConfigMap exists that matches the desired state
			return cm, true, nil
		}
	}

	// No ConfigMap exists that matches the desired state
	// Return the desired ConfigMap
	return desired, false, nil
}

// syncHandler runs the controller for the given key which contains a MemcachedProxy namespace and name
func (c *Controller) syncHandler(key string) error {
	// Log how long the sync handler takes to run
	startTime := time.Now()
	c.l.Info.V(4).Printf("started syncing %q (%v)", key, startTime)
	defer func() {
		c.l.Info.V(4).Printf("finished syncing %q (%v)", key, time.Now().Sub(startTime))
	}()

	// Split out the namespace and name of the MemcachedProxy from the key
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
	// TODO: Use generation/ObservedGeneration
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

	// Get the desired ConfigMap and if it has been created already
	cm, exists, err := c.desiredConfigMapForProxy(p)
	if err != nil {
		return err
	}
	if !exists {
		c.l.Info.V(4).Printf("Creating configmap for %q", key)

		cmNew, err = c.client.CoreV1().ConfigMaps(ns).Create(cm)
		if err != nil {
			c.recordConfigMapEvent("create", p, cm, err)
			return err
		}

		c.recordConfigMapEvent("create", p, cmNew, nil)
	}
	return nil
}
