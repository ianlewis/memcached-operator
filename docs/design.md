# Memcached Operator Design

This document describes the architecture & design of the Memcached Operator.

## Overview

The Memcached Operator allows users to create a cluster that is accessable via a single Kubernetes `Service` endpoint and shards memcached keys using a proxy server.

The Memcached Operator introduces a `MemcachedCluster` third party resource (TPR) which is used to describe the desired state of a Memcached cluster. It provides options, such as the number of replicas, to scale and manage the cluster.  For each `MemcachedCluster` two `Deployment`s are created, one for memcached itself, and one for the memcached proxy. A `ConfigMap` and `Service` for use by the proxy are also created and managed by the Memcached Operator.

If a user scales the cluster by changing the number of replicas, the MemcachedOperator will scale the Memcached `Deployment` and update the proxy's `ConfigMap` to update the Pod IPs for the cluster.

Currently the Memcached Operator supports using [twemproxy](https://github.com/twitter/twemproxy) as a memcached proxy but may support others in the future. For twemproxy, a new `ConfigMap` is created and bound to the `Deployment`. This causes a rolling update for twemproxy and new configuration is gradually applied to the cluster.

## MemcachedCluster Third-PartyResource

The `MemcachedCluster` third party resource has the following structure. This example shows the default values. All values of the `spec` field are optional.

```
apiVersion: ianlewis.org/v1alpha1
kind: MemcachedCluster
  name: test-cluster
spec:
  replicas: 5
  memcachedImage: memcached:1.4.36-alpine
  proxy:
    twemproxy:
      distribution: ketama
      hash: fnv1a_64
      image: asia.gcr.io/ian-corp-dev/twemproxy:v0.4.1-0
      serverFailureLimit: 1
      serverRetryTimeout: 2000
```

Future support for other proxies can be added in the `proxy` field of the `MemcachedCluster.spec`.

## Health Checking

The Memcached Operator does not rely on Kubernetes readiness checks to assess whether Memcached instances can serve traffic. Instead, it relies on the proxy server to forward requests only to healthy instances. If no healthy instances are available the proxy server may return errors.
