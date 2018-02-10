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

package v1alpha1

import (
	"strconv"

	"github.com/mitchellh/hashstructure"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	ReplicatedRuleType = "replicated"
	ShardedRuleType    = "sharded"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MemcachedProxy enables creating a managed memcached cluster.
type MemcachedProxy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MemcachedProxySpec   `json:"spec"`
	Status MemcachedProxyStatus `json:"status"`
}

func (p *MemcachedProxy) ApplyDefaults() {
	p.Spec.ApplyDefaults(p)
}

// MemcachedProxySpec is the specification of the desired state of a MemcachedProxy.
type MemcachedProxySpec struct {
	Rules    RuleSpec     `json:"rules"`
	McRouter McRouterSpec `json:"mcrouter"`
}

func (s *MemcachedProxySpec) ApplyDefaults(p *MemcachedProxy) {
	s.McRouter.ApplyDefaults(p)
	s.Rules.ApplyDefaults(p)
}

// UpdateHash updates the
func (s *MemcachedProxySpec) GetHash() (string, error) {
	hash, err := hashstructure.Hash(s, nil)
	if err != nil {
		return "", err
	}
	// Return hex format.
	return strconv.FormatUint(hash, 16), nil
}

type McRouterSpec struct {
	Image           string                      `json:"image,omitempty"`
	Resources       corev1.ResourceRequirements `json:"resources,omitempty"`
	Port            *int32                      `json:"port,omitempty"`
	SecurityContext *corev1.SecurityContext     `json:"securityContext,omitempty"`
	// StatsRoot is the directory for storing stats files (--stats-root)
	StatsRoot string `json:"statsRoot,omitempty"`
}

func (s *McRouterSpec) ApplyDefaults(p *MemcachedProxy) {
	if s.Image == "" {
		s.Image = "ianmlewis/mcrouter:v0.36.0-0"
	}
	if s.Port == nil {
		port := int32(11211)
		s.Port = &port
	}
	if s.StatsRoot == "" {
		s.StatsRoot = "/var/mcrouter/stats"
	}
	if s.SecurityContext == nil {
		nonRoot := true
		readOnly := true
		s.SecurityContext = &corev1.SecurityContext{
			// Verify that the container runs as a non-root user but defer to
			// the image metadata to determine which user to actually run as.
			RunAsNonRoot:           &nonRoot,
			ReadOnlyRootFilesystem: &readOnly,
		}
	}
}

// RuleSpec defines a routing rule to either a list of services or child rules
type RuleSpec struct {
	Type     string       `json:"type"`
	Service  *ServiceSpec `json:"service,omitempty"`
	Children []RuleSpec   `json:"children,omitempty"`
}

func (r *RuleSpec) ApplyDefaults(p *MemcachedProxy) {
	if r.Type == "" {
		r.Type = ShardedRuleType
	}
	if r.Service != nil {
		r.Service.ApplyDefaults(p)
	}
	for _, r := range r.Children {
		r.ApplyDefaults(p)
	}
}

type ServiceSpec struct {
	Name      string             `json:"name"`
	Namespace string             `json:"namespace"`
	Port      intstr.IntOrString `json:"port"`
}

func (s *ServiceSpec) ApplyDefaults(p *MemcachedProxy) {
	if s.Namespace == "" {
		s.Namespace = p.Namespace
	}
}

// MemcachedProxyStatus is the most recently observed status of the cluster
type MemcachedProxyStatus struct {
	// The generation observed by the MemcachedProxy controller. Not used currently as generation is not updated for CRDs.
	// Proper support for CRDs is being worked on.
	// See: https://github.com/kubernetes/community/pull/913
	// TODO: implement ObservedGeneration
	// ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// This is a workaround for the fact that Generation and sub-resources are not fully supported for CRDs yet.
	// We assume that end users will not update the status object and especially this field.
	ObservedSpecHash string `json:"observedSpecHash,omitempty"`
	// TODO: updated replicas in status?
	// Replicas int32 `json:"replicas,omitempty"`

	// TODO: Determine other status fields (ready? stats from mcrouter?)
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MemcachedProxyList is a list of MemcachedProxy objects.
type MemcachedProxyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []MemcachedProxy `json:"items"`
}
