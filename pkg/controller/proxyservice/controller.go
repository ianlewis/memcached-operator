package proxyservice

import (
	"context"
	"fmt"
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

	ianlewisorgclientset "github.com/ianlewis/memcached-operator/pkg/client/clientset/versioned"
	ianlewisorglisters "github.com/ianlewis/memcached-operator/pkg/client/listers/ianlewis/v1alpha1"
	"github.com/ianlewis/memcached-operator/pkg/controller"
)

var (
	KeyFunc = cache.DeletionHandlingMetaNamespaceKeyFunc

	memcachedPort = int32(11211)
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

	// TODO: watch services for changes and self-heal

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
			return fmt.Errorf("memcached proxy '%s' in work queue no longer exists", key)
		}

		return err
	}

	// Get the service for this proxy
	sName := controller.GetProxyServiceName(p)
	s, err := c.sLister.Services(ns).Get(sName)
	if err != nil {
		// if the resource doesn't exist we need to create it.
		if errors.IsNotFound(err) {
			c.l.Info.V(4).Printf("Creating service %q", ns+"/"+sName)

			// Create the service
			dName := controller.GetProxyDeploymentName(p)
			s := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:            sName,
					Namespace:       ns,
					OwnerReferences: []metav1.OwnerReference{*controller.NewProxyOwnerRef(p)},
				},
				Spec: corev1.ServiceSpec{
					Selector: map[string]string{
						// TODO: Use a better selector.
						"proxy-name": dName,
					},
					Type: "ClusterIP",
					Ports: []corev1.ServicePort{
						corev1.ServicePort{
							Name:     "memcached",
							Protocol: "TCP",
							Port:     memcachedPort,
							TargetPort: intstr.IntOrString{
								IntVal: memcachedPort,
							},
						},
					},
				},
			}

			result, err := c.client.CoreV1().Services(ns).Create(s)
			if err != nil {
				msg := fmt.Sprintf("failed to create service %q: %v", ns+"/"+sName, err)
				logging.PrintMulti(c.l.Error, map[logging.Level]string{
					4: msg,
					9: fmt.Sprintf("failed to create service %q: %v: %#v", ns+"/"+sName, err, s),
				})
				c.recorder.Event(p, corev1.EventTypeWarning, FailedProxyServiceCreateReason, msg)
				return err
			}

			msg := fmt.Sprintf("service %q created", ns+"/"+sName)
			logging.PrintMulti(c.l.Info, map[logging.Level]string{
				4: msg,
				9: fmt.Sprintf("service %q created: %#v", ns+"/"+sName, result),
			})
			c.recorder.Event(p, corev1.EventTypeNormal, ProxyServiceCreateReason, msg)
		}

		return err
	}

	// TODO: compare service to desired service and self-heal
	c.l.Info.V(4).Printf("found existing service %q", s.Namespace+"/"+s.Name)

	return nil
}
