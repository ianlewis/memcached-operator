package controller_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	// "k8s.io/client-go/kubernetes/fake"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/ianlewis/memcached-operator/pkg/apis/ianlewis.org/v1alpha1"
	"github.com/ianlewis/memcached-operator/pkg/controller"
)

func newConfigMapLister(cMaps ...*corev1.ConfigMap) corev1listers.ConfigMapLister {
	indexer := cache.NewIndexer(
		cache.DeletionHandlingMetaNamespaceKeyFunc,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)

	for _, cm := range cMaps {
		indexer.Add(cm)
	}

	return corev1listers.NewConfigMapLister(indexer)
}

// TestGetConfigMapsForProxy tests the GetConfigMapsForProxy function.
func TestGetConfigMapsForProxy(t *testing.T) {
	p := &v1alpha1.MemcachedProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hoge",
			Namespace: metav1.NamespaceDefault,
			UID:       uuid.NewUUID(),
		},
		Spec: v1alpha1.MemcachedProxySpec{
			Rules: v1alpha1.RuleSpec{
				Service: &v1alpha1.ServiceSpec{
					Name: "fuga",
				},
			},
		},
	}

	t.Run("returns single config map", func(t *testing.T) {
		cmLister := newConfigMapLister(&corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:            "hoge",
				Namespace:       p.Namespace,
				OwnerReferences: []metav1.OwnerReference{*controller.NewProxyOwnerRef(p)},
			},
		})

		cMaps, err := controller.GetConfigMapsForProxy(cmLister, p)
		if assert.NoError(t, err) {
			assert.Len(t, cMaps, 1, "1 config map should be returned")
		}
	})

	t.Run("returns only owned config maps", func(t *testing.T) {
		cmLister := newConfigMapLister(
			&corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "ConfigMap",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "hoge1",
					Namespace:       p.Namespace,
					OwnerReferences: []metav1.OwnerReference{*controller.NewProxyOwnerRef(p)},
				},
			},
			&corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "ConfigMap",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "hoge2",
					Namespace:       p.Namespace,
					OwnerReferences: []metav1.OwnerReference{*controller.NewProxyOwnerRef(p)},
				},
			},
			&corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "ConfigMap",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fuga1",
					Namespace: p.Namespace,
					// NOTE: This config map is not owned
					OwnerReferences: []metav1.OwnerReference{},
				},
			},
		)

		cMaps, err := controller.GetConfigMapsForProxy(cmLister, p)
		if assert.NoError(t, err) {
			assert.Len(t, cMaps, 2, "2 config maps should be returned")
			for _, cm := range cMaps {
				assert.Condition(t, func() bool { return metav1.IsControlledBy(cm, p) }, "config map should be owned by memcached proxy")
			}
		}
	})
}
