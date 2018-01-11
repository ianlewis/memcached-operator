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
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"

	"github.com/ianlewis/controllerutil/logging"

	"github.com/ianlewis/memcached-operator/pkg/apis/ianlewis.org/v1alpha1"
	"github.com/ianlewis/memcached-operator/pkg/test"
)

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
		cf.ServiceInformer,
		cf.EndpointsInformer,
		&record.FakeRecorder{},
		logging.New(""),
		1,
	)

	return &fixture{cf, c}
}
