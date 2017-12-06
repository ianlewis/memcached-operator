# Sharded Pools

The memcached operator allows you to create a memcached proxy which shards keys among a group of memcached instances.

Once a dataset becomes to big for a single memcached instance, you will want to split it among multiple servers. The memcached operator can configure mcrouter to using a [sharded pool](https://github.com/facebook/mcrouter/wiki/Sharded-pools-setup). Each key saved to memcached is hashed and that hash is used to determine what server to save the data to. Keys will be evently distributed among the servers in the pool.

The memcached operator determines the servers in the pool by using a Kubernetes service. Each memcached pod that is a member of the service is treated as a member of the memcached pool.

Here is an example of a `MemcachedProxy` where keys are sharded among a group of memcached instances that are members of the "sharded-memcached" service.

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
