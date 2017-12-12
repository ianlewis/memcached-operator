# The MemcachedProxy Custom Resource

The `MemcachedProxy` custom resource is used to define the configuration used
to create a mcrouter memcached proxy instance. This document is a reference for
the format of the `MemcachedProxy` resource.

The overall format of the proxy object consists of generic Kubernetes fields like the API version, kind, and metadata, as well as the specification for the proxy.

[embedmd]:# (sharded-example.yaml yaml /apiVersion/ $)
```yaml
apiVersion: ianlewis.org/v1alpha1
kind: MemcachedProxy
metadata:
  name: sharded-example
spec:
  rules:
    type: "sharded"
    service:
      name: "sharded-memcached"
      port: 11211
```

# The Memcached Proxy Specification

The `MemcachedProxySpec` defines the specification for the proxy. It is made up
of mcrouter specific configuration the and a root routing rule.

Here is the definition for the proxy specification in Go:

[embedmd]:# (../pkg/apis/ianlewis.org/v1alpha1/types.go go /type MemcachedProxySpec struct/ /}/)
```go
type MemcachedProxySpec struct {
	Rules    RuleSpec     `json:"rules"`
	McRouter McRouterSpec `json:"mcrouter"`
}
```

[//]: # (TODO: Doc for the mcrouter config specifcation)

# Routing Rules

Routing rules form an arbitrary tree where the root nodes are rules that reference Kubernetes services. Each routing rule has a `type` and can either have a `service` or `children`.

* `type` can be either "sharded" or "replicated". The default is "sharded". 
  * "sharded" indicates that requests are sharded among member pods in the referenced service, or among each child routing rule. See the [sharded pools documentation](sharded-pools.md) for more details.
  * "replicated" indicates that write requests are replicated to each member pod in the referenced service, or to each child routing rule. Get requests are routed to a random pod or child route based on host ID of the mcrouter instance. See the [replicated pools documentation](replicated-pools.md) for more details.
* `service` indicates that requests to this route should be routed to member pods of the service.
* `children` indicates that requests to this route should be routed tothe given child routes.

Here is the definition for the routing rule in Go:

[embedmd]:# (../pkg/apis/ianlewis.org/v1alpha1/types.go go /type RuleSpec struct/ /}/)
```go
type RuleSpec struct {
	Type     string       `json:"type"`
	Service  *ServiceSpec `json:"service,omitempty"`
	Children []RuleSpec   `json:"children,omitempty"`
}
```

[//]: # (TODO: Documentation for the MemcachedProxy status object)
