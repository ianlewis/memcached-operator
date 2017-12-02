// Copyright 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1informers "k8s.io/client-go/informers/core/v1"
	v1beta1informers "k8s.io/client-go/informers/extensions/v1beta1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/ianlewis/controllerutil"
	"github.com/ianlewis/controllerutil/controller"

	"github.com/ianlewis/memcached-operator/pkg/apis/ianlewis.org/v1alpha1"
	ianlewisorgclientset "github.com/ianlewis/memcached-operator/pkg/client/clientset/versioned"
	ianlewisorginformers "github.com/ianlewis/memcached-operator/pkg/client/informers/externalversions/ianlewis/v1alpha1"
	"github.com/ianlewis/memcached-operator/pkg/controller/proxy"
	"github.com/ianlewis/memcached-operator/pkg/controller/proxyconfigmap"
	"github.com/ianlewis/memcached-operator/pkg/controller/proxydeployment"
	"github.com/ianlewis/memcached-operator/pkg/controller/proxyservice"
)

func main() {
	kubeconfig := flag.String("kubeconfig", "", "The path to a kubeconfig. Default is in-cluster config.")

	namespace := flag.String("namespace", metav1.NamespaceAll, "The namespace to watch. Defaults to all namespaces.")
	defaultResync := flag.Duration("default-resync", 12*time.Hour, "The default resync interval. Default is 12 hours.")
	serviceWorkers := flag.Int("concurrent-service-syncs", 5, "The number of memcached proxy services that are allowed to sync concurrently. A larger number is more responsive, but more CPU and network load. Default is 5.")
	configMapWorkers := flag.Int("concurrent-configmap-syncs", 5, "The number of memcached proxy configmaps that are allowed to sync concurrently. A larger number is more responsive, but more CPU and network load. Default is 5.")

	flag.Parse()

	config, err := buildConfig(*kubeconfig)
	if err != nil {
		glog.Exitf("Could not read kubeconfig %q: %v", *kubeconfig, err)
	}

	client, err := clientset.NewForConfig(config)
	if err != nil {
		glog.Exitf("Could not create Kubernetes API client: %v", err)
	}

	ianlewisorgClient, err := ianlewisorgclientset.NewForConfig(config)
	if err != nil {
		glog.Exitf("Could not create ianlewis.org API client: %v", err)
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

	m := controllerutil.NewControllerManager("memcached-operator", client)

	m.Register("memcached-proxy", func(ctx *controller.Context) controller.Interface {
		return proxy.New(
			"memcached-proxy",
			ctx.Client,
			ianlewisorgClient,
			ctx.SharedInformers.InformerFor(
				&v1alpha1.MemcachedProxy{},
				func() cache.SharedIndexInformer {
					return ianlewisorginformers.NewMemcachedProxyInformer(
						ianlewisorgClient,
						*namespace,
						*defaultResync,
						cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
					)
				},
			),
			ctx.Recorder,
			ctx.Logger,
			*serviceWorkers,
		)
	})

	m.Register("memcached-proxy-service", func(ctx *controller.Context) controller.Interface {
		return proxyservice.New(
			"memcached-proxy-service",
			ctx.Client,
			ianlewisorgClient,
			ctx.SharedInformers.InformerFor(
				&v1alpha1.MemcachedProxy{},
				func() cache.SharedIndexInformer {
					return ianlewisorginformers.NewMemcachedProxyInformer(
						ianlewisorgClient,
						*namespace,
						*defaultResync,
						cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
					)
				},
			),
			ctx.SharedInformers.InformerFor(
				&corev1.Service{},
				func() cache.SharedIndexInformer {
					return corev1informers.NewServiceInformer(
						ctx.Client,
						*namespace,
						*defaultResync,
						cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
					)
				},
			),
			ctx.Recorder,
			ctx.Logger,
			*serviceWorkers,
		)
	})

	m.Register("memcached-proxy-configmap", func(ctx *controller.Context) controller.Interface {
		return proxyconfigmap.New(
			"memcached-proxy-configmap",
			ctx.Client,
			ianlewisorgClient,
			ctx.SharedInformers.InformerFor(
				&v1alpha1.MemcachedProxy{},
				func() cache.SharedIndexInformer {
					return ianlewisorginformers.NewMemcachedProxyInformer(
						ianlewisorgClient,
						*namespace,
						*defaultResync,
						cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
					)
				},
			),
			ctx.SharedInformers.InformerFor(
				&corev1.ConfigMap{},
				func() cache.SharedIndexInformer {
					return corev1informers.NewConfigMapInformer(
						ctx.Client,
						*namespace,
						*defaultResync,
						cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
					)
				},
			),
			ctx.SharedInformers.InformerFor(
				&corev1.Service{},
				func() cache.SharedIndexInformer {
					return corev1informers.NewServiceInformer(
						ctx.Client,
						*namespace,
						*defaultResync,
						cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
					)
				},
			),
			ctx.SharedInformers.InformerFor(
				&corev1.Endpoints{},
				func() cache.SharedIndexInformer {
					return corev1informers.NewEndpointsInformer(
						ctx.Client,
						*namespace,
						*defaultResync,
						cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
					)
				},
			),
			ctx.SharedInformers.InformerFor(
				&corev1.Pod{},
				func() cache.SharedIndexInformer {
					return corev1informers.NewPodInformer(
						ctx.Client,
						*namespace,
						*defaultResync,
						cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
					)
				},
			),
			ctx.Recorder,
			ctx.Logger,
			*configMapWorkers,
		)
	})

	m.Register("memcached-proxy-deployment", func(ctx *controller.Context) controller.Interface {
		return proxydeployment.New(
			"memcached-proxy-deployment",
			ctx.Client,
			ianlewisorgClient,
			ctx.SharedInformers.InformerFor(
				&v1alpha1.MemcachedProxy{},
				func() cache.SharedIndexInformer {
					return ianlewisorginformers.NewMemcachedProxyInformer(
						ianlewisorgClient,
						*namespace,
						*defaultResync,
						cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
					)
				},
			),
			ctx.SharedInformers.InformerFor(
				&v1beta1.Deployment{},
				func() cache.SharedIndexInformer {
					return v1beta1informers.NewDeploymentInformer(
						ctx.Client,
						*namespace,
						*defaultResync,
						cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
					)
				},
			),
			ctx.SharedInformers.InformerFor(
				&corev1.ConfigMap{},
				func() cache.SharedIndexInformer {
					return corev1informers.NewConfigMapInformer(
						ctx.Client,
						*namespace,
						*defaultResync,
						cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
					)
				},
			),
			ctx.Recorder,
			ctx.Logger,
			*configMapWorkers,
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
