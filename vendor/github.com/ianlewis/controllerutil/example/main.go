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

package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang/glog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/ianlewis/controllerutil"
	"github.com/ianlewis/controllerutil/controller"

	examplev1 "github.com/ianlewis/controllerutil/example/pkg/apis/example.com/v1"
	fooclientset "github.com/ianlewis/controllerutil/example/pkg/client/clientset/versioned"
	fooinformers "github.com/ianlewis/controllerutil/example/pkg/client/informers/externalversions/example/v1"
	foo "github.com/ianlewis/controllerutil/example/pkg/controller/foo"
)

func main() {
	kubeconfig := flag.String("kubeconfig", "", "The path to a kubeconfig. Default is in-cluster config.")
	defaultResync := flag.Duration("default-resync", 12*time.Hour, "The default amount of time between re-syncs.")
	namespace := flag.String("namespace", metav1.NamespaceAll, "The namespace to watch. Defaults to all namespaces.")

	flag.Parse()

	config, err := buildConfig(*kubeconfig)
	if err != nil {
		glog.Exitf("Could not read kubeconfig %q: %v", *kubeconfig, err)
	}

	client, err := clientset.NewForConfig(config)
	if err != nil {
		glog.Exitf("Could not create Kubernetes API client: %v", err)
	}

	fooclient, err := fooclientset.NewForConfig(config)
	if err != nil {
		glog.Exitf("Could not create Foo API client: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Watch for SIGINT or SIGTERM and cancel the Operator's context if one is received.
	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
		s := <-signals
		glog.Infof("Received signal %s, exiting...", s)
		cancel()
	}()

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

	err = m.Run(ctx)

	// Ensure cancel() is called to clean up.
	cancel()

	if err != nil {
		glog.Exitf("Unhandled error received. Exiting: %#v", err)
	}
}

func buildConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}
