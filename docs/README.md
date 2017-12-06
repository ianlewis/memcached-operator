# Memcached Operator Documentation

memcached-operator is a Kubernetes [Operator](https://coreos.com/blog/introducing-operators.html) for deploying and managing a cluster of [Memcached](https://memcached.org/) instances.

memcached-operator provides a single Service endpoint that memcached client
applications can connect to to make use of the memcached cluster. It provides
this via a memcached proxy which is automatically updated whenever memcached
instances are added or removed from the cluster.

- [Architecture](design.md)
- [Sharded pools](sharded-pools.md)
- [Replicated pools](replicated-pools.md)
- [Combined Sharded+Replicated pools](combined-pools.md)
