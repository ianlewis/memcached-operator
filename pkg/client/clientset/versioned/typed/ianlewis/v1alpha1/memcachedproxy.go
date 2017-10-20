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
	v1alpha1 "github.com/IanLewis/memcached-operator/pkg/apis/ianlewis.org/v1alpha1"
	scheme "github.com/IanLewis/memcached-operator/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// MemcachedProxiesGetter has a method to return a MemcachedProxyInterface.
// A group's client should implement this interface.
type MemcachedProxiesGetter interface {
	MemcachedProxies(namespace string) MemcachedProxyInterface
}

// MemcachedProxyInterface has methods to work with MemcachedProxy resources.
type MemcachedProxyInterface interface {
	Create(*v1alpha1.MemcachedProxy) (*v1alpha1.MemcachedProxy, error)
	Update(*v1alpha1.MemcachedProxy) (*v1alpha1.MemcachedProxy, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.MemcachedProxy, error)
	List(opts v1.ListOptions) (*v1alpha1.MemcachedProxyList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.MemcachedProxy, err error)
	MemcachedProxyExpansion
}

// memcachedProxies implements MemcachedProxyInterface
type memcachedProxies struct {
	client rest.Interface
	ns     string
}

// newMemcachedProxies returns a MemcachedProxies
func newMemcachedProxies(c *IanlewisV1alpha1Client, namespace string) *memcachedProxies {
	return &memcachedProxies{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the memcachedProxy, and returns the corresponding memcachedProxy object, and an error if there is any.
func (c *memcachedProxies) Get(name string, options v1.GetOptions) (result *v1alpha1.MemcachedProxy, err error) {
	result = &v1alpha1.MemcachedProxy{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("memcachedproxies").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of MemcachedProxies that match those selectors.
func (c *memcachedProxies) List(opts v1.ListOptions) (result *v1alpha1.MemcachedProxyList, err error) {
	result = &v1alpha1.MemcachedProxyList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("memcachedproxies").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested memcachedProxies.
func (c *memcachedProxies) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("memcachedproxies").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a memcachedProxy and creates it.  Returns the server's representation of the memcachedProxy, and an error, if there is any.
func (c *memcachedProxies) Create(memcachedProxy *v1alpha1.MemcachedProxy) (result *v1alpha1.MemcachedProxy, err error) {
	result = &v1alpha1.MemcachedProxy{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("memcachedproxies").
		Body(memcachedProxy).
		Do().
		Into(result)
	return
}

// Update takes the representation of a memcachedProxy and updates it. Returns the server's representation of the memcachedProxy, and an error, if there is any.
func (c *memcachedProxies) Update(memcachedProxy *v1alpha1.MemcachedProxy) (result *v1alpha1.MemcachedProxy, err error) {
	result = &v1alpha1.MemcachedProxy{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("memcachedproxies").
		Name(memcachedProxy.Name).
		Body(memcachedProxy).
		Do().
		Into(result)
	return
}

// Delete takes name of the memcachedProxy and deletes it. Returns an error if one occurs.
func (c *memcachedProxies) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("memcachedproxies").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *memcachedProxies) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("memcachedproxies").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched memcachedProxy.
func (c *memcachedProxies) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.MemcachedProxy, err error) {
	result = &v1alpha1.MemcachedProxy{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("memcachedproxies").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
