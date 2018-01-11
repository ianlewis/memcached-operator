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
