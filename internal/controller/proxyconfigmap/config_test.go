// Copyright 2018 Google LLC
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

package proxyconfigmap

import (
	"testing"

	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/uuid"

	"github.com/ianlewis/memcached-operator/internal/apis/ianlewis.org/v1alpha1"
)

// TestRouteForRule tests route creation for a MemcachedProxy rule spec
func TestRouteForRule(t *testing.T) {
	t.Run("sharded rule with service should be created correctly", func(t *testing.T) {
		f := newFixture(t, nil, nil, nil, nil)

		r := v1alpha1.RuleSpec{
			Type: v1alpha1.ShardedRuleType,
			Service: &v1alpha1.ServiceSpec{
				Name:      "foo",
				Namespace: "default",
				Port:      intstr.IntOrString{IntVal: 11211},
			},
		}

		services := make(map[string]*v1alpha1.ServiceSpec)
		route, _, err := f.controller.routeForRule(r, services)
		if !assert.NoError(t, err, "must be able to generate routes") {
			assert.FailNow(t, "failing early")
		}
		assert.Equal(t, route.StringVal, "PoolRoute|"+poolNameForServiceSpec(r.Service))

	})

	t.Run("sharded rule with children should be created correctly", func(t *testing.T) {
		f := newFixture(t, nil, nil, nil, nil)

		r := v1alpha1.RuleSpec{
			Type: v1alpha1.ShardedRuleType,
			Children: []v1alpha1.RuleSpec{
				{
					Type: v1alpha1.ShardedRuleType,
					Service: &v1alpha1.ServiceSpec{
						Name:      "foo",
						Namespace: "default",
						Port:      intstr.IntOrString{IntVal: 11211},
					},
				},
				{
					Type: v1alpha1.ShardedRuleType,
					Service: &v1alpha1.ServiceSpec{
						Name:      "bar",
						Namespace: "default",
						Port:      intstr.IntOrString{IntVal: 11211},
					},
				},
			},
		}

		services := make(map[string]*v1alpha1.ServiceSpec)
		route, _, err := f.controller.routeForRule(r, services)
		if !assert.NoError(t, err, "must be able to generate routes") {
			assert.FailNow(t, "failing early")
		}

		assert.Equal(t, route.Route.Type, "HashRoute")
		if !assert.Len(t, route.Route.Children, 2) {
			assert.FailNow(t, "failing early")
		}
		assert.Equal(t, route.Route.Children[0].StringVal, "PoolRoute|"+poolNameForServiceSpec(r.Children[0].Service))
		assert.Equal(t, route.Route.Children[1].StringVal, "PoolRoute|"+poolNameForServiceSpec(r.Children[1].Service))
	})

	t.Run("replicated rule with service should be created correctly", func(t *testing.T) {
		f := newFixture(t, nil, nil, nil, nil)

		r := v1alpha1.RuleSpec{
			Type: v1alpha1.ReplicatedRuleType,
			Service: &v1alpha1.ServiceSpec{
				Name:      "foo",
				Namespace: "default",
				Port:      intstr.IntOrString{IntVal: 11211},
			},
		}

		services := make(map[string]*v1alpha1.ServiceSpec)
		route, _, err := f.controller.routeForRule(r, services)
		if !assert.NoError(t, err, "must be able to generate routes") {
			assert.FailNow(t, "failing early")
		}

		assert.Equal(t, route.Route.Type, "OperationSelectorRoute")
		assert.Equal(t, route.Route.DefaultPolicy.StringVal, "AllSyncRoute|Pool|"+poolNameForServiceSpec(r.Service))
		assert.Equal(t, route.Route.OperationPolicies.Delete.StringVal, "LatestRoute|Pool|"+poolNameForServiceSpec(r.Service))
	})

	t.Run("replicated rule with children should be created correctly", func(t *testing.T) {
		f := newFixture(t, nil, nil, nil, nil)

		r := v1alpha1.RuleSpec{
			Type: v1alpha1.ReplicatedRuleType,
			Children: []v1alpha1.RuleSpec{
				{
					Type: v1alpha1.ShardedRuleType,
					Service: &v1alpha1.ServiceSpec{
						Name:      "foo",
						Namespace: "default",
						Port:      intstr.IntOrString{IntVal: 11211},
					},
				},
				{
					Type: v1alpha1.ShardedRuleType,
					Service: &v1alpha1.ServiceSpec{
						Name:      "bar",
						Namespace: "default",
						Port:      intstr.IntOrString{IntVal: 11211},
					},
				},
			},
		}

		services := make(map[string]*v1alpha1.ServiceSpec)
		route, _, err := f.controller.routeForRule(r, services)
		if !assert.NoError(t, err, "must be able to generate routes") {
			assert.FailNow(t, "failing early")
		}

		assert.Equal(t, route.Route.Type, "OperationSelectorRoute")
		assert.Equal(t, route.Route.DefaultPolicy.Route.Type, "AllSyncRoute")
		assert.Len(t, route.Route.DefaultPolicy.Route.Children, 2)

		assert.Equal(t, route.Route.OperationPolicies.Delete.Route.Type, "LatestRoute")
		assert.Len(t, route.Route.OperationPolicies.Delete.Route.Children, 2)
	})
}

// TestPoolForService tests the poolForService controller method
func TestPoolForService(t *testing.T) {
	t.Run("service with multiple ports should use correct mapped port", func(t *testing.T) {
		// Create the MemcachedProxy
		// The proxy references port 1000 on service "fuga".
		// That maps to port 11211 on the service so memcached-operator
		// should pick the endpoints for that port.
		p := &v1alpha1.MemcachedProxy{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1alpha1.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				UID:         uuid.NewUUID(),
				Name:        "hoge",
				Namespace:   metav1.NamespaceDefault,
				Annotations: make(map[string]string),
				Generation:  1,
			},
			Spec: v1alpha1.MemcachedProxySpec{
				Rules: v1alpha1.RuleSpec{
					Type: v1alpha1.ReplicatedRuleType,
					Service: &v1alpha1.ServiceSpec{
						Name: "fuga",
						Port: intstr.FromInt(1000),
					},
				},
			},
		}
		p.Status.ObservedGeneration = p.Generation

		p.ApplyDefaults()

		// Create the service "fuga". This service has two ports.
		// Port 1000 maps to 11211 on target pods.
		s := &corev1.Service{
			TypeMeta: metav1.TypeMeta{
				APIVersion: corev1.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				UID:         uuid.NewUUID(),
				Name:        "fuga",
				Namespace:   metav1.NamespaceDefault,
				Annotations: make(map[string]string),
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{
					"app": "memcached",
				},
				Type: "ClusterIP",
				Ports: []corev1.ServicePort{
					{
						Name:       "memcached",
						Protocol:   "TCP",
						Port:       1000,
						TargetPort: intstr.FromInt(11211),
					},
					{
						Name:       "other",
						Protocol:   "TCP",
						Port:       8080,
						TargetPort: intstr.FromInt(8081),
					},
				},
			},
		}

		ep := &corev1.Endpoints{
			TypeMeta: metav1.TypeMeta{
				APIVersion: corev1.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				UID:         uuid.NewUUID(),
				Name:        "fuga",
				Namespace:   metav1.NamespaceDefault,
				Annotations: make(map[string]string),
			},
			Subsets: []corev1.EndpointSubset{
				{
					Addresses: []corev1.EndpointAddress{
						{
							IP: "123.123.123.123",
							// NOTE: omitting targetref & nodename
						},
						{
							IP: "123.123.123.124",
							// NOTE: omitting targetref & nodename
						},
					},
					Ports: []corev1.EndpointPort{
						{
							Name:     "memcached",
							Protocol: "TCP",
							Port:     11211,
						},
						{
							Name:     "other",
							Protocol: "TCP",
							Port:     8081,
						},
					},
				},
			},
		}

		f := newFixture(
			t,
			[]*v1alpha1.MemcachedProxy{p},
			nil,
			[]*corev1.Service{s},
			[]*corev1.Endpoints{ep},
		)

		pool, err := f.controller.poolForService(p.Spec.Rules.Service)
		if !assert.NoError(t, err, "must be able to generate pool") {
			assert.FailNow(t, "failing early")
		}

		// Check that the right target port was chosen.
		assert.Equal(t, []string{"123.123.123.123:11211", "123.123.123.124:11211"}, pool.Servers)
	})
}
