// Copyright 2017 Google LLC
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

package proxy

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

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

func newFixture(t *testing.T, proxies []*v1alpha1.MemcachedProxy) *fixture {
	cf := test.NewClientFixture(t, proxies, nil, nil, nil, nil, nil)

	c := New(
		"test-proxy-controller",
		cf.Client,
		cf.CRDClient,
		cf.MemcachedProxyInformer,
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

// TestSync tests the handling of the memcached proxy
// controller when a new memcached proxy events occur
func TestSync(t *testing.T) {
	t.Run("the new proxy should be updated", func(t *testing.T) {
		// Create a raw MemcachedProxy object as it
		// would be created via kubectl
		p := &v1alpha1.MemcachedProxy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "hoge",
				Namespace: metav1.NamespaceDefault,
			},
			Spec: v1alpha1.MemcachedProxySpec{
				Rules: v1alpha1.RuleSpec{
					Service: &v1alpha1.ServiceSpec{
						Name: "fuga",
					},
				},
			},
		}

		f := newFixture(t, []*v1alpha1.MemcachedProxy{p})

		f.runSync(getKey(p, t))

		f.ExpectCRDClientActions([]test.ExpectedAction{
			{
				core.NewUpdateAction(schema.GroupVersionResource{Resource: "memcachedproxies"}, p.Namespace, p),
				func(t *testing.T, action core.Action) {
					a := action.(core.UpdateAction)
					pNew := a.GetObject().(*v1alpha1.MemcachedProxy)

					// Check that the default rule type was set
					assert.Equal(t, pNew.Spec.Rules.Type, v1alpha1.ShardedRuleType, "rules type must be equal")

					// Check that the observed hash was set
					assert.NotEmpty(t, pNew.Status.ObservedSpecHash, "observed spec hash must be set")
				},
			},
		})
	})
}
