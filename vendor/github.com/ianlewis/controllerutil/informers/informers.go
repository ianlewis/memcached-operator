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

package informers

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

// NewInformerFunc is a simple factory for creating a new informer
type NewInformerFunc func() cache.SharedIndexInformer

// SharedInformers allows controllers to register informers that watch the Kubernetes API for changes in various API types and maintain a local cache of those objects.
type SharedInformers interface {
	// InformerFor registers an informer for the given type. If an informer for the given type has not been
	// registered before, the given constructor function is called to construct the informer.
	InformerFor(runtime.Object, NewInformerFunc) cache.SharedIndexInformer
	// WaitForCacheSync waits for all started informers' cache were synced.
	WaitForCacheSync(context.Context) error
	// Run starts all registered informers.
	Run(context.Context) error
}

type sharedInformers struct {
	lock             sync.Mutex
	informers        map[reflect.Type]cache.SharedIndexInformer
	startedInformers map[reflect.Type]bool
}

// NewSharedInformers creates an empty SharedInformers instance.
func NewSharedInformers() SharedInformers {
	return &sharedInformers{
		informers:        make(map[reflect.Type]cache.SharedIndexInformer),
		startedInformers: make(map[reflect.Type]bool),
	}
}

// InformerFor registers an informer for the given type. If an informer for the given type has not been registered before, the given constructor function is called to construct the informer.
func (s *sharedInformers) InformerFor(obj runtime.Object, f NewInformerFunc) cache.SharedIndexInformer {
	s.lock.Lock()
	defer s.lock.Unlock()

	informerType := reflect.TypeOf(obj)
	informer, nsExists := s.informers[informerType]
	if nsExists {
		return informer
	}
	informer = f()
	s.informers[informerType] = informer

	return informer
}

// WaitForCacheSync waits for all informers' cache to sync.
func (s *sharedInformers) WaitForCacheSync(ctx context.Context) error {
	informers := func() map[reflect.Type]cache.SharedIndexInformer {
		s.lock.Lock()
		defer s.lock.Unlock()

		informers := map[reflect.Type]cache.SharedIndexInformer{}
		for informerType, informer := range s.informers {
			informers[informerType] = informer
		}
		return informers
	}()

	for informerType, informer := range informers {
		if !cache.WaitForCacheSync(ctx.Done(), informer.HasSynced) {
			return fmt.Errorf("informer for type %s failed to sync", informerType)
		}
	}
	return nil
}

// Run runs all registered informers. Blocks until the context is finished.
func (s *sharedInformers) Run(ctx context.Context) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	for informerType, informer := range s.informers {
		if !s.startedInformers[informerType] {
			go informer.Run(ctx.Done())
			s.startedInformers[informerType] = true
		}
	}

	<-ctx.Done()

	return nil
}
