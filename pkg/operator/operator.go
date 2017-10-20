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

package operator

import (
	"context"

	"golang.org/x/sync/errgroup"

	"github.com/golang/glog"

	clientset "k8s.io/client-go/kubernetes"

	ianlewisorgclientset "github.com/IanLewis/memcached-operator/pkg/client/clientset/versioned"
)

// Operator manages all goroutines for the application.
type Operator struct {
	// client is a Kubernetes API client
	client            clientset.Interface
	ianlewisorgClient ianlewisorgclientset.Interface
}

// New creates a new Operator instance.
func New(client clientset.Interface, ianlewisorgClient ianlewisorgclientset.Interface) *Operator {
	o := &Operator{
		client:            client,
		ianlewisorgClient: ianlewisorgClient,
	}

	return o
}

// Run starts the operator and required goroutines
func (o *Operator) Run(ctx context.Context) error {
	var wg errgroup.Group

	glog.V(3).Infof("Initializing operator.")

	return wg.Wait()
}
