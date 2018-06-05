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
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
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
	"github.com/ianlewis/memcached-operator/pkg/util/ownerref"
)

var (
	KeyFunc            = cache.DeletionHandlingMetaNamespaceKeyFunc
	memcachedProxyType = v1alpha1.SchemeGroupVersion.WithKind("MemcachedProxy")
	replicaSetType     = appsv1.SchemeGroupVersion.WithKind("ReplicaSet")
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
	dLister  appsv1listers.DeploymentLister
	rsLister appsv1listers.ReplicaSetLister
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
	deploymentInformer cache.SharedIndexInformer,
	replicasetInformer cache.SharedIndexInformer,
	serviceInformer cache.SharedIndexInformer,
	endpointsInformer cache.SharedIndexInformer,
	recorder record.EventRecorder,
	logger *logging.Logger,
	workers int,
) *Controller {
	c := &Controller{
		client:            client,
		ianlewisorgClient: ianlewisorgClient,

		namespace: namespace,

		pLister:  ianlewisorglisters.NewMemcachedProxyLister(proxyInformer.GetIndexer()),
		cmLister: corev1listers.NewConfigMapLister(configMapInformer.GetIndexer()),
		dLister:  appsv1listers.NewDeploymentLister(deploymentInformer.GetIndexer()),
		rsLister: appsv1listers.NewReplicaSetLister(replicasetInformer.GetIndexer()),
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

	// Add an event handler for deployments to catch when new replicasets
	// are added etc. This is ok because deployment controller updates
	// the deployment's status thus we can observe changes at the deployment
	// level.
	deploymentInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueueOwned,
		UpdateFunc: func(old, new interface{}) {
			// Enqueue the old owner and new owner. Usually they will be the
			// same but just in case the owners change we want to be able to
			// work properly.
			c.enqueueOwned(old)
			c.enqueueOwned(new)
		},
		DeleteFunc: c.enqueueOwned,
	})

	return c
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

// configMapForProxy gets a configmap based on state of pods in referenced memcached clusters.
func (c *Controller) configMapForProxy(p *v1alpha1.MemcachedProxy) (*corev1.ConfigMap, error) {
	// Render config file
	config, err := c.configForProxy(p)
	if err != nil {
		return nil, fmt.Errorf("failed to create mcrouter config: %v", err)
	}
	configBytes, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create render config.json: %v", err)
	}
	configJSON := string(configBytes)
	data := map[string]string{
		"config.json": configJSON,
	}

	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			// Generate the name of the configmap
			Name:            controller.MakeName(fmt.Sprintf("%s-config-", p.Name), []string{p.Name, configJSON}),
			Namespace:       p.Namespace,
			OwnerReferences: []metav1.OwnerReference{*controller.NewProxyOwnerRef(p)},
		},
		Data: data,
	}

	return cm, nil
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

	// Process configmaps that are new (ownerreference pointing to the memcachedproxy)
	cmList, err := controller.GetConfigMapsForProxy(c.cmLister, p)
	if err != nil {
		return fmt.Errorf("failed to get configmaps for %q: %v", key, err)
	}
	switch len(cmList) {
	case 0:
		c.l.Info.V(4).Printf("No new configmaps found")
		// There are no new configmaps. Check if a new one needs to be created.
		cmNew, err := c.configMapForProxy(p)
		if err != nil {
			c.recordEvent("create", p, "ConfigMap", nil, err)
			return err
		}
		cm, err := c.cmLister.ConfigMaps(p.Namespace).Get(cmNew.Name)
		if err != nil {
			if errors.IsNotFound(err) {
				// The ConfigMap doesn't exist and needs to be created.
				c.l.Info.V(4).Printf("Creating configmap for %q", key)

				cm, err = c.client.CoreV1().ConfigMaps(ns).Create(cmNew)
				if err != nil {
					c.recordConfigMapEvent("create", p, cm, err)
					return err
				}

				c.recordConfigMapEvent("create", p, cm, nil)

				// Requeue in case any new cluster changes happened in addition to a replicaset being created
				c.queue.AddRateLimited(key)

				return nil
			} else {
				// Some other kind of error occurred.
				return err
			}
		}

		// A configmap exists with a name that matches a hash of the current cluster state.
		// This is either because it is the latest configmap or because the cluster state
		// has returned to a state that was seen previously. Here we will reuse the existing
		// configmap by adding and ownerref to the memcachedproxy. It will, however, retain
		// it's previous ownerref to it's previous replicaset.
		if err := c.reuseConfigMap(p, cm); err != nil {
			return err
		}

		// Don't need to requeue because we have to wait for a replicaset to be created by the deployment controller.
	case 1:
		// There is only one new configmap.
		// Check if a replicaset has been created for the configmap. If so update the ownerref to point to the replicaset.
		cm := cmList[0]
		c.l.Info.V(4).Printf("One new configmap found: %s", cm.Name)
		applied, err := c.applyOwnerRefToCM(p, cm)
		if err != nil {
			return fmt.Errorf("failed to apply ownerrefs to configmap for %q: %v", key, err)
		}

		if applied {
			// Update the configmap to the api server
			cm, err = c.client.CoreV1().ConfigMaps(cm.Namespace).Update(cm)
			if err != nil {
				c.recordConfigMapEvent("update", p, cm, err)
				return fmt.Errorf("failed to update configmaps for %q: %v", key, err)
			}
			c.recordConfigMapEvent("update", p, cm, nil)

			// Requeue in case any new cluster changes happened in addition to a replicaset being created
			c.queue.AddRateLimited(key)
		} else {
			c.l.Info.V(4).Printf("No replicaset created yet. requeueing...")
			// Enqueue after 1 second. This should be enough under normal operation.
			// TODO: Backoff requeue just in case something bad happens.
			c.queue.AddAfter(key, time.Second)
		}
	default:
		// There are more than one new configmaps. This shouldn't happen but we should try to recover.
		// TODO: Delete any configmaps with only one ownerref pointing to the memcachedproxy. Remove ownerrefs to the memcachedproxy for configmaps with other ownerrefs.
		c.l.Error.Printf("More than one new configmap found for %q", key)
	}

	return nil
}

// reuseConfigMap reuses an existing configmap. It first checks if the ConfigMap is not the
// latest version of and if it isn't then it updates the ConfigMap with a ownerref pointing
// to the MemcachedProxy
func (c *Controller) reuseConfigMap(p *v1alpha1.MemcachedProxy, cm *corev1.ConfigMap) error {
	// Only update the ConfigMap if the ConfigMap is not the latest ConfigMap version.
	// Check the deployment's podspec to see if it references the ConfigMap. If it does
	// then the ConfigMap is the latest
	isLatest := false

	d, _, err := controller.GetDeploymentsForProxy(c.dLister, p)
	if err != nil {
		return err
	}
	if d == nil {
		return fmt.Errorf("No deployment found for \"%s/%s\"", p.Namespace, p.Name)
	}

	for _, v := range d.Spec.Template.Spec.Volumes {
		if v.VolumeSource.ConfigMap != nil && v.VolumeSource.ConfigMap.Name == cm.Name {
			// The Deployment references the configmap so the configmap is the latest
			isLatest = true
		}
	}
	if !isLatest {
		cm.SetOwnerReferences(append(cm.GetOwnerReferences(), *metav1.NewControllerRef(p, memcachedProxyType)))
	}

	// Update the configmap to the api server
	cm, err = c.client.CoreV1().ConfigMaps(cm.Namespace).Update(cm)
	if err != nil {
		c.recordConfigMapEvent("update", p, cm, err)
		return fmt.Errorf("failed to update configmaps for \"%s/%s\": %v", p.Namespace, p.Name, err)
	}
	c.recordConfigMapEvent("update", p, cm, nil)

	return nil
}

// applyOwnerRefToCM applies owner references to the configmap for each replicaset owned by the given memcachedproxy whose podspec references the configmap. Ownerreferences to the memcachedproxy are removed.
func (c *Controller) applyOwnerRefToCM(p *v1alpha1.MemcachedProxy, cm *corev1.ConfigMap) (bool, error) {
	// Add ownerref to replicasets
	rs, err := c.getNewRSForProxy(p)
	if err != nil {
		return false, err
	}
	if rs == nil {
		return false, nil
	}

	// Do not add a new ownerref if the configmap is already owned. This could happen if
	// both the configmap and replicaset are reused.
	if !ownerref.IsOwnedBy(cm, rs) {
		c.l.Info.V(4).Printf("add rs owner ref %q", cm.Name)
		cm.SetOwnerReferences(append(cm.GetOwnerReferences(), *ownerref.NewOwnerRef(rs, replicaSetType)))
	}

	ownerRefs := []metav1.OwnerReference{}
	for _, ref := range cm.GetOwnerReferences() {
		// Add all but references to the memcachedproxy
		if ref.UID != p.GetUID() {
			ownerRefs = append(ownerRefs, ref)
		}
	}
	c.l.Info.V(4).Printf("remove proxy ownerref %q %v", cm.Name, ownerRefs)
	cm.SetOwnerReferences(ownerRefs)

	return true, nil
}

// EqualIgnoreHash returns true if two given podTemplateSpec are equal, ignoring the diff in value of Labels[pod-template-hash]
// We ignore pod-template-hash because:
// 1. The hash result would be different upon podTemplateSpec API changes
//    (e.g. the addition of a new field will cause the hash code to change)
// 2. The deployment template won't have hash labels
// Copied from kubernetes pkg/controller/deployment/util/deployment_util.go
func EqualIgnoreHash(template1, template2 *corev1.PodTemplateSpec) bool {
	t1Copy := template1.DeepCopy()
	t2Copy := template2.DeepCopy()
	// Remove hash labels from template.Labels before comparing
	delete(t1Copy.Labels, appsv1.DefaultDeploymentUniqueLabelKey)
	delete(t2Copy.Labels, appsv1.DefaultDeploymentUniqueLabelKey)
	return apiequality.Semantic.DeepEqual(t1Copy, t2Copy)
}

// getNewRSForProxy retrieves the newest replicaset for the Deployment managed by the MemcachedProxy. If no replicaset exists yet then return nil.
func (c *Controller) getNewRSForProxy(p *v1alpha1.MemcachedProxy) (*appsv1.ReplicaSet, error) {
	d, _, err := controller.GetDeploymentsForProxy(c.dLister, p)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, nil
	}

	selector, err := metav1.LabelSelectorAsSelector(d.Spec.Selector)
	if err != nil {
		return nil, err
	}

	// Select replicasects for the deployment
	rsList, err := c.rsLister.ReplicaSets(p.Namespace).List(selector)
	if err != nil {
		return nil, err
	}

	// Find the new ReplicaSet by comparing it's pod spec to the deployment's pod spec
	for _, rs := range rsList {
		if EqualIgnoreHash(&rs.Spec.Template, &d.Spec.Template) {
			return rs, nil
		}
	}

	return nil, nil
}
