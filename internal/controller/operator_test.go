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

package controller_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	// "k8s.io/client-go/kubernetes/fake"
	appsv1listers "k8s.io/client-go/listers/apps/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/ianlewis/memcached-operator/internal/apis/ianlewis.org/v1alpha1"
	"github.com/ianlewis/memcached-operator/internal/controller"
)

func newDeploymentLister(deployments ...*appsv1.Deployment) appsv1listers.DeploymentLister {
	indexer := cache.NewIndexer(
		cache.DeletionHandlingMetaNamespaceKeyFunc,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)

	for _, d := range deployments {
		indexer.Add(d)
	}

	return appsv1listers.NewDeploymentLister(indexer)
}

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

// TestGetDeploymentsForProxy tests the GetDeploymentsForProxy function.
func TestGetDeploymentsForProxy(t *testing.T) {
	p := &v1alpha1.MemcachedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hoge",
			Namespace: metav1.NamespaceDefault,
			UID:       uuid.NewUUID(),
		},
		Spec: v1alpha1.MemcachedClusterSpec{
			Rules: v1alpha1.RuleSpec{
				Service: &v1alpha1.ServiceSpec{
					Name: "fuga",
				},
			},
		},
	}

	t.Run("returns single deployment", func(t *testing.T) {
		dLister := newDeploymentLister(&appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:            "hoge",
				Namespace:       p.Namespace,
				OwnerReferences: []metav1.OwnerReference{*controller.NewProxyOwnerRef(p)},
			},
		})

		d, dList, err := controller.GetDeploymentsForProxy(dLister, p)
		if assert.NoError(t, err) {
			assert.NotNil(t, d, "1 deployment should be returned")
			assert.Len(t, dList, 0, "1 deployment should be returned")
		}
	})

	t.Run("returns only owned deployments", func(t *testing.T) {
		dLister := newDeploymentLister(
			&appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "hoge1",
					Namespace:       p.Namespace,
					OwnerReferences: []metav1.OwnerReference{*controller.NewProxyOwnerRef(p)},
				},
			},
			&appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "hoge2",
					Namespace:       p.Namespace,
					OwnerReferences: []metav1.OwnerReference{*controller.NewProxyOwnerRef(p)},
				},
			},
			&appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fuga1",
					Namespace: p.Namespace,
					// NOTE: this deployment is not owned
					OwnerReferences: []metav1.OwnerReference{},
				},
			},
		)

		newest, dList, err := controller.GetDeploymentsForProxy(dLister, p)
		if assert.NoError(t, err) {
			assert.NotNil(t, newest, "The newest deployment should be returned")
			assert.Len(t, dList, 1, "1 extra deployments should be returned")
			assert.Condition(t, func() bool { return metav1.IsControlledBy(newest, p) }, "deployment should be owned by memcached proxy")
			for _, d := range dList {
				assert.Condition(t, func() bool { return metav1.IsControlledBy(d, p) }, "deployment should be owned by memcached proxy")
			}
		}
	})
}

// TestGetConfigMapsForProxy tests the GetConfigMapsForProxy function.
func TestGetConfigMapsForProxy(t *testing.T) {
	p := &v1alpha1.MemcachedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hoge",
			Namespace: metav1.NamespaceDefault,
			UID:       uuid.NewUUID(),
		},
		Spec: v1alpha1.MemcachedClusterSpec{
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
