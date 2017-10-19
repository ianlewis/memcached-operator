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

// TODO: Delete service.go. Currently left for reference.
// import (
// 	"fmt"

// 	"github.com/golang/glog"

// 	"k8s.io/apimachinery/pkg/api/errors"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// 	"k8s.io/apimachinery/pkg/util/intstr"
// 	"k8s.io/client-go/pkg/api/v1"
// 	"k8s.io/client-go/tools/cache"

// 	"github.com/IanLewis/memcached-operator/pkg/api/ianlewis.org/v1alpha1"
// )

// func (cc *Controller) addService(obj interface{}) {
// 	s := obj.(*v1.Service)

// 	// Verify that this is a service owned by the cluster.
// 	c, err := cc.getClusterForObject(s)
// 	if err != nil {
// 		if errors.IsNotFound(err) {
// 			// MemcachedCluster has been deleted.
// 			return
// 		}

// 		if glog.V(9) {
// 			glog.Errorf("Error getting cluster for service %#v: %#v", s, err)
// 		} else {
// 			glog.Errorf("Error getting cluster for service %q: %s", s.Name, err)
// 		}
// 		return
// 	}

// 	if c == nil {
// 		// Service not owned by a MemcachedCluster
// 		return
// 	}

// 	logObject("Adding Service: ", s)

// 	cc.enqueueCluster(c)
// }

// func (cc *Controller) updateService(old, cur interface{}) {
// 	oldS := old.(*v1.Service)
// 	curS := cur.(*v1.Service)

// 	// TODO: Sync old cluster if cluster owner changed?

// 	// Verify that this is a Service owned by the cluster.
// 	c, err := cc.getClusterForObject(curS)
// 	if err != nil {
// 		if errors.IsNotFound(err) {
// 			// MemcachedCluster has been deleted.
// 			return
// 		}

// 		if glog.V(9) {
// 			glog.Errorf("Error getting cluster for Service %#v: %#v", curS, err)
// 		} else {
// 			glog.Errorf("Error getting cluster for Service %q: %s", curS.Name, err)
// 		}
// 		return
// 	}

// 	if c == nil {
// 		// Service not owned by a MemcachedCluster
// 		return
// 	}

// 	logObject("Updating Service: ", oldS, curS)

// 	cc.enqueueCluster(c)
// }

// func (cc *Controller) deleteService(obj interface{}) {
// 	s, ok := obj.(*v1.Service)
// 	if !ok {
// 		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
// 		if !ok {
// 			glog.Errorf("Couldn't get object from tombstone %#v", obj)
// 			return
// 		}
// 		s, ok = tombstone.Obj.(*v1.Service)
// 		if !ok {
// 			glog.Errorf("Tombstone contained object that is not a Service %#v", obj)
// 			return
// 		}
// 	}

// 	// Verify that this is a Service owned by the cluster.
// 	c, err := cc.getClusterForObject(s)
// 	if err != nil {
// 		if errors.IsNotFound(err) {
// 			// MemcachedCluster has been deleted.
// 			return
// 		}

// 		if glog.V(9) {
// 			glog.Errorf("Error getting cluster for Service %#v: %#v", s, err)
// 		} else {
// 			glog.Errorf("Error getting cluster for Service %q: %s", s.Name, err)
// 		}
// 		return
// 	}

// 	if c == nil {
// 		// Service not owned by a MemcachedCluster
// 		return
// 	}

// 	logObject("Deleting Service: ", s)

// 	cc.enqueueCluster(c)
// }

// func (cc *Controller) syncTwemproxyService(c *v1alpha1.MemcachedCluster) error {
// 	s, err := cc.getTwemproxyServiceForCluster(c)
// 	switch {
// 	case errors.IsNotFound(err):
// 		glog.V(3).Infof("Twemproxy Service for MemcachedCluster %q not found. Creating.", c.Name)
// 		s, err = cc.createTwemproxyService(c)
// 		if err != nil {
// 			return fmt.Errorf("Error creating Service for MemcachedCluster %q: %v", c.Name, err)
// 		}
// 		logObject("Service created: ", s)

// 		return nil

// 	case err != nil:
// 		// Some other error occurred.
// 		return fmt.Errorf("Error getting Service for MemcachedCluster %q: %v", c.Name, err)
// 	}

// 	glog.V(9).Infof("Syncing Twemproxy Service %#v", s)

// 	// This Service is owned by the cluster.
// 	// TODO: Compare Service against desired Service to see if it's been modified. Revert if so.

// 	return nil
// }

// func (cc *Controller) createTwemproxyService(c *v1alpha1.MemcachedCluster) (*v1.Service, error) {
// 	name := serviceNameFromCluster(c)
// 	deployName := twemproxyNameFromCluster(c)

// 	service := &v1.Service{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:            name,
// 			Namespace:       c.Namespace,
// 			OwnerReferences: []metav1.OwnerReference{*newClusterOwnerRef(c)},
// 		},
// 		Spec: v1.ServiceSpec{
// 			Selector: map[string]string{
// 				// TODO: Use a better selector.
// 				"twemproxy-name": deployName,
// 			},
// 			Type: "ClusterIP",
// 			Ports: []v1.ServicePort{
// 				v1.ServicePort{
// 					Name:     "memcached",
// 					Protocol: "TCP",
// 					Port:     memcachedPort,
// 					TargetPort: intstr.IntOrString{
// 						IntVal: memcachedPort,
// 					},
// 				},
// 			},
// 		},
// 	}

// 	result, err := cc.client.CoreV1().Services(c.Namespace).Create(service)
// 	if err != nil {
// 		// Return the Service we attempted to created w/ the error
// 		return service, err
// 	}

// 	// Return the created Service
// 	return result, nil
// }

// func (cc *Controller) deleteTwemproxyService(key string) error {
// 	namespace, clusterName, err := cache.SplitMetaNamespaceKey(key)
// 	if err != nil {
// 		return err
// 	}

// 	name, err := serviceNameFromClusterKey(key)
// 	if err != nil {
// 		return err
// 	}

// 	s, err := cc.serviceLister.Services(namespace).Get(name)
// 	if err != nil {
// 		if errors.IsNotFound(err) {
// 			// If IsNotFound() then Service has already been deleted.
// 			return nil
// 		}

// 		// Another error occurred
// 		return err
// 	}

// 	// verify owner ref
// 	ref := getOwnerRefOfKind(s, v1alpha1.SchemeGroupVersion, "MemcachedCluster")
// 	if ref == nil || ref.Name != clusterName {
// 		// Log the error and return nil since this is not a temporary error.
// 		glog.Errorf("service %q ownerreference is %q; expected %q", name, ref.Name, clusterName)
// 		return nil
// 	}

// 	// Delete the Twemproxy Service.
// 	if err := cc.client.CoreV1().Services(namespace).Delete(name, nil); err != nil {
// 		return fmt.Errorf("failed to delete Twemproxy Service %q: %v", name, err)
// 	}
// 	glog.V(2).Infof("Service %q deleted", name)

// 	return nil
// }

// func serviceNameFromCluster(c *v1alpha1.MemcachedCluster) string {
// 	// TODO: Use hash of cluster name and collision count
// 	return makeName(fmt.Sprintf("%s-memcached", c.Name), []string{c.Name})
// }
// func serviceNameFromClusterKey(key string) (string, error) {
// 	_, name, err := cache.SplitMetaNamespaceKey(key)
// 	if err != nil {
// 		return "", err
// 	}
// 	return makeName(fmt.Sprintf("%s-memcached", name), []string{name}), nil
// }
