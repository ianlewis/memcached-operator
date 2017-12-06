# Combined Pools

The memcached operator allows you to define combinations of [sharded](sharded-pools.md) and [replicated](replicated-pools.md) pools.

When a cache data set becomes big enough and you want to maintain high availability, you may want to replicate data to multiple failure zones. Once you have replicated to multiple failure zones you may want to shard the data within each failure zone. This allows you to take advantage of the high availability of replicated pools, with the scalability of sharded pools.

Here is an example of a `MemcachedProxy` where the keys are replicated to two separate pools of servers and then sharded within that pool. So in this example, the key is replicated to two servers where each are members of separate pools.

[embedmd]:# (combined-example.yaml yaml /apiVersion/ $)
```yaml
apiVersion: ianlewis.org/v1alpha1
kind: MemcachedProxy
metadata:
  name: example
spec:
  rules:
    type: "replicated"
    children:
      - type: "sharded"
        service:
          name: "sharded1-memcached"
          port: 11211
      - type: "sharded"
        service:
          name: "sharded2-memcached"
          port: 11211
```
