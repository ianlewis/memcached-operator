package memcachedproxyservice

import (
	"fmt"
	"time"

	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/IanLewis/memcached-operator/pkg/api/ianlewis.org/v1alpha1"
	"github.com/IanLewis/memcached-operator/pkg/controller"
)

func (sc *Controller) sync(key string) error {
	startTime := time.Now()
	glog.V(2).Infof("Started syncing MemcachedCluster for key %q (%v)", key, startTime)
	defer func() {
		glog.V(2).Infof("Finished syncing MemcachedCluster for key %q (%v)", key, time.Now().Sub(startTime))
	}()

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	c, err := sc.clusterLister.MemcachedClusters(namespace).Get(name)
	if errors.IsNotFound(err) {
		glog.V(3).Infof("MemcachedCluster for key %q has been deleted", key)
		return nil
	}
	if err != nil {
		return err
	}

	// Wait until the latest state of the has been observed by the MemcachedCluster controller.
	// We want it to have the latest status updates, defaults etc.
	if !controller.IsCurrent(c) {
		return fmt.Errorf("cluster %q is not current: %v", err)
	}

	s, err := controller.GetProxyServiceForCluster(sc.serviceLister, c)
	switch {
	case errors.IsNotFound(err):
		glog.V(3).Infof("Twemproxy Service for MemcachedCluster %q not found. Creating.", c.Name)
		s, err = sc.createTwemproxyService(c)
		if err != nil {
			return fmt.Errorf("failed to create service for cluster %q: %v", c.Name, err)
		}
		controller.LogObject("Service created: ", s)

		return nil

	case err != nil:
		// Some other error occurred.
		return fmt.Errorf("failed to get service for cluster %q: %v", c.Name, err)
	}

	// This Service is owned by the cluster.
	// TODO: Compare Service against desired Service to see if it's been modified. Revert if so.

	return nil
}

func (sc *Controller) createTwemproxyService(c *v1alpha1.MemcachedCluster) (*v1.Service, error) {
	name := controller.GetProxyServiceName(c)
	deployName := controller.GetTwemproxyDeploymentName(c)

	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       c.Namespace,
			OwnerReferences: []metav1.OwnerReference{*controller.NewClusterOwnerRef(c)},
		},
		Spec: v1.ServiceSpec{
			Selector: map[string]string{
				// TODO: Use a better selector.
				"twemproxy-name": deployName,
			},
			Type: "ClusterIP",
			Ports: []v1.ServicePort{
				v1.ServicePort{
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

	result, err := sc.client.CoreV1().Services(c.Namespace).Create(service)
	if err != nil {
		// Return the Service we attempted to created w/ the error
		return service, err
	}

	// Return the created Service
	return result, nil
}
