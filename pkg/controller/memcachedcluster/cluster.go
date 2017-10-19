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
	"reflect"

	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"

	"github.com/IanLewis/memcached-operator/pkg/api/ianlewis.org/v1alpha1"
	"github.com/IanLewis/memcached-operator/pkg/controller"
)

func (cc *Controller) addMemcachedCluster(obj interface{}) {
	cluster := obj.(*v1alpha1.MemcachedCluster)
	logObject("Adding cluster: ", cluster)

	cc.enqueueCluster(cluster)
}

func (cc *Controller) updateMemcachedCluster(old, cur interface{}) {
	oldC := old.(*v1alpha1.MemcachedCluster)
	curC := cur.(*v1alpha1.MemcachedCluster)
	logObject("Updating cluster: ", oldC, curC)

	cc.enqueueCluster(curC)
}

func (cc *Controller) deleteMemcachedCluster(obj interface{}) {
	cluster, ok := obj.(*v1alpha1.MemcachedCluster)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			glog.Errorf("Couldn't get object from tombstone %#v", obj)
			return
		}
		cluster, ok = tombstone.Obj.(*v1alpha1.MemcachedCluster)
		if !ok {
			glog.Errorf("Tombstone contained object that is not a MemcachedCluster %#v", obj)
			return
		}
	}
	logObject("Deleting cluster: ", cluster)

	cc.enqueueCluster(cluster)
}

func newClusterOwnerRef(c *v1alpha1.MemcachedCluster) *metav1.OwnerReference {
	// Set the owner reference so that dependant objects can be deleted by garbage collection.
	// See: https://kubernetes.io/docs/concepts/workloads/controllers/garbage-collection/
	controller := true
	// NOTE: As of 1.7, Garbage Collection is not supported for ThirdPartyResources or CustomResourceDefinitions.
	blockOwnerDeletion := true
	return &metav1.OwnerReference{
		APIVersion:         controllerKind.GroupVersion().String(),
		Kind:               controllerKind.Kind,
		Name:               c.Name,
		UID:                c.UID,
		Controller:         &controller,
		BlockOwnerDeletion: &blockOwnerDeletion,
	}
}

// teardownCluster cleans up
func (cc *Controller) teardownCluster(key string) error {
	glog.V(2).Infof("Tearing down MemcachedCluster key %q", key)

	if err := cc.deleteTwemproxyDeployment(key); err != nil {
		return fmt.Errorf("failed to delete Twemproxy Deployment for key %q: %v", key, err)
	}

	if err := cc.deleteMemcachedDeployment(key); err != nil {
		return fmt.Errorf("failed to delete Memcached Deployment for key %q: %v", key, err)
	}

	if err := cc.deleteTwemproxyConfigMaps(key); err != nil {
		return fmt.Errorf("failed to delete Twemproxy ConfigMaps for key %q: %v", key, err)
	}

	return nil
}

func (cc *Controller) calculateClusterStatus(c *v1alpha1.MemcachedCluster) (v1alpha1.MemcachedClusterStatus, error) {
	status := v1alpha1.MemcachedClusterStatus{
		ObservedGeneration: c.Generation,
	}

	s, err := controller.GetProxyServiceForCluster(cc.serviceLister, c)
	if err != nil {
		if !errors.IsNotFound(err) {
			return status, err
		}
	} else {
		status.ServiceName = s.Name
		status.ServiceIP = s.Spec.ClusterIP
	}

	// TODO: Efficiency improvements
	d, err := cc.getMemcachedDeploymentForCluster(c)
	if err != nil {
		return status, err
	}

	podIPs, err := cc.getMemcachedPodIPs(d)
	if err != nil {
		return status, err
	}

	status.Replicas = int32(len(podIPs))

	hash, err := controller.HashMemcachedClusterSpec(c.Spec)
	if err != nil {
		return status, err
	}
	status.ObservedSpecHash = hash

	return status, nil
}

func (cc *Controller) syncMemcachedCluster(c *v1alpha1.MemcachedCluster) error {
	err := cc.cClient.Put().
		Namespace(c.Namespace).
		Resource("memcachedclusters").
		Name(c.Name).
		// SubResource("status").
		Body(c).
		Do().
		Error()
	if err != nil {
		return fmt.Errorf("failed to update cluster %q: %v", c.Name, err)
	}

	return nil
}

func (cc *Controller) syncMemcachedClusterStatus(c *v1alpha1.MemcachedCluster) error {
	newStatus, err := cc.calculateClusterStatus(c)
	if err != nil {
		return fmt.Errorf("failed to calculate status for cluster %q: %v", c.Name, err)
	}

	if reflect.DeepEqual(c.Status, newStatus) {
		return nil
	}

	c.Status = newStatus

	// TODO: Create MemcachedClusters client with UpdateStatus
	return cc.syncMemcachedCluster(c)
}

func (cc *Controller) getTwemproxyDeploymentForCluster(c *v1alpha1.MemcachedCluster) (*v1beta1.Deployment, error) {
	name := twemproxyNameFromCluster(c)

	d, err := cc.deploymentLister.Deployments(c.Namespace).Get(name)
	if err != nil {
		return nil, err
	}

	// verify owner ref.
	ref := getOwnerRefOfKind(d, v1alpha1.SchemeGroupVersion, "MemcachedCluster")
	if ref == nil || ref.Name != c.Name {
		return nil, nil
	}

	return d, nil
}

func (cc *Controller) getMemcachedDeploymentForCluster(c *v1alpha1.MemcachedCluster) (*v1beta1.Deployment, error) {
	name := memcachedNameFromCluster(c)

	d, err := cc.deploymentLister.Deployments(c.Namespace).Get(name)
	if err != nil {
		return nil, err
	}

	// verify owner ref
	ref := getOwnerRefOfKind(d, v1alpha1.SchemeGroupVersion, "MemcachedCluster")
	if ref == nil || ref.Name != c.Name {
		return nil, nil
	}

	return d, nil
}

func (cc *Controller) getTwemproxyServiceForCluster(c *v1alpha1.MemcachedCluster) (*v1.Service, error) {
	name := controller.GetProxyServiceName(c)

	s, err := cc.serviceLister.Services(c.Namespace).Get(name)
	if err != nil {
		return nil, err
	}

	// verify owner ref
	ref := getOwnerRefOfKind(s, v1alpha1.SchemeGroupVersion, "MemcachedCluster")
	if ref == nil || ref.Name != c.Name {
		return nil, nil
	}

	return s, nil
}

func (cc *Controller) getClusterForObject(obj metav1.Object) (*v1alpha1.MemcachedCluster, error) {
	// Verify that this is an object owned by the cluster.
	ref := getOwnerRefOfKind(obj, v1alpha1.SchemeGroupVersion, "MemcachedCluster")
	if ref != nil {
		c, err := cc.clusterLister.MemcachedClusters(obj.GetNamespace()).Get(ref.Name)
		if err != nil {
			return nil, err
		}

		if ref.UID != c.UID {
			// TODO: Adopt object and update UID of OwnerReference
			glog.Errorf("object %#v ownerreference UID is %q; expected %q", obj, ref.UID, c.UID)
		}

		// This object is owned by the cluster.
		return c, nil
	}

	// Indicates that the object is not owned by a MemcachedCluster
	return nil, nil
}

func (cc *Controller) getDeploymentForObject(obj metav1.Object) (*v1beta1.Deployment, error) {
	// Verify that this is an object owned by the cluster.
	ref := getOwnerRefOfKind(obj, v1beta1.SchemeGroupVersion, "Deployment")
	if ref != nil {
		d, err := cc.deploymentLister.Deployments(obj.GetNamespace()).Get(ref.Name)
		if err != nil {
			return nil, err
		}

		if ref.UID == d.ObjectMeta.UID {
			// This object is owned by the Deployment.
			return d, nil
		}
	}

	// Indicates that the object is not owned by a MemcachedCluster
	return nil, nil
}

func (cc *Controller) getClusterForPod(pod *v1.Pod) (*v1alpha1.MemcachedCluster, error) {
	d, err := cc.getDeploymentForObject(pod)
	if err != nil {
		if errors.IsNotFound(err) {
			// Deployment has been deleted.
			return nil, nil
		}

		return nil, err
	}

	if d == nil {
		// Deployment not owned by a MemcachedCluster
		return nil, nil
	}

	// Verify that this is a deployment owned by the cluster.
	c, err := cc.getClusterForObject(d)
	if err != nil {
		return nil, err
	}

	if c == nil {
		// Deployment not owned by a MemcachedCluster
		return nil, nil
	}

	return c, nil
}
