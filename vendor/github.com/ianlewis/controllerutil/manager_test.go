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

package controllerutil

import (
	"context"
	"sync"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"github.com/ianlewis/controllerutil/controller"
)

type TestController struct {
	called bool
}

func (c *TestController) Run(ctx context.Context) error {
	c.called = true
	return nil
}

// TestRun tests the run method to make sure that registered controllers were started.
func TestRun(t *testing.T) {
	client := newClient(t)

	m := New("test-run", client)
	c := &TestController{}
	m.Register("test", func(ctx *controller.Context) controller.Interface {
		return c
	})

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	wg.Add(1)
	go func() {
		err := m.Run(ctx)
		if err != nil {
			t.Error("error was returned from Run", err)
		}
		wg.Done()
	}()

	cancel()
	wg.Wait()

	if !c.called {
		t.Error("controller was never started")
	}
}

func newClient(t *testing.T) clientset.Interface {
	client, err := clientset.NewForConfig(&rest.Config{
		Host: "http://testserver",
		ContentConfig: rest.ContentConfig{
			GroupVersion:         &corev1.SchemeGroupVersion,
			NegotiatedSerializer: serializer.DirectCodecFactory{CodecFactory: scheme.Codecs},
		},
	})

	if err != nil {
		t.Error("could not create test client", err)
		return nil
	}

	return client
}
