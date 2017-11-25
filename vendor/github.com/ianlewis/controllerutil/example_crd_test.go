// Copyright 2017 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controllerutil_test

import (
	"context"
	"log"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	"github.com/ianlewis/controllerutil"
	"github.com/ianlewis/controllerutil/controller"
	"github.com/ianlewis/controllerutil/logging"

	// CRD type definitions
	examplev1 "github.com/ianlewis/controllerutil/example/pkg/apis/example.com/v1"
	// Generated CRD client
	fooclientset "github.com/ianlewis/controllerutil/example/pkg/client/clientset/versioned"
	// Generated CRD informer
	fooinformers "github.com/ianlewis/controllerutil/example/pkg/client/informers/externalversions/example/v1"
)

// FooController is a controller that watches Foo custom resource instances.
type FooController struct {
	client      clientset.Interface
	fooClient   fooclientset.Interface
	fooInformer cache.SharedIndexInformer
	recorder    record.EventRecorder
	infoLogger  *logging.Logger
	errorLogger *log.Logger
}

func (c *FooController) Run(ctx context.Context) error {
	// run control loop
	return nil
}

// NewFooController creates a new controller instance
func NewFooController(
	client clientset.Interface,
	fooClient fooclientset.Interface,
	fooInformer cache.SharedIndexInformer,
	recorder record.EventRecorder,
	infoLogger *logging.Logger,
	errorLogger *log.Logger,
) *FooController {
	c := &FooController{
		client:      client,
		fooClient:   fooClient,
		fooInformer: fooInformer,
		recorder:    recorder,
		infoLogger:  infoLogger,
		errorLogger: errorLogger,
	}

	// Attach event handlers
	// fooInformer.AddEventHandler(...)

	return c
}

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
	m := controllerutil.New("foo", client)

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
				metav1.NamespaceAll,
				metav1.GroupVersionKind{
					Group:   examplev1.SchemeGroupVersion.Group,
					Version: examplev1.SchemeGroupVersion.Version,
					Kind:    "Foo",
				},
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
			// ctx.InfoLogger is a convenient wrapper used for logging.
			ctx.InfoLogger,
			// ctx.ErrorLogger is a convenient wrapper used for error logging.
			ctx.ErrorLogger,
		)
	})

	if err := m.Run(context.Background()); err != nil {
		// handle error
	}
}
