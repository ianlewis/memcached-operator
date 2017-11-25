# Memcached Operator

memcached-operator is a Kubernetes [Operator](https://coreos.com/blog/introducing-operators.html) for deploying and managing a cluster of [Memcached](https://memcached.org/) instances.

memcached-operator provides a single Service endpoint that memcached client applications can connect to to make use of the memcached cluster. It provides this via a memcached proxy which is automatically updated whenever memcached instances are added or removed from the cluster.

## Project Status

**Project status:** *alpha* 

memcached-operator is still under active development and has not been extensively tested yet. Use at your own risk. Backward-compatibility is not supported for alpha releases.

## Prerequisites

* Version >= 1.8 of Kubernetes.

## Installation

TODO: Create installation manifests and instructions

## Usage

TODO: Create command line client
TODO: Usage documentation

## Removal

TODO: Removal instructions.

## Development

Check out memcached-operator to your `GOPATH`

### Building

memcached-operator can be built using the normal Go build tools.

```
$ go build github.com/ianlewis/memcached-operator/cmd/memcached-operator
```

### Running Tests

Tests can be run using the normal Go tools.

TODO: Include dependencies for running tests for vendored libraries in vendor

```
$ go test
```

Tests in the vendor directory can be omitted like so.

```
$ go test $(go list ./... | grep -v /vendor/)
``` 

### End to End Tests

TODO: Create end-to-end tests and instructions.

## Disclaimers

This is not an official Google product
