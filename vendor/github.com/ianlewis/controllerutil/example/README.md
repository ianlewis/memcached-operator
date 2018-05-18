# example

This is an example application that has multiple controllers/control loops. It watches a [custom resource](https://kubernetes.io/docs/concepts/api-extension/custom-resources/) named `Foo` and prints the object.

[embedmd]:# (artifacts/crd.yaml yaml /apiVersion/ $)
```yaml
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: foos.example.com
spec:
  group: example.com
  version: v1
  names:
    kind: Foo
    plural: foos
  scope: Namespaced
```

The example application includes two controllers, "foo" and "bar". Each print changes to the `Foo` objects and run in parallel.

Each Foo object looks something like this:

[embedmd]:# (artifacts/example.yaml yaml /apiVersion/ $)
```yaml
apiVersion: example.com/v1
kind: Foo
metadata:
  name: example
spec:
  foo: "foo"
  bar: "bar"
```

The example application uses the `controllerutil` package to create and manage controllers and informers.

[embedmd]:# (main.go go /.*NewControllerManager/ /\}\)/)
```go
	m := controllerutil.NewControllerManager("foo-controller", client)

	// Register the foo controller.
	m.Register("foo", func(ctx *controller.Context) controller.Interface {
		return foo.New(
			"foo",
			ctx.Client,
			fooclient,
			ctx.SharedInformers.InformerFor(
				&examplev1.Foo{},
				func() cache.SharedIndexInformer {
					return fooinformers.NewFooInformer(
						fooclient,
						*namespace,
						*defaultResync,
						cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
					)
				},
			),
			ctx.Recorder,
			ctx.Logger,
		)
	})
```

# Building

Build by running in the current directory.

    go build
