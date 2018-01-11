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
