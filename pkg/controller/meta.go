package controller

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ianlewis/memcached-operator/pkg/apis/ianlewis.org/v1alpha1"
)

// NewProxyOwnerRef returns a new OwnerReference referencing the given proxy.
func NewProxyOwnerRef(c *v1alpha1.MemcachedProxy) *metav1.OwnerReference {
	controllerKind := v1alpha1.SchemeGroupVersion.WithKind("MemcachedProxy")
	// Set the owner reference so that dependant objects can be deleted by garbage collection.
	// Requires Kubernetes 1.8+ for support for CRDs
	// See: https://kubernetes.io/docs/concepts/workloads/controllers/garbage-collection/
	controller := true
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
