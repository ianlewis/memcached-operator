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
package fake

import (
	example_com_v1 "github.com/ianlewis/controllerutil/example/pkg/apis/example.com/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeFoos implements FooInterface
type FakeFoos struct {
	Fake *FakeExampleV1
	ns   string
}

var foosResource = schema.GroupVersionResource{Group: "example.com", Version: "v1", Resource: "foos"}

var foosKind = schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "Foo"}

// Get takes name of the foo, and returns the corresponding foo object, and an error if there is any.
func (c *FakeFoos) Get(name string, options v1.GetOptions) (result *example_com_v1.Foo, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(foosResource, c.ns, name), &example_com_v1.Foo{})

	if obj == nil {
		return nil, err
	}
	return obj.(*example_com_v1.Foo), err
}

// List takes label and field selectors, and returns the list of Foos that match those selectors.
func (c *FakeFoos) List(opts v1.ListOptions) (result *example_com_v1.FooList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(foosResource, foosKind, c.ns, opts), &example_com_v1.FooList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &example_com_v1.FooList{}
	for _, item := range obj.(*example_com_v1.FooList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested foos.
func (c *FakeFoos) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(foosResource, c.ns, opts))

}

// Create takes the representation of a foo and creates it.  Returns the server's representation of the foo, and an error, if there is any.
func (c *FakeFoos) Create(foo *example_com_v1.Foo) (result *example_com_v1.Foo, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(foosResource, c.ns, foo), &example_com_v1.Foo{})

	if obj == nil {
		return nil, err
	}
	return obj.(*example_com_v1.Foo), err
}

// Update takes the representation of a foo and updates it. Returns the server's representation of the foo, and an error, if there is any.
func (c *FakeFoos) Update(foo *example_com_v1.Foo) (result *example_com_v1.Foo, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(foosResource, c.ns, foo), &example_com_v1.Foo{})

	if obj == nil {
		return nil, err
	}
	return obj.(*example_com_v1.Foo), err
}

// Delete takes name of the foo and deletes it. Returns an error if one occurs.
func (c *FakeFoos) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(foosResource, c.ns, name), &example_com_v1.Foo{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeFoos) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(foosResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &example_com_v1.FooList{})
	return err
}

// Patch applies the patch and returns the patched foo.
func (c *FakeFoos) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *example_com_v1.Foo, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(foosResource, c.ns, name, data, subresources...), &example_com_v1.Foo{})

	if obj == nil {
		return nil, err
	}
	return obj.(*example_com_v1.Foo), err
}
