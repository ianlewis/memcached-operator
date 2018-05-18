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

// THIS FILE IS AUTOMATICALLY GENERATED. DO NOT EDIT DIRECTLY.
package v1alpha1

import (
	v1alpha1 "github.com/ianlewis/memcached-operator/pkg/apis/ianlewis.org/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// MemcachedProxyLister helps list MemcachedProxies.
type MemcachedProxyLister interface {
	// List lists all MemcachedProxies in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.MemcachedProxy, err error)
	// MemcachedProxies returns an object that can list and get MemcachedProxies.
	MemcachedProxies(namespace string) MemcachedProxyNamespaceLister
	MemcachedProxyListerExpansion
}

// memcachedProxyLister implements the MemcachedProxyLister interface.
type memcachedProxyLister struct {
	indexer cache.Indexer
}

// NewMemcachedProxyLister returns a new MemcachedProxyLister.
func NewMemcachedProxyLister(indexer cache.Indexer) MemcachedProxyLister {
	return &memcachedProxyLister{indexer: indexer}
}

// List lists all MemcachedProxies in the indexer.
func (s *memcachedProxyLister) List(selector labels.Selector) (ret []*v1alpha1.MemcachedProxy, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.MemcachedProxy))
	})
	return ret, err
}

// MemcachedProxies returns an object that can list and get MemcachedProxies.
func (s *memcachedProxyLister) MemcachedProxies(namespace string) MemcachedProxyNamespaceLister {
	return memcachedProxyNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// MemcachedProxyNamespaceLister helps list and get MemcachedProxies.
type MemcachedProxyNamespaceLister interface {
	// List lists all MemcachedProxies in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1alpha1.MemcachedProxy, err error)
	// Get retrieves the MemcachedProxy from the indexer for a given namespace and name.
	Get(name string) (*v1alpha1.MemcachedProxy, error)
	MemcachedProxyNamespaceListerExpansion
}

// memcachedProxyNamespaceLister implements the MemcachedProxyNamespaceLister
// interface.
type memcachedProxyNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all MemcachedProxies in the indexer for a given namespace.
func (s memcachedProxyNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.MemcachedProxy, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.MemcachedProxy))
	})
	return ret, err
}

// Get retrieves the MemcachedProxy from the indexer for a given namespace and name.
func (s memcachedProxyNamespaceLister) Get(name string) (*v1alpha1.MemcachedProxy, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("memcachedproxy"), name)
	}
	return obj.(*v1alpha1.MemcachedProxy), nil
}