package operator

import (
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	"github.com/IanLewis/memcached-operator/pkg/api/ianlewis.org/v1alpha1"
	"github.com/IanLewis/memcached-operator/pkg/controller/memcachedcluster"
	"github.com/IanLewis/memcached-operator/pkg/controller/memcachedproxyservice"
)

const jitterFactor = 1.0
const minResyncPeriod = 12 * time.Hour

// Operator manages all goroutines for the application.
type Operator struct {
	// client is a Kubernetes API client
	client clientset.Interface

	cInformer   cache.SharedIndexInformer
	dInformer   cache.SharedIndexInformer
	cmInformer  cache.SharedIndexInformer
	sInformer   cache.SharedIndexInformer
	podInformer cache.SharedIndexInformer

	cController *memcachedcluster.Controller
	sController *memcachedproxyservice.Controller
}

// NewOperator creates a new Operator instance.
func NewOperator(client clientset.Interface, cClient *rest.RESTClient, workers int) *Operator {

	// TODO: Move informer initialization out of main()
	clusterInformer := cache.NewSharedIndexInformer(
		cache.NewListWatchFromClient(
			cClient,
			"memcachedclusters",
			metav1.NamespaceAll,
			fields.Everything(),
		),
		&v1alpha1.MemcachedCluster{},
		wait.Jitter(minResyncPeriod, jitterFactor),
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)
	deploymentInformer := cache.NewSharedIndexInformer(
		cache.NewListWatchFromClient(
			client.ExtensionsV1beta1().RESTClient(),
			"deployments",
			metav1.NamespaceAll,
			fields.Everything(),
		),
		&v1beta1.Deployment{},
		wait.Jitter(minResyncPeriod, jitterFactor),
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)
	configMapInformer := cache.NewSharedIndexInformer(
		cache.NewListWatchFromClient(
			client.CoreV1().RESTClient(),
			"configmaps",
			metav1.NamespaceAll,
			fields.Everything(),
		),
		&v1.ConfigMap{},
		wait.Jitter(minResyncPeriod, jitterFactor),
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)
	serviceInformer := cache.NewSharedIndexInformer(
		cache.NewListWatchFromClient(
			client.CoreV1().RESTClient(),
			"services",
			metav1.NamespaceAll,
			fields.Everything(),
		),
		&v1.Service{},
		wait.Jitter(minResyncPeriod, jitterFactor),
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)
	podInformer := cache.NewSharedIndexInformer(
		cache.NewListWatchFromClient(
			client.CoreV1().RESTClient(),
			"pods",
			metav1.NamespaceAll,
			fields.Everything(),
		),
		&v1.Pod{},
		wait.Jitter(minResyncPeriod, jitterFactor),
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)

	o := &Operator{
		client: client,

		cInformer:   clusterInformer,
		dInformer:   deploymentInformer,
		cmInformer:  configMapInformer,
		sInformer:   serviceInformer,
		podInformer: podInformer,

		cController: memcachedcluster.NewMemcachedClusterController(
			clusterInformer,
			deploymentInformer,
			configMapInformer,
			serviceInformer,
			podInformer,
			client,
			cClient,
			// TODO: Make cluster controller workers an option
			workers,
		),

		sController: memcachedproxyservice.NewMemcachedProxyServiceController(
			clusterInformer,
			serviceInformer,
			client,
			cClient,
			// TODO: Make service controller workers an option
			workers,
		),
	}

	return o
}

// Run starts the operator and required goroutines
func (o *Operator) Run(stopCh <-chan struct{}) error {
	var wg errgroup.Group

	backoff := wait.Backoff{
		Duration: 1 * time.Second,
		Factor:   1.1,
		Steps:    10,
	}

	glog.V(3).Infof("Initializing operator.")
	err := wait.ExponentialBackoff(backoff, func() (bool, error) {
		// Exit if there is a value on the stopCh
		select {
		case <-stopCh:
			return true, nil
		default:
		}

		// Attempt to initialize.
		err := o.init()
		if err != nil {
			return false, err
		}

		return true, nil
	})
	if err != nil {
		return fmt.Errorf("failed to initialize operator: %v", err)
	}

	wg.Go(func() error { return o.cController.Run(stopCh) })
	wg.Go(func() error { return o.sController.Run(stopCh) })
	wg.Go(func() error {
		o.cInformer.Run(stopCh)
		return nil
	})
	wg.Go(func() error {
		o.dInformer.Run(stopCh)
		return nil
	})
	wg.Go(func() error {
		o.cmInformer.Run(stopCh)
		return nil
	})
	wg.Go(func() error {
		o.sInformer.Run(stopCh)
		return nil
	})
	wg.Go(func() error {
		o.podInformer.Run(stopCh)
		return nil
	})

	return wg.Wait()
}

// init() Creates the necessary memcached-cluster ThirdPartyResource object if it does not exist already.
func (o *Operator) init() error {
	// TODO: Use CustomResourceDefinition
	_, err := o.client.ExtensionsV1beta1().ThirdPartyResources().Get("memcached-cluster.ianlewis.org", metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			tpr := &v1beta1.ThirdPartyResource{
				ObjectMeta: metav1.ObjectMeta{
					Name: "memcached-cluster.ianlewis.org",
				},
				Versions: []v1beta1.APIVersion{
					{Name: "v1alpha1"},
				},
				Description: "A Memcached Cluster Resource",
			}

			result, err := o.client.ExtensionsV1beta1().ThirdPartyResources().Create(tpr)
			if err != nil {
				if glog.V(9) {
					glog.Errorf("Error creating ThirdPartyResource: %#v with error %#v", tpr, err)
				} else if glog.V(2) {
					glog.Errorf("Error creating ThirdPartyResource: %s", err)
				}
				return err
			}

			if glog.V(9) {
				glog.Infof("Created ThirdPartyResource %#v from result %#v", tpr, result)
			} else if glog.V(2) {
				glog.Infof("Created ThirdPartyResource %q", tpr.Name)
			}
		} else {
			return err
		}
	}

	return nil
}
