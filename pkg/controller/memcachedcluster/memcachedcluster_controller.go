// Copyright 2017 Google Inc.
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

package memcachedcluster

import (
	"fmt"
	"strconv"
	"text/template"
	"time"

	"github.com/golang/glog"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	v1_listers "k8s.io/client-go/listers/core/v1"
	v1beta1_listers "k8s.io/client-go/listers/extensions/v1beta1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/IanLewis/memcached-operator/pkg/api/ianlewis.org/v1alpha1"
	v1alpha1_listers "github.com/IanLewis/memcached-operator/pkg/listers/ianlewis.org/v1alpha1"
)

var keyFunc = cache.DeletionHandlingMetaNamespaceKeyFunc
var controllerKind = v1alpha1.SchemeGroupVersion.WithKind("MemcachedCluster")

const memcachedPort = 11211

// This template generates a Twemproxy config.yaml where each server has equal weight.
var configYamlTemplate = `pool:
  listen: 0.0.0.0:` + strconv.Itoa(memcachedPort) + `
  hash: {{.Hash}}
  distribution: {{.Distribution}} 
  auto_eject_hosts: true
  server_retry_timeout: {{.ServerRetryTimeout}}
  server_failure_limit: {{.ServerFailureLimit}}
{{if .Servers}}  servers:
{{range .Servers}}    - {{.}}:` + strconv.Itoa(memcachedPort) + `:1
{{end}}{{else}}  servers: []{{end}}`

const (
	// maxRetries is the number of times a Memcached cluster will be retried before it is dropped out of the queue.
	// With the current rate-limiter in use (5ms*2^(maxRetries-1)) the following numbers represent the times
	// a deployment is going to be requeued:
	//
	// 5ms, 10ms, 20ms, 40ms, 80ms, 160ms, 320ms, 640ms, 1.3s, 2.6s, 5.1s, 10.2s, 20.4s, 41s, 82s
	maxRetries = 15
)

// Controller managinges MemcachedClusters.
type Controller struct {
	// client is a Kubernetes API client
	client  clientset.Interface
	cClient *rest.RESTClient

	// Listers for various objects
	// These allow us to list from the cached index.
	clusterLister    v1alpha1_listers.MemcachedClusterLister
	deploymentLister v1beta1_listers.DeploymentLister
	configMapLister  v1_listers.ConfigMapLister
	serviceLister    v1_listers.ServiceLister
	podLister        v1_listers.PodLister

	// cListerSynced returns true if the MemcachedCluster store has been synced at least once.
	// Added as a member to the struct to allow injection for testing.
	cListerSynced cache.InformerSynced
	// dListerSynced returns true if the Deployment store has been synced at least once.
	// Added as a member to the struct to allow injection for testing.
	dListerSynced cache.InformerSynced
	// cmListerSynced returns true if the ConfigMap store has been synced at least once.
	// Added as a member to the struct to allow injection for testing.
	cmListerSynced cache.InformerSynced
	// sListerSynced returns true if the service store has been synced at least once.
	// Added as a member to the struct to allow injection for testing.
	sListerSynced cache.InformerSynced
	// podListerSynced returns true if the pod store has been synced at least once.
	// Added as a member to the struct to allow injection for testing.
	podListerSynced cache.InformerSynced

	// Internal template used to write config.yaml for Twemproxy to a ConfigMap
	configYamlTemplate *template.Template

	// CLusters that need to be synced
	queue workqueue.RateLimitingInterface

	// The number of workes used to sync MemcachedClusters
	numWorkers int
}

// NewMemcachedClusterController creates a new memcached cluster controller instance.
func NewMemcachedClusterController(clusterInformer, deploymentInformer, configMapInformer, serviceInformer, podInformer cache.SharedIndexInformer, client clientset.Interface, cClient *rest.RESTClient, workers int) *Controller {
	cc := &Controller{
		client:           client,
		cClient:          cClient,
		clusterLister:    v1alpha1_listers.NewMemcachedClusterLister(clusterInformer.GetIndexer()),
		deploymentLister: v1beta1_listers.NewDeploymentLister(deploymentInformer.GetIndexer()),
		configMapLister:  v1_listers.NewConfigMapLister(configMapInformer.GetIndexer()),
		serviceLister:    v1_listers.NewServiceLister(serviceInformer.GetIndexer()),
		podLister:        v1_listers.NewPodLister(podInformer.GetIndexer()),

		configYamlTemplate: template.Must(template.New("twemproxyConfigYaml").Parse(configYamlTemplate)),
		queue:              workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "memcachedcluster"),
		numWorkers:         workers,
	}

	clusterInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    cc.addMemcachedCluster,
		UpdateFunc: cc.updateMemcachedCluster,
		DeleteFunc: cc.deleteMemcachedCluster,
	})

	deploymentInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    cc.addDeployment,
		UpdateFunc: cc.updateDeployment,
		DeleteFunc: cc.deleteDeployment,
	})

	configMapInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    cc.addConfigMap,
		UpdateFunc: cc.updateConfigMap,
		DeleteFunc: cc.deleteConfigMap,
	})

	// Pod IPs get assigned post-scheduling so need to watch for updates.
	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		// TODO: Do we need to watch for adds? The Pod IP is likely not assigned yet.
		AddFunc:    cc.addPod,
		UpdateFunc: cc.updatePod,
		DeleteFunc: cc.deletePod,
	})

	cc.cListerSynced = clusterInformer.HasSynced
	cc.dListerSynced = deploymentInformer.HasSynced
	cc.cmListerSynced = configMapInformer.HasSynced
	cc.sListerSynced = serviceInformer.HasSynced
	cc.podListerSynced = podInformer.HasSynced

	return cc
}

// Run starts the controller and associated goroutines.
func (cc *Controller) Run(stopCh <-chan struct{}) error {
	// TODO: support writing events to Kubernetes API server

	defer cc.queue.ShutDown()

	glog.V(3).Infof("Waiting for caches to sync.")

	if !cache.WaitForCacheSync(stopCh, cc.cListerSynced, cc.dListerSynced, cc.cmListerSynced, cc.sListerSynced, cc.podListerSynced) {
		return fmt.Errorf("unable to sync caches")
	}

	glog.V(3).Infof("Caches are synced.")

	for i := 0; i < cc.numWorkers; i++ {
		go wait.Until(cc.worker, time.Second, stopCh)
	}

	<-stopCh

	return nil
}

func (cc *Controller) enqueueCluster(c *v1alpha1.MemcachedCluster) {
	key, err := keyFunc(c)
	if err != nil {
		glog.Errorf("Couldn't get key for MemcachedCluster: %s", err)
		return
	}
	cc.queue.Add(key)
}

// worker runs a worker thread that just dequeues items, processes them, and marks them done.
// It enforces that the syncHandler is never invoked concurrently with the same key.
func (cc *Controller) worker() {
	for cc.processNextWorkItem() {
	}
}

// processNextWorkItem processes the next item in the worker queue.
func (cc *Controller) processNextWorkItem() bool {
	key, quit := cc.queue.Get()
	if quit {
		return false
	}
	defer cc.queue.Done(key)

	err := cc.sync(key.(string))
	if err == nil {
		cc.queue.Forget(key)
		return true
	}

	if cc.queue.NumRequeues(key) < maxRetries {
		if glog.V(9) {
			glog.Errorf("Error syncing cluster with key %q: %#v", key, err)
		} else {
			glog.Errorf("Error syncing cluster with key %q: %s", key, err)
		}
		cc.queue.AddRateLimited(key)
		return true
	}

	utilruntime.HandleError(err)
	if glog.V(9) {
		glog.Errorf("Dropping MemcachedCluster %q out of the queue: %#v", key, err)
	} else {
		glog.Errorf("Dropping MemcachedCluster %q out of the queue: %s", key, err)
	}
	cc.queue.Forget(key)

	return true
}
