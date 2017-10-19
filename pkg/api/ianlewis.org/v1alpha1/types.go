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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MemcachedCluster enables creating a managed memcached cluster.
type MemcachedCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MemcachedClusterSpec   `json:"spec,omitempty"`
	Status MemcachedClusterStatus `json:"status,omitempty"`
}

// MemcachedClusterSpec is the specification of the desired state of a MemcachedCluster.
type MemcachedClusterSpec struct {
	Replicas       *int32 `json:"replicas,omitempty"`
	MemcachedImage string `json:"memcachedImage,omitempty"`
	// TODO: Support setting the service name
	// ServiceName string `json:"serviceName"`
	Proxy *MemcachedProxySpec `json:"proxy,omitempty"`
}

// MemcachedProxySpec is the specification for Memcached proxy used.
type MemcachedProxySpec struct {
	TwemproxyProxySpec *TwemproxySpec `json:"twemproxy,omitempty"`
}

// TwemproxySpec is the specification for a Twemproxy proxy
type TwemproxySpec struct {
	Image              string `json:"image,omitempty"`
	Hash               string `json:"hash,omitempty"`
	Distribution       string `json:"distribution,omitempty"`
	ServerRetryTimeout *int64 `json:"serverRetryTimeout,omitempty"`
	ServerFailureLimit *int32 `json:"serverFailureLimit,omitempty"`
}

// MemcachedClusterStatus is the most recently observed status of the cluster
type MemcachedClusterStatus struct {
	// The generation observed by the MemcachedCluster controller. Not used currently as generation is not updated for TPRs.
	// Proper support for CRDs is being worked on.
	// See: https://github.com/kubernetes/community/pull/913
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// This is a workaround for the fact that Generation and sub-resources are not fully supported for TPRs and CRDs yet.
	// We assume that end users will not update the status object and especially this field.
	ObservedSpecHash string `json:"observedSpecHash,omitempty"`
	Replicas         int32  `json:"replicas,omitempty"`
	ServiceName      string `json:"serviceName,omitempty"`
	ServiceIP        string `json:"serviceIP,omitempty"`
}

// GetObjectMeta returns the object's metadata. Required to satisfy ObjectMetaAccessor interface
func (m *MemcachedCluster) GetObjectMeta() metav1.Object {
	return &m.ObjectMeta
}

// MemcachedClusterList is a list of MemcachedClusters
type MemcachedClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []MemcachedCluster `json:"items"`
}

// GetListMeta returns the list's metadata. Required to satisfy ListMetaAccessor interface
func (m *MemcachedClusterList) GetListMeta() metav1.List {
	return &m.ListMeta
}
