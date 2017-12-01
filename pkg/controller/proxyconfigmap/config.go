package proxyconfigmap

import (
	"github.com/ianlewis/memcached-operator/pkg/apis/ianlewis.org/v1alpha1"
)

// Pool is a memcached server pool
type Pool struct {
	Servers []string `json:"servers"`
}

// OperationPolicies configures operation policies for different actions via mcrouter
type OperationPolicies struct {
	Add    *Route `json:"add,omitempty"`
	Set    *Route `json:"set,omitempty"`
	Delete *Route `json:"delete,omitempty"`
	Get    *Route `json:"empty,omitempty"`
}

// Route describes an mcrouter route
type Route struct {
	Type              string             `json:"type"`
	Pool              string             `json:"pool,omitempty"`
	DefaultPolicy     string             `json:"default_policy,omitempty"`
	OperationPolicies *OperationPolicies `json:"operation_policies,omitempty"`
	Children          []*Route           `json:"children,omitempty"`
}

// McRouterConfig represents JSON configuration for mcrouter.
type McRouterConfig struct {
	Pools  []*Pool  `json:"pools,omitempty"`
	Route  *Route   `json:"route,omitempty"`
	Routes []*Route `json:"routes,omitempty"`
}

func (c *Controller) configForProxy(p *v1alpha1.MemcachedProxy) *McRouterConfig {
	return &McRouterConfig{}
}
