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

package proxyconfigmap

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"

	"github.com/ianlewis/controllerutil/logging"

	"github.com/ianlewis/memcached-operator/pkg/apis/ianlewis.org/v1alpha1"
	"github.com/ianlewis/memcached-operator/pkg/test"
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

func alwaysSynced() bool {
	return true
}

func newFixture(
	t *testing.T,
	proxies []*v1alpha1.MemcachedProxy,
	configMaps []*corev1.ConfigMap,
	services []*corev1.Service,
	endpoints []*corev1.Endpoints,
) *fixture {
	cf := test.NewClientFixture(
		t,
		proxies,
		nil,
		nil,
		configMaps,
		services,
		endpoints,
	)

	c := New(
		"test-proxyconfigmap-controller",
		cf.Client,
		cf.CRDClient,
		metav1.NamespaceAll,
		cf.MemcachedProxyInformer,
		cf.ConfigMapInformer,
		cf.DeploymentInformer,
		cf.ReplicaSetInformer,
		cf.ServiceInformer,
		cf.EndpointsInformer,
		&record.FakeRecorder{},
		logging.New(""),
		1,
	)

	return &fixture{cf, c}
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

// TestSync tests the handling of the memcached proxyconfigmap
// controller when a new memcached proxy events occur
func TestSync(t *testing.T) {
	t.Run("a new configmap should be created", func(t *testing.T) {
		p := test.NewShardedProxy("hoge", "fuga")
		s, ep := test.NewMemcachedService("fuga")

		f := newFixture(
			t,
			[]*v1alpha1.MemcachedProxy{p},
			nil,
			[]*corev1.Service{s},
			[]*corev1.Endpoints{ep},
		)

		f.runSync(getKey(p, t))

		cm := test.NewProxyConfigMap(p)

		// Expect that a new ConfigMap was created
		f.ExpectClientActions([]test.ExpectedAction{
			{
				core.NewCreateAction(schema.GroupVersionResource{Resource: "configmaps"}, cm.Namespace, cm), func(t *testing.T, action core.Action) {
					a := action.(core.CreateAction)
					cmNew := a.GetObject().(*corev1.ConfigMap)
					assert.Regexp(t, fmt.Sprintf("^%s-config-", p.Name), cmNew.Name, "Name should be set")
					assert.Equal(t, cmNew.Namespace, p.Namespace, "Namespace should be set")
					assert.True(t, metav1.IsControlledBy(cmNew, p), "owner reference should be set")
				},
			},
		})
	})
}
