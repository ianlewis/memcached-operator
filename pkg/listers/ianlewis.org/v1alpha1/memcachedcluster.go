// Copyright 2017 Google Inc.
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

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"

	"github.com/IanLewis/memcached-operator/pkg/api/ianlewis.org/v1alpha1"
)

// MemcachedClusterLister is a helper for listing MemcachedClusters.
type MemcachedClusterLister interface {
	// List lists all MemcachedClusters in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.MemcachedCluster, err error)
	// MemcachedClusters() returns an object that can list and get MemcachedClusters
	MemcachedClusters(namespace string) MemcachedClusterNamespaceLister
}

type memcachedClusterLister struct {
	indexer cache.Indexer
}

// NewMemcachedClusterLister returns a new MemcachedClusterLister.
func NewMemcachedClusterLister(indexer cache.Indexer) MemcachedClusterLister {
	return &memcachedClusterLister{indexer: indexer}
}

// List lists all MemcachedClusters in the indexer.
func (s *memcachedClusterLister) List(selector labels.Selector) (ret []*v1alpha1.MemcachedCluster, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.MemcachedCluster))
	})
	return ret, err
}

// MemcachedClusters returns an object that can list and get MemcachedClusters.
func (s *memcachedClusterLister) MemcachedClusters(namespace string) MemcachedClusterNamespaceLister {
	return memcachedClusterNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// MemcachedClusterNamespaceLister helps list and get MemcachedClusters.
type MemcachedClusterNamespaceLister interface {
	// List lists all MemcachedClusters in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1alpha1.MemcachedCluster, err error)
	// Get retrieves the MemcachedCluster from the indexer for a given namespace and name.
	Get(name string) (*v1alpha1.MemcachedCluster, error)
}

// memcachedClusterNamespaceLister implements the MemcachedClusterNamespaceLister
// interface.
type memcachedClusterNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all MemcachedClusters in the indexer for a given namespace.
func (s memcachedClusterNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.MemcachedCluster, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.MemcachedCluster))
	})
	return ret, err
}

// Get retrieves the MemcachedCluster from the indexer for a given namespace and name.
func (s memcachedClusterNamespaceLister) Get(name string) (*v1alpha1.MemcachedCluster, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("memcachedcluster"), name)
	}
	return obj.(*v1alpha1.MemcachedCluster), nil
}
