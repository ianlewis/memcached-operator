package proxyconfigmap

import (
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/ianlewis/memcached-operator/pkg/apis/ianlewis.org/v1alpha1"
	"github.com/ianlewis/memcached-operator/pkg/controller"
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

// Route describes an mcrouter route. It can be either an object or a string.
type Route struct {
	Type              string             `json:"type"`
	DefaultPolicy     *Route             `json:"default_policy,omitempty"`
	OperationPolicies *OperationPolicies `json:"operation_policies,omitempty"`
	Children          []Route            `json:"children,omitempty"`
	StringVal         string
}

// MarshalJSON implements the json.Marshaller interface.
func (r Route) MarshalJSON() ([]byte, error) {
	if r.StringVal != "" {
		return json.Marshal(r.StringVal)
	} else {
		return json.Marshal(r)
	}
}

// McRouterConfig represents JSON configuration for mcrouter.
type McRouterConfig struct {
	Pools  []Pool  `json:"pools,omitempty"`
	Routes []Route `json:"routes,omitempty"`
}

func (c *Controller) configForProxy(p *v1alpha1.MemcachedProxy) (*McRouterConfig, error) {
	config := &McRouterConfig{}
	// A map of pool name to ServiceSpec
	services := make(map[string]*v1alpha1.ServiceSpec)

	// Parse the proxy's rules and create the mcrouter routes
	// A list of services is also built up when parsing the rules
	for _, r := range p.Spec.Rules {
		route, _, err := c.routeForRule(r, services)
		if err != nil {
			return nil, err
		}

		config.Routes = append(config.Routes, route)
	}

	// Create a memcached server pool for each service found
	for _, s := range services {
		pool, err := c.poolForService(s)
		if err != nil {
			return nil, err
		}
		config.Pools = append(config.Pools, pool)
	}

	return config, nil
}

// poolNameForService generates a unique pool name for each service and port combination
func poolNameForServiceSpec(s *v1alpha1.ServiceSpec) string {
	return controller.MakeName("", []string{s.Name, s.Port.String()})
}

// poolForService creates a mcroute pool for the given service spec.
func (c *Controller) poolForService(s *v1alpha1.ServiceSpec) (Pool, error) {
	pool := Pool{}

	// Get the service
	svc, err := c.sLister.Services(s.Namespace).Get(s.Name)
	if err != nil {
		return pool, fmt.Errorf("failed to get service %q: %v", s.Namespace+"/"+s.Name, err)
	}

	// Find the service port. We need to be able to resolve the service port by name.
	var port *corev1.ServicePort
	for _, p := range svc.Spec.Ports {
		np := p
		switch s.Port.Type {
		case intstr.Int:
			if p.Port == s.Port.IntVal {
				port = &np
			}
		default:
			if p.Name == s.Port.StrVal {
				port = &np
			}
		}
	}

	if port == nil {
		return pool, fmt.Errorf("failed to find port %q for service %q", s.Port.String(), s.Namespace+"/"+s.Name)
	}

	// Get the endpoints for the service
	ep, err := c.epLister.Endpoints(svc.Namespace).Get(svc.Name)
	if err != nil {
		return pool, fmt.Errorf("failed to get endpoints %q: %v", svc.Namespace+"/"+svc.Name, err)
	}

	// Build the list of servers from the endpoints
	// We need to find the right port in the endpoints in order to know what endpoint subset
	// to build the addresses from. Here we match the endpoint subset's ports with the service's port
	// specified in the proxy object.
	for _, subset := range ep.Subsets {
		for _, eport := range subset.Ports {
			if portMatches(eport, port.TargetPort) {
				for _, addr := range subset.Addresses {
					pool.Servers = append(pool.Servers, addr.IP+":"+string(port.Port))
				}
			}
		}
	}

	return pool, nil
}

// portMatches returns true if the endpoint port is a TCP port and matches the given service target port
func portMatches(eport corev1.EndpointPort, targetPort intstr.IntOrString) bool {
	switch targetPort.Type {
	case intstr.Int:
		return eport.Protocol == corev1.ProtocolTCP && eport.Port == targetPort.IntVal
	default:
		return eport.Protocol == corev1.ProtocolTCP && eport.Name == targetPort.StrVal
	}
}

// routeForRule creates a mcrouter route from a memcached proxy rule
func (c *Controller) routeForRule(r v1alpha1.RuleSpec, services map[string]*v1alpha1.ServiceSpec) (Route, map[string]*v1alpha1.ServiceSpec, error) {
	if r.Type == v1alpha1.ReplicatedRuleType {
		// Create a replicated route
		// See: https://github.com/facebook/mcrouter/wiki/Replicated-pools-setup
		var route Route
		if r.Service != nil {
			// Replicate among a group of servers in a pool
			poolName := poolNameForServiceSpec(r.Service)
			services[poolName] = r.Service
			route = Route{
				Type: "OperationSelectorRoute",
				DefaultPolicy: &Route{
					StringVal: "AllSyncRoute|Pool|" + poolName,
				},
				OperationPolicies: &OperationPolicies{
					Delete: &Route{
						StringVal: "LatestRoute|Pool|" + poolName,
					},
				},
			}
		} else {
			// Replicate among a group of child routes
			var children []Route
			for _, route := range r.Children {
				child, _, err := c.routeForRule(route, services)
				if err != nil {
					return Route{}, services, err
				}
				children = append(children, child)
			}

			route = Route{
				Type: "OperationSelectorRoute",
				DefaultPolicy: &Route{
					Type:     "AllSyncRoute",
					Children: children,
				},
				OperationPolicies: &OperationPolicies{
					Delete: &Route{
						Type:     "LatestRoute",
						Children: children,
					},
				},
			}

		}

		return route, services, nil
	} else if r.Type == v1alpha1.ShardedRuleType {
		// Create a sharded rule
		// See: https://github.com/facebook/mcrouter/wiki/Sharded-pools-setup
		var route Route
		if r.Service != nil {
			// Shard among a group of servers in a pool
			poolName := poolNameForServiceSpec(r.Service)
			services[poolName] = r.Service
			route = Route{
				StringVal: "PoolRoute|" + poolName,
			}
		} else {
			// Shard among a group of child routes
			var children []Route
			for _, route := range r.Children {
				child, _, err := c.routeForRule(route, services)
				if err != nil {
					return Route{}, services, err
				}
				children = append(children, child)
			}
			route = Route{
				Type:     "HashRoute",
				Children: children,
			}
		}

		return route, services, nil
	}

	return Route{}, services, fmt.Errorf("unknown rule type: %q", r.Type)
}
