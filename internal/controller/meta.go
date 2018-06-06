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

package controller

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ianlewis/memcached-operator/internal/apis/ianlewis.org/v1alpha1"
)

// NewProxyOwnerRef returns a new OwnerReference referencing the given proxy.
func NewProxyOwnerRef(c *v1alpha1.MemcachedCluster) *metav1.OwnerReference {
	controllerKind := v1alpha1.SchemeGroupVersion.WithKind("MemcachedCluster")
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
