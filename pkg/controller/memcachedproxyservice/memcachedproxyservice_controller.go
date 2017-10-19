package memcachedproxyservice

import (
	"fmt"
	"time"

	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	v1_listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/IanLewis/memcached-operator/pkg/api/ianlewis.org/v1alpha1"
	"github.com/IanLewis/memcached-operator/pkg/controller"
	v1alpha1_listers "github.com/IanLewis/memcached-operator/pkg/listers/ianlewis.org/v1alpha1"
)

var keyFunc = cache.DeletionHandlingMetaNamespaceKeyFunc
var controllerKind = v1alpha1.SchemeGroupVersion.WithKind("MemcachedCluster")

const (
	memcachedPort = 11211

	// maxRetries is the number of times a Memcached cluster will be retried before it is dropped out of the queue.
	// With the current rate-limiter in use (5ms*2^(maxRetries-1)) the following numbers represent the times
	// a deployment is going to be requeued:
	//
	// 5ms, 10ms, 20ms, 40ms, 80ms, 160ms, 320ms, 640ms, 1.3s, 2.6s, 5.1s, 10.2s, 20.4s, 41s, 82s
	maxRetries = 15
)

// Controller manages creation of Service objects for the memcached proxy for each MemcachedCluster.
type Controller struct {
	// client is a Kubernetes API client
	client  clientset.Interface
	cClient *rest.RESTClient

	// Listers for various objects
	// These allow us to list from the cached index.
	clusterLister v1alpha1_listers.MemcachedClusterLister
	serviceLister v1_listers.ServiceLister

	// cListerSynced returns true if the MemcachedCluster store has been synced at least once.
	// Added as a member to the struct to allow injection for testing.
	cListerSynced cache.InformerSynced
	// sListerSynced returns true if the service store has been synced at least once.
	// Added as a member to the struct to allow injection for testing.
	sListerSynced cache.InformerSynced

	// Clusters that need to be synced
	queue workqueue.RateLimitingInterface

	// The number of workes used to sync MemcachedClusters
	numWorkers int
}

func NewMemcachedProxyServiceController(clusterInformer, serviceInformer cache.SharedIndexInformer, client clientset.Interface, cClient *rest.RESTClient, workers int) *Controller {
	sc := &Controller{
		client:  client,
		cClient: cClient,

		clusterLister: v1alpha1_listers.NewMemcachedClusterLister(clusterInformer.GetIndexer()),
		serviceLister: v1_listers.NewServiceLister(serviceInformer.GetIndexer()),
		queue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "memcachedproxyservice"),
		numWorkers:    workers,
	}

	clusterInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: sc.enqueueCluster,
		UpdateFunc: func(old, cur interface{}) {
			sc.enqueueCluster(cur)
		},
		DeleteFunc: sc.enqueueCluster,
	})

	serviceInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: sc.enqueueClusterFromService,
		UpdateFunc: func(old, cur interface{}) {
			sc.enqueueClusterFromService(cur)
		},
		DeleteFunc: sc.enqueueClusterFromService,
	})

	sc.cListerSynced = clusterInformer.HasSynced
	sc.sListerSynced = serviceInformer.HasSynced

	return sc
}

// Run starts the controller and associated goroutines.
func (sc *Controller) Run(stopCh <-chan struct{}) error {
	// TODO: support writing events to Kubernetes API server
	defer sc.queue.ShutDown()

	glog.V(3).Infof("Waiting for caches to sync.")
	if !cache.WaitForCacheSync(stopCh, sc.cListerSynced, sc.sListerSynced) {
		return fmt.Errorf("unable to sync caches")
	}

	glog.V(3).Infof("Caches are synced.")

	for i := 0; i < sc.numWorkers; i++ {
		go wait.Until(sc.worker, time.Second, stopCh)
	}

	<-stopCh

	return nil
}

// obj could be an *v1alpha1.MemcachedCluster or a DeletionFinalStateUnknown marker item.
func (sc *Controller) enqueueCluster(obj interface{}) {
	key, err := keyFunc(obj)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("failed to get key for object %#v: %v", err))
		return
	}
	sc.queue.Add(key)
}

// obj could be an *v1alpha1.MemcachedCluster or a DeletionFinalStateUnknown marker item.
func (sc *Controller) enqueueClusterFromService(obj interface{}) {
	s, ok := obj.(*v1.Service)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			glog.Errorf("Couldn't get object from tombstone %#v", obj)
			return
		}
		s, ok = tombstone.Obj.(*v1.Service)
		if !ok {
			glog.Errorf("Tombstone contained object that is not a Service %#v", obj)
			return
		}
	}

	// Verify that this is a Service owned by the cluster.
	c, err := controller.GetClusterForObject(sc.clusterLister, s)
	if err != nil {
		if errors.IsNotFound(err) {
			// MemcachedCluster has been deleted.
			return
		}

		if glog.V(9) {
			glog.Errorf("Error getting cluster for Service %#v: %#v", s, err)
		} else {
			glog.Errorf("Error getting cluster for Service %q: %s", s.Name, err)
		}
		return
	}

	if c == nil {
		// Service not owned by a MemcachedCluster
		return
	}

	sc.enqueueCluster(c)
}

// worker runs a worker thread that just dequeues items, processes them, and marks them done.
// It enforces that the syncHandler is never invoked concurrently with the same key.
func (sc *Controller) worker() {
	for sc.processNextWorkItem() {
	}
}

// processNextWorkItem processes the next item in the worker queue.
func (sc *Controller) processNextWorkItem() bool {
	key, quit := sc.queue.Get()
	if quit {
		return false
	}
	defer sc.queue.Done(key)

	err := sc.sync(key.(string))
	if err == nil {
		sc.queue.Forget(key)
		return true
	}

	if sc.queue.NumRequeues(key) < maxRetries {
		if glog.V(9) {
			glog.Errorf("Error syncing proxy service with key %q: %#v", key, err)
		} else {
			glog.Errorf("Error syncing proxy service with key %q: %s", key, err)
		}
		sc.queue.AddRateLimited(key)
		return true
	}

	utilruntime.HandleError(err)
	if glog.V(9) {
		glog.Errorf("Dropping proxy service (%q) out of the queue: %#v", key, err)
	} else {
		glog.Errorf("Dropping proxy service (%q) out of the queue: %s", key, err)
	}
	sc.queue.Forget(key)

	return true
}
