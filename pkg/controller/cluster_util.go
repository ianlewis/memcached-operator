package controller

import (
	"fmt"
	"strconv"

	"github.com/golang/glog"

	"github.com/mitchellh/hashstructure"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1_listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/pkg/api/v1"

	"github.com/IanLewis/memcached-operator/pkg/api/ianlewis.org/v1alpha1"
	v1alpha1_listers "github.com/IanLewis/memcached-operator/pkg/listers/ianlewis.org/v1alpha1"
)

func HashMemcachedClusterSpec(s v1alpha1.MemcachedClusterSpec) (string, error) {
	hash, err := hashstructure.Hash(s, nil)
	if err != nil {
		return "", err
	}
	// Return hex format.
	return strconv.FormatUint(hash, 16), nil
}

// IsCurrent returns true if the current MemcachedCluster has been observed by the MemcachedCluster controller.
func IsCurrent(c *v1alpha1.MemcachedCluster) bool {
	// ObservedSpecHash is a workaround until ObservedGeneration is supported properly in CRDs
	if c.Status.ObservedSpecHash == "" {
		return false
	}
	hash, err := HashMemcachedClusterSpec(c.Spec)
	if err != nil {
		return false
	}
	return hash == c.Status.ObservedSpecHash
}

// GetProxyServiceName returns the name of the proxy service for the given cluster
func GetProxyServiceName(c *v1alpha1.MemcachedCluster) string {
	// TODO: Use hash of cluster name and collision count
	return MakeName(fmt.Sprintf("%s-memcached", c.Name), []string{c.Name})
}

// GetTwemproxyDeploymentName returns the name of the twemproxy deployment for the given cluster.
func GetTwemproxyDeploymentName(c *v1alpha1.MemcachedCluster) string {
	// TODO: Use hash of cluster name and collision count
	return MakeName(fmt.Sprintf("%s-twemproxy", c.Name), []string{c.Name})
}

// GetProxyServiceForCluster returns the Service object for the MemcachedCluster
func GetProxyServiceForCluster(serviceLister v1_listers.ServiceLister, c *v1alpha1.MemcachedCluster) (*v1.Service, error) {
	name := GetProxyServiceName(c)

	s, err := serviceLister.Services(c.Namespace).Get(name)
	if err != nil {
		return nil, err
	}

	// verify owner ref
	ref := GetOwnerRefOfKind(s, v1alpha1.SchemeGroupVersion, "MemcachedCluster")
	if ref == nil || ref.Name != c.Name {
		return nil, nil
	}

	return s, nil
}

func GetClusterForObject(clusterLister v1alpha1_listers.MemcachedClusterLister, obj metav1.Object) (*v1alpha1.MemcachedCluster, error) {
	// Verify that this is an object owned by the cluster.
	ref := GetOwnerRefOfKind(obj, v1alpha1.SchemeGroupVersion, "MemcachedCluster")
	if ref != nil {
		c, err := clusterLister.MemcachedClusters(obj.GetNamespace()).Get(ref.Name)
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

// NewClusterOwnerRef returns a new OwnerReference referencing the given cluster.
func NewClusterOwnerRef(c *v1alpha1.MemcachedCluster) *metav1.OwnerReference {
	controllerKind := v1alpha1.SchemeGroupVersion.WithKind("MemcachedCluster")

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
