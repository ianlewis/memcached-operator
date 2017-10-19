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
	"k8s.io/apimachinery/pkg/runtime"
)

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	scheme.AddTypeDefaultingFunc(&MemcachedCluster{}, func(obj interface{}) {
		SetMemcachedClusterDefaults(obj.(*MemcachedCluster))
	})
	return nil
}

// SetMemcachedClusterDefaults sets the defaults for a MemcachedCluster object
func SetMemcachedClusterDefaults(obj *MemcachedCluster) {
	if obj.Spec.Replicas == nil {
		obj.Spec.Replicas = new(int32)
		*obj.Spec.Replicas = 1
	}

	if obj.Spec.MemcachedImage == "" {
		obj.Spec.MemcachedImage = "memcached:1.4.36-alpine"
	}

	if obj.Spec.Proxy == nil {
		obj.Spec.Proxy = &MemcachedProxySpec{}
	}

	SetMemcachedClusterProxySpecDefaults(obj.Spec.Proxy)
}

func SetMemcachedClusterProxySpecDefaults(p *MemcachedProxySpec) {
	if p.TwemproxyProxySpec == nil {
		p.TwemproxyProxySpec = &TwemproxySpec{}
	}

	if p.TwemproxyProxySpec.Image == "" {
		p.TwemproxyProxySpec.Image = "asia.gcr.io/ian-corp-dev/twemproxy:v0.4.1-0"
	}

	if p.TwemproxyProxySpec.Hash == "" {
		p.TwemproxyProxySpec.Hash = "fnv1a_64"
	}

	if p.TwemproxyProxySpec.Distribution == "" {
		p.TwemproxyProxySpec.Distribution = "ketama"
	}

	if p.TwemproxyProxySpec.ServerRetryTimeout == nil {
		p.TwemproxyProxySpec.ServerRetryTimeout = new(int64)
		*p.TwemproxyProxySpec.ServerRetryTimeout = 2000
	}

	if p.TwemproxyProxySpec.ServerFailureLimit == nil {
		p.TwemproxyProxySpec.ServerFailureLimit = new(int32)
		*p.TwemproxyProxySpec.ServerFailureLimit = 1
	}
}
