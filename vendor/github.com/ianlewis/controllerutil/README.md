# Kubernetes Controller Utilities

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

# Usage

This example shows how to use the controllerutil package to implement a Kubernetes controller or operator application that makes use of Kubernetes custom resources. See the [example](./example) directory for a full example.

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

# Disclaimers

This is not an official Google product.
