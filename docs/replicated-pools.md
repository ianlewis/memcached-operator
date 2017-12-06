# Replicated Pools

The memcached operator supports creating a memcached proxy that replicates keys among a group of memcached instances.

When memcached instances become unavailable this could have an impact on the performance of an application. In order to support a more highly available setup, you will want to create multiple memcached instances that have a full copy of cached data. Each key is saved to every member of the replicated pool.

The memcached operator determines the servers in the pool by using a Kubernetes service. Each memcached pod that is a member of the service is treated as a member of the memcached pool.

Here is an example of a `MemcachedProxy` where keys are replicated among a group of memcached instances that are members of the "replicated-memcached" service.

[embedmd]:# (replicated-example.yaml yaml /apiVersion/ $)
