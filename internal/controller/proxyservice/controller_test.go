// Copyright 2018 Google LLC
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

package proxyservice

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"

	"github.com/ianlewis/controllerutil/logging"

	"github.com/ianlewis/memcached-operator/internal/apis/ianlewis.org/v1alpha1"
	"github.com/ianlewis/memcached-operator/internal/test"
)

func getKey(o interface{}, t *testing.T) string {
	key, err := KeyFunc(o)
	if !assert.NoError(t, err, "must be able to generate key for object") {
		assert.FailNow(t, "failing early")
	}
	return key
}

type fixture struct {
	test.ClientFixture

	controller *Controller
}

func (f *fixture) runSync(key string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		err := f.Informers.Run(ctx)
		assert.NoError(f.T, err, "informers must start successfully")
	}()

	err := f.controller.syncHandler(key)
	assert.NoError(f.T, err, "syncHandler must complete successfully")
}

func newFixture(t *testing.T, proxies []*v1alpha1.MemcachedCluster, services []*corev1.Service) *fixture {
	cf := test.NewClientFixture(t, proxies, nil, nil, nil, services, nil)

	c := New(
		"test-proxy-controller",
		cf.Client,
		cf.CRDClient,
		cf.MemcachedClusterInformer,
		cf.ServiceInformer,
		&record.FakeRecorder{},
		logging.New(""),
		1,
	)

	return &fixture{cf, c}
}

// TestSyncHandler tests the functionality of the proxyservice
// controller's syncHandler method
func TestSyncHandler(t *testing.T) {
	// Tests whether a service is created when a new MemcachedCluster
	// is created.
	t.Run("a new service should be created", func(t *testing.T) {
		p := test.NewShardedProxy("hoge", "fuga")
		f := newFixture(t, []*v1alpha1.MemcachedCluster{p}, nil)

		f.runSync(getKey(p, t))

		s := test.NewMemcachedClusterService(p)
		f.ExpectClientActions([]test.ExpectedAction{
			{
				core.NewCreateAction(schema.GroupVersionResource{Resource: "services"}, s.Namespace, s),
				func(t *testing.T, action core.Action) {
					a := action.(core.CreateAction)
					sNew := a.GetObject().(*corev1.Service)

					assert.True(t, metav1.IsControlledBy(sNew, p), "owner reference should be set")

					// Check that port is set properly
					assert.Equal(t, sNew.Spec.Ports[0].Port,
						*p.Spec.McRouter.Port, "service port must be set from proxy")

					// Check that port is set properly
					assert.Equal(t, sNew.Spec.Ports[0].TargetPort.IntVal,
						*p.Spec.McRouter.Port, "target port must be set from proxy")
				},
			},
		})
	})
}
