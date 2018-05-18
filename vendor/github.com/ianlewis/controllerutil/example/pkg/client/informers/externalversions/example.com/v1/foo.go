// Copyright 2017 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package v1

import (
	time "time"

	example_com_v1 "github.com/ianlewis/controllerutil/example/pkg/apis/example.com/v1"
	versioned "github.com/ianlewis/controllerutil/example/pkg/client/clientset/versioned"
	internalinterfaces "github.com/ianlewis/controllerutil/example/pkg/client/informers/externalversions/internalinterfaces"
	v1 "github.com/ianlewis/controllerutil/example/pkg/client/listers/example.com/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// FooInformer provides access to a shared informer and lister for
// Foos.
type FooInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.FooLister
}

type fooInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewFooInformer constructs a new informer for Foo type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFooInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredFooInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredFooInformer constructs a new informer for Foo type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredFooInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ExampleV1().Foos(namespace).List(options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ExampleV1().Foos(namespace).Watch(options)
			},
		},
		&example_com_v1.Foo{},
		resyncPeriod,
		indexers,
	)
}

func (f *fooInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredFooInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *fooInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&example_com_v1.Foo{}, f.defaultInformer)
}

func (f *fooInformer) Lister() v1.FooLister {
	return v1.NewFooLister(f.Informer().GetIndexer())
}
