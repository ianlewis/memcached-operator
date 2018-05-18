# Kubernetes Controller Utilities

[![](https://godoc.org/github.com/ianlewis/controllerutil?status.svg)](http://godoc.org/github.com/ianlewis/controllerutil) [![Go Report Card](https://goreportcard.com/badge/github.com/ianlewis/controllerutil)](https://goreportcard.com/report/github.com/ianlewis/controllerutil)

The `github.com/ianlewis/controllerutil` package provides a simple way to manage multiple Kubernetes controllers as part of a single process.

# Project Status

Project status: *alpha*

controllerutil is still under active development and has not been extensively tested yet. Use at your own risk. Backward-compatibility is not supported for alpha releases.

# Motivation

Kubernetes controllers are often run together as part of a single process but often it's unclear how to manage the lifecycle of each controller. Each controller also makes use of one or more "informers" which are used to watch the Kubernetes API for changes to objects and maintain a local cache per object type. Informers have their own lifecycle and need to managed as well.

This complexity, while powerful, often results in architectures that are error prone and don't handle edge cases well; particularly failure cases. The goal of this library is to provide an easy way for developers of controllers to take advantage of Go programming, logging, and Kubernetes API client best practices.

# Installation


Install using

    go get github.com/ianlewis/controllerutil

# Architecture

`controllerutil` helps manage client-go data structures. Each controller application watches Kubernetes objects for changes via the Kubernetes API. Watches are done via data structures called "informers". Informer logic is combined with a local cache of objects in a data structure called a "shared index informer. Each shared index informer is shared across the process per API type. Maintaining a cache local to the application is done to lessen the load on the Kubernetes API server.

Each controller application can contain multiple control loops called "controllers". Shared index informers for the necessary types are passed to controllers in their constructor. Each controller is run in parallel with others in a goroutine.

`controllerutil` implements a `ControllerManager` which manages each controller and associated goroutine. Since most controller applications require all controllers to properly function, If any controller fails ControllerManager stops all controllers and terminates. It is assumed that the controller will be run with a supervisor such as the kubelet and will be restarted.

`controllerutil` also has a `SharedInformers` data structure which manages a list of informers that is unique per Kubernetes API type and manages the associated goroutines.

# Usage

This example shows how to use the controllerutil package to implement a Kubernetes controller or operator application that makes use of Kubernetes custom resources. See the [example](./example) directory for a full example.

Here we create a `ControllerManager` via the `NewControllerManager` function. Then controllers are registered via the `Register` method. Finally we call the `Run` method to start all the registered controllers.

[embedmd]:# (example_crd_test.go go /func Example_customResourceDefinition/ $)
```go
func Example_customResourceDefinition() {
	// Initialize an in-cluster Kubernetes client
	config, _ := rest.InClusterConfig()
	client, _ := clientset.NewForConfig(config)

	// Create the client for the custom resource definition (CRD) "Foo"
	fooclient, err := fooclientset.NewForConfig(config)
	if err != nil {
		// handle error
	}

	// Create a new ControllerManager instance
	m := controllerutil.NewControllerManager("foo", client)

	// Register the foo controller.
	m.Register("foo", func(ctx *controller.Context) controller.Interface {
		return NewFooController(
			// ctx.Client is the same client passed to controllerutil.New
			ctx.Client,
			// fooclient is the CRD client instance
			fooclient,
			// ctx.SharedInformers manages lifecycle of all shared informers
			// InformerFor registers the informer for the given type if it hasn't been registered already.
			ctx.SharedInformers.InformerFor(
				&examplev1.Foo{},
				func() cache.SharedIndexInformer {
					return fooinformers.NewFooInformer(
						fooclient,
						metav1.NamespaceAll,
						12*time.Hour,
						cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
					)
				},
			),
			// ctx.Recorder is used for recording Kubernetes events.
			ctx.Recorder,
			// ctx.Logger is a convenient wrapper used for logging.
			ctx.Logger,
		)
	})

	if err := m.Run(context.Background()); err != nil {
		// handle error
	}
}
```

## Logging

`containerutil` provides a logger object per controller which can optionally be used to log messages. Loggers are a wrapper around [glog](https://github.com/golang/glog/) but provides easy creation of standard log.Logger objects. Each logger is given a prefix based on the controller name which allows for easier log viewing.

# Disclaimers

This is not an official Google product.
