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

package test

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/uuid"

	"github.com/ianlewis/memcached-operator/internal/apis/ianlewis.org/v1alpha1"
)

func NewShardedProxy(name string, service string) *v1alpha1.MemcachedCluster {
	return NewMemcachedCluster(name, v1alpha1.RuleSpec{
		Type: v1alpha1.ShardedRuleType,
		Service: &v1alpha1.ServiceSpec{
			Name: service,
			Port: intstr.FromInt(11211),
		},
	})
}

func NewReplicatedProxy(name string, service string) *v1alpha1.MemcachedCluster {
	return NewMemcachedCluster(name, v1alpha1.RuleSpec{
		Type: v1alpha1.ReplicatedRuleType,
		Service: &v1alpha1.ServiceSpec{
			Name: service,
			Port: intstr.FromInt(11211),
		},
	})
}

func NewMemcachedCluster(name string, rules v1alpha1.RuleSpec) *v1alpha1.MemcachedProxy {
	p := &v1alpha1.MemcachedCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			UID:         uuid.NewUUID(),
			Name:        name,
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
			Generation:  1,
		},
		Spec: v1alpha1.MemcachedClusterSpec{
			Rules: rules,
		},
	}
	p.Status.ObservedGeneration = p.Generation

	p.ApplyDefaults()

	return p
}
