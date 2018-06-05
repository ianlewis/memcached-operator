# Memcached Operator Design

This document describes the architecture & design of the memcached operator.

## Overview

The memcached operator makes it easy to stand-up, manage, and utilize a memcached cluster in a Kubernetes cluster. It allows users to access a cluster via a single Kubernetes `Service` endpoint that supports sharding memcached keys using a proxy server.

## Background

Operating a memcached cluster in Kubernetes has a number of difficulties:

1. Memcached is difficult to use in a Kubernetes cluster because it does not match the Kubernetes [Service](https://kubernetes.io/docs/concepts/services-networking/service/) model closely. 
2. Memcached is often deployed with cache keys replicated to multiple failure domains. It is difficult to setup and maintain clusters this way.

Kubernetes provides a built in mechanism for service discovery and rudamentary load balancing using [Services](https://kubernetes.io/docs/concepts/services-networking/service/). Unfortunately, this is not ideal for services such as memcached. A typical memcached cluster shardes cache data among servers and uses consistent hashing of the cache key determine the server to connect to. Kubernetes services operate on TCP/UDP connections and does not have this level of knowledge of the memcached protocol.

Users can solve this issue with client side load balancing but it requires a method of updating the list of memcached server endpoints they are added or deleted.

One approach would be to build client side load balancing into the application using the endpoints from Kubernetes. The downside of this is that it requires application code changes and would couple your application to the Kubernetes environment. Client side load balancing could be achieved some other way using a third party service discovery application like [Consul](https://www.consul.io/), [Zookeeper](https://zookeeper.apache.org/), or [ectd](https://coreos.com/etcd/). The downside of this approach is that another service will need to be maintained for service discovery.

Another approach is to create a sidecar for your application to proxy memcached requests. This would decouple the application from Kubernetes and provide a single endpoint for the application to connect to. However, it would require extra resources in the cluster that scales with the number of client application instances.

## Architecture

The memcached operator introduces a `MemcachedProxy` [custom resource definition](https://kubernetes.io/docs/tasks/access-kubernetes-api/extend-api-custom-resource-definitions/) (CRD) which is used to describe a proxy to a set of memcached instances. For each `MemcachedProxy` a deployment is created to manage the proxy itself. The memcached operator watches for changes, such as additions or deletions, to memcached server pool and updates the configmap and deployment to apply the new changes. A service for the proxy are also created and managed by the memcached operator. Users connect to the proxy via this service endpoint.

Internally, the memcached operator uses [mcrouter](https://github.com/facebook/mcrouter) as the memcached proxy.

![diagram](design.png)

## Pools

Each service for a set of memcached clusters maps one-to-one with a [mcrouter pool](https://github.com/facebook/mcrouter/wiki/Pools). The memcached operator supports [sharded](sharded-pools.md) and [replicated](replicated-pools.md) pools, as well as combinations of both, via mcrouter. Sharded clusters shard keys based on a [hash](https://github.com/facebook/mcrouter/wiki/Pools#hash-functions) of the key. Replicated clusters store all keys in on all members of the pool.

## Internal architecture

The memcached operator application is written in [Go](http://www.golang.org/) and makes extensive use of the Kubernetes [client-go](https://github.com/kubernetes/client-go) client library. The main logic is performed by four Kubernetes controllers (control loops); the [proxy](../pkg/controller/proxy/) controller, the [proxydeployment](../pkg/controller/proxydeployment/) controller, the [proxyservice](../pkg/controller/proxyservice) controller, and the [proxyconfigmap](../pkg/controller/proxyconfigmap/) controller. Each controller's control loop is run in parallel in a goroutine. Controllers are managed using the [controllerutil](https://github.com/ianlewis/controllerutil) library.

## Scaling Memcached Pools

When the pods for the memcached pool are added or deleted, the memcached operator generates a new [mcrouter configuration file](https://github.com/facebook/mcrouter/wiki/Config-Files) and creates a new a configmap that holds the configuration. The new configmap name is applied to the deployment, triggering a rolling update.
