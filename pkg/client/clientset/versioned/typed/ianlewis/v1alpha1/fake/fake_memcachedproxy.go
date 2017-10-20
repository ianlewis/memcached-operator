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

package fake

import (
	v1alpha1 "github.com/IanLewis/memcached-operator/pkg/apis/ianlewis.org/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeMemcachedProxies implements MemcachedProxyInterface
type FakeMemcachedProxies struct {
	Fake *FakeIanlewisV1alpha1
	ns   string
}

var memcachedproxiesResource = schema.GroupVersionResource{Group: "ianlewis.org", Version: "v1alpha1", Resource: "memcachedproxies"}

var memcachedproxiesKind = schema.GroupVersionKind{Group: "ianlewis.org", Version: "v1alpha1", Kind: "MemcachedProxy"}

// Get takes name of the memcachedProxy, and returns the corresponding memcachedProxy object, and an error if there is any.
func (c *FakeMemcachedProxies) Get(name string, options v1.GetOptions) (result *v1alpha1.MemcachedProxy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(memcachedproxiesResource, c.ns, name), &v1alpha1.MemcachedProxy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MemcachedProxy), err
}

// List takes label and field selectors, and returns the list of MemcachedProxies that match those selectors.
func (c *FakeMemcachedProxies) List(opts v1.ListOptions) (result *v1alpha1.MemcachedProxyList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(memcachedproxiesResource, memcachedproxiesKind, c.ns, opts), &v1alpha1.MemcachedProxyList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.MemcachedProxyList{}
	for _, item := range obj.(*v1alpha1.MemcachedProxyList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested memcachedProxies.
func (c *FakeMemcachedProxies) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(memcachedproxiesResource, c.ns, opts))

}

// Create takes the representation of a memcachedProxy and creates it.  Returns the server's representation of the memcachedProxy, and an error, if there is any.
func (c *FakeMemcachedProxies) Create(memcachedProxy *v1alpha1.MemcachedProxy) (result *v1alpha1.MemcachedProxy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(memcachedproxiesResource, c.ns, memcachedProxy), &v1alpha1.MemcachedProxy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MemcachedProxy), err
}

// Update takes the representation of a memcachedProxy and updates it. Returns the server's representation of the memcachedProxy, and an error, if there is any.
func (c *FakeMemcachedProxies) Update(memcachedProxy *v1alpha1.MemcachedProxy) (result *v1alpha1.MemcachedProxy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(memcachedproxiesResource, c.ns, memcachedProxy), &v1alpha1.MemcachedProxy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MemcachedProxy), err
}

// Delete takes name of the memcachedProxy and deletes it. Returns an error if one occurs.
func (c *FakeMemcachedProxies) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(memcachedproxiesResource, c.ns, name), &v1alpha1.MemcachedProxy{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeMemcachedProxies) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(memcachedproxiesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.MemcachedProxyList{})
	return err
}

// Patch applies the patch and returns the patched memcachedProxy.
func (c *FakeMemcachedProxies) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.MemcachedProxy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(memcachedproxiesResource, c.ns, name, data, subresources...), &v1alpha1.MemcachedProxy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MemcachedProxy), err
}
