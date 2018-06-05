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

package proxydeployment

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
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

// newFixture creates a new fixture for the proxydeployment controller
func newFixture(t *testing.T, proxies []*v1alpha1.MemcachedProxy, deployments []*appsv1.Deployment, configMaps []*corev1.ConfigMap) *fixture {
	cf := test.NewClientFixture(t, proxies, deployments, nil, configMaps, nil, nil)

	c := New(
		"test-proxy-controller",
		cf.Client,
		cf.CRDClient,
		cf.MemcachedProxyInformer,
		cf.DeploymentInformer,
		cf.ConfigMapInformer,
		&record.FakeRecorder{},
		logging.New(""),
		1,
	)

	return &fixture{cf, c}
}

// TestSyncHandler tests the functionality of the proxydeployment
// controller's syncHandler method
func TestSyncHandler(t *testing.T) {
	// Tests whether a deployment for mcrouter is created when
	// a new MemcachedProxy is created.
	t.Run("a new deployment should be created", func(t *testing.T) {
		p := test.NewShardedProxy("hoge", "fuga")
		cm := test.NewProxyConfigMap(p)
		f := newFixture(t, []*v1alpha1.MemcachedProxy{p}, nil, []*corev1.ConfigMap{cm})

		f.runSync(getKey(p, t))

		d := test.NewMemcachedProxyDeployment(p, cm)
		f.ExpectClientActions([]test.ExpectedAction{
			{
				core.NewCreateAction(schema.GroupVersionResource{Resource: "deployments"}, d.Namespace, d),
				func(t *testing.T, action core.Action) {
					a := action.(core.UpdateAction)
					dNew := a.GetObject().(*appsv1.Deployment)

					// Check object ownership
					assert.True(t, metav1.IsControlledBy(dNew, p), "deployment must be owned by memcached proxy")

					// Check the image is set properly
					assert.Equal(t, dNew.Spec.Template.Spec.Containers[0].Image,
						p.Spec.McRouter.Image, "deployment image must be set from proxy")

					// Check that port is set properly
					assert.Equal(t, dNew.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort,
						*p.Spec.McRouter.Port, "container port must be set from proxy")
				},
			},
		})
	})
}
