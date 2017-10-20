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

	"github.com/golang/glog"

	"golang.org/x/sync/errgroup"

	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	// Required for Auth with GKE/Azure/OIDC
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	ianlewisorgclientset "github.com/IanLewis/memcached-operator/pkg/client/clientset/versioned"
	"github.com/IanLewis/memcached-operator/pkg/operator"
)

func main() {
	kubeconfig := flag.String("kubeconfig", "", "The path to a kubeconfig. Default is in-cluster config.")
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
	wg, ctx := errgroup.WithContext(ctx)

	oper := operator.New(client, ianlewisorgClient)

	wg.Go(func() error { return oper.Run(ctx) })

	glog.Infof("Memcached Operator started.")

	// Watch for SIGINT or SIGTERM and cancel the Operator's context if one is received.
	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
		s := <-signals
		glog.Infof("Received signal %s, exiting...", s)
		cancel()
	}()

	// Wait on the Operator.
	err = wg.Wait()

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
