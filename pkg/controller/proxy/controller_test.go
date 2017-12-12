package proxy

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	"github.com/ianlewis/controllerutil/informers"
	"github.com/ianlewis/controllerutil/logging"

	"github.com/ianlewis/memcached-operator/pkg/apis/ianlewis.org/v1alpha1"
	ianlewisorgfake "github.com/ianlewis/memcached-operator/pkg/client/clientset/versioned/fake"
	ianlewisorginformers "github.com/ianlewis/memcached-operator/pkg/client/informers/externalversions/ianlewis/v1alpha1"
)

func getKey(o interface{}, t *testing.T) string {
	if key, err := KeyFunc(o); err != nil {
		t.Errorf("Unexpected error getting key for %v: %v", o, err)
		return ""
	} else {
		return key
	}
}

func alwaysSynced() bool {
	return true
}

type fixture struct {
	t *testing.T

	client            *fake.Clientset
	ianlewisorgClient *ianlewisorgfake.Clientset

	proxies []*v1alpha1.MemcachedProxy

	// Actions expected to happen on the client.
	actions []core.Action
}

func newFixture(t *testing.T) *fixture {
	return &fixture{
		t: t,
	}
}

func (f *fixture) newController() (*Controller, informers.SharedInformers) {
	objects := make([]runtime.Object, len(f.proxies))
	for i, p := range f.proxies {
		objects[i] = p
	}

	f.client = fake.NewSimpleClientset()
	f.ianlewisorgClient = ianlewisorgfake.NewSimpleClientset(objects...)

	informers := informers.NewSharedInformers()
	proxyInformer := informers.InformerFor(
		&v1alpha1.MemcachedProxy{},
		func() cache.SharedIndexInformer {
			return ianlewisorginformers.NewMemcachedProxyInformer(
				f.ianlewisorgClient,
				corev1.NamespaceAll,
				0,
				cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
			)
		},
	)

	c := New(
		"test-proxy-controller",
		f.client,
		f.ianlewisorgClient,
		proxyInformer,
		&record.FakeRecorder{},
		logging.New(""),
		1,
	)

	c.pListerSynced = alwaysSynced
	for _, p := range f.proxies {
		proxyInformer.GetIndexer().Add(p)
	}

	return c, informers
}

func (f *fixture) run(key string) {
	f.run_(key, false)
}

func (f *fixture) runExpectError(key string) {
	f.run_(key, true)
}

func (f *fixture) run_(key string, expectError bool) {
	c, informers := f.newController()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		err := informers.Run(ctx)
		if err != nil {
			f.t.Fatalf("error returned from informers: %v", err)
		}
	}()

	err := c.syncHandler(key)
	if !expectError && err != nil {
		f.t.Errorf("error syncing memcached proxy: %v", err)
	} else if expectError && err == nil {
		f.t.Error("expected error syncing memcached proxy, got nil")
	}

	actions := f.client.Actions()
	if len(actions) > 0 {
		f.t.Errorf("%d unexpected actions: %+v", len(actions), actions)
	}

	actions = f.ianlewisorgClient.Actions()
	for i, action := range actions {
		if len(f.actions) < i+1 {
			f.t.Errorf("%d unexpected actions: %+v", len(actions)-len(f.actions), actions[i:])
			break
		}

		expectedAction := f.actions[i]
		if !(expectedAction.Matches(action.GetVerb(), action.GetResource().Resource) && action.GetSubresource() == expectedAction.GetSubresource()) {
			f.t.Errorf("Expected\n\t%#v\ngot\n\t%#v", expectedAction, action)
			continue
		}
	}

	if len(f.actions) > len(actions) {
		f.t.Errorf("%d additional expected actions: %+v", len(f.actions)-len(actions), f.actions[len(actions):])
	}
}

// TestNewProxy tests the handling of the memcached proxy controller when a new memcached proxy is created
func TestNewProxy(t *testing.T) {
	f := newFixture(t)

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

	f.proxies = append(f.proxies, p)
	f.actions = append(f.actions, core.NewUpdateAction(schema.GroupVersionResource{Resource: "memcachedproxies"}, p.Namespace, p))
	f.run(getKey(p, t))
}
