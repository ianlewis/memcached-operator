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

package test

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	appsv1informers "k8s.io/client-go/informers/apps/v1"
	corev1informers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	"github.com/ianlewis/controllerutil/informers"

	"github.com/ianlewis/memcached-operator/pkg/apis/ianlewis.org/v1alpha1"
	crdfake "github.com/ianlewis/memcached-operator/pkg/client/clientset/versioned/fake"
	crdinformers "github.com/ianlewis/memcached-operator/pkg/client/informers/externalversions/ianlewis/v1alpha1"
)

// ClientFixture holds fake clients and informers
type ClientFixture struct {
	T         *testing.T
	Client    *fake.Clientset
	CRDClient *crdfake.Clientset

	MemcachedProxyInformer cache.SharedIndexInformer
	DeploymentInformer     cache.SharedIndexInformer
	ReplicaSetInformer     cache.SharedIndexInformer
	ConfigMapInformer      cache.SharedIndexInformer
	ServiceInformer        cache.SharedIndexInformer
	EndpointsInformer      cache.SharedIndexInformer

	Informers informers.SharedInformers
}

// NewClientFixture sets up fake clients and informers, and returns a fixture containing those objects
func NewClientFixture(
	t *testing.T,
	proxies []*v1alpha1.MemcachedProxy,
	deployments []*appsv1.Deployment,
	replicasets []*appsv1.ReplicaSet,
	configMaps []*corev1.ConfigMap,
	services []*corev1.Service,
	endpoints []*corev1.Endpoints,
) ClientFixture {
	crdObjects := make([]runtime.Object, len(proxies))
	for i, p := range proxies {
		crdObjects[i] = p
	}
	var objects []runtime.Object
	for _, d := range deployments {
		objects = append(objects, d)
	}
	for _, rs := range replicasets {
		objects = append(objects, rs)
	}
	for _, cm := range configMaps {
		objects = append(objects, cm)
	}
	for _, s := range services {
		objects = append(objects, s)
	}
	for _, ep := range endpoints {
		objects = append(objects, ep)
	}

	client := fake.NewSimpleClientset(objects...)
	crdClient := crdfake.NewSimpleClientset(crdObjects...)

	informers := informers.NewSharedInformers()

	// Create the MemcachedProxy Informer
	proxyInformer := informers.InformerFor(
		&v1alpha1.MemcachedProxy{},
		func() cache.SharedIndexInformer {
			return crdinformers.NewMemcachedProxyInformer(
				crdClient,
				metav1.NamespaceAll,
				0,
				cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
			)
		},
	)
	for _, p := range proxies {
		proxyInformer.GetIndexer().Add(p)
	}

	// Create the Deployment Informer
	deploymentInformer := informers.InformerFor(
		&appsv1.Deployment{},
		func() cache.SharedIndexInformer {
			return appsv1informers.NewDeploymentInformer(
				client,
				metav1.NamespaceAll,
				0,
				cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
			)
		},
	)
	for _, d := range deployments {
		deploymentInformer.GetIndexer().Add(d)
	}

	// Create the ReplicaSet Informer
	replicasetInformer := informers.InformerFor(
		&appsv1.ReplicaSet{},
		func() cache.SharedIndexInformer {
			return appsv1informers.NewReplicaSetInformer(
				client,
				metav1.NamespaceAll,
				0,
				cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
			)
		},
	)
	for _, rs := range replicasets {
		replicasetInformer.GetIndexer().Add(rs)
	}

	// Create the ConfigMap Informer
	configMapInformer := informers.InformerFor(
		&corev1.ConfigMap{},
		func() cache.SharedIndexInformer {
			return corev1informers.NewConfigMapInformer(
				client,
				metav1.NamespaceAll,
				0,
				cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
			)
		},
	)
	for _, cm := range configMaps {
		configMapInformer.GetIndexer().Add(cm)
	}

	// Create the Service Informer
	serviceInformer := informers.InformerFor(
		&corev1.Service{},
		func() cache.SharedIndexInformer {
			return corev1informers.NewServiceInformer(
				client,
				metav1.NamespaceAll,
				0,
				cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
			)
		},
	)
	for _, s := range services {
		serviceInformer.GetIndexer().Add(s)
	}

	// Create the Endpoints Informer
	endpointsInformer := informers.InformerFor(
		&corev1.Endpoints{},
		func() cache.SharedIndexInformer {
			return corev1informers.NewEndpointsInformer(
				client,
				metav1.NamespaceAll,
				0,
				cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
			)
		},
	)
	for _, ep := range endpoints {
		endpointsInformer.GetIndexer().Add(ep)
	}

	return ClientFixture{
		T:         t,
		Client:    client,
		CRDClient: crdClient,

		MemcachedProxyInformer: proxyInformer,
		DeploymentInformer:     deploymentInformer,
		ReplicaSetInformer:     replicasetInformer,
		ConfigMapInformer:      configMapInformer,
		ServiceInformer:        serviceInformer,
		EndpointsInformer:      endpointsInformer,

		Informers: informers,
	}
}
