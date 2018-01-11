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

	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/ianlewis/memcached-operator/pkg/apis/ianlewis.org/v1alpha1"
)

// TestRouteForRule tests route creation for a MemcachedProxy rule spec
func TestRouteForRule(t *testing.T) {
	f := newFixture(t, nil, nil, nil, nil)

	t.Run("sharded rule with service should be created correctly", func(t *testing.T) {
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
