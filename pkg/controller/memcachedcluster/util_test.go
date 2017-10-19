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

package memcachedcluster

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type testObject struct {
	metav1.ObjectMeta
}

func assertOwnerRef(t *testing.T, got *metav1.OwnerReference, expected *metav1.OwnerReference) {
	if got == nil && expected == nil {
		return
	}

	if got != nil && expected == nil {
		t.Fatalf("Expected no OwnerReference but got OwnerReference with UID %q", got.UID)
	}
	if got == nil && expected != nil {
		t.Fatalf("Expected OwnerReference with UID %q but no OwnerReference returned", expected.UID)
	}
	if got.UID != expected.UID {
		t.Errorf("Expected OwnerReference with UID %q but got %q", expected.UID, got.UID)
	}
}

func TestGetOwnerRefOfKind(t *testing.T) {
	fixtures := []struct {
		name       string
		obj        metav1.Object
		apiVersion schema.GroupVersion
		kind       string
		expected   *metav1.OwnerReference
	}{
		{
			name: "no owner references",
			obj: &testObject{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "test",
					Namespace:       "default",
					OwnerReferences: []metav1.OwnerReference{},
				},
			},
			apiVersion: schema.GroupVersion{Group: "extensions", Version: "v1beta1"},
			kind:       "Deployment",
			expected:   nil,
		},
		{
			name: "single references",
			obj: &testObject{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
					OwnerReferences: []metav1.OwnerReference{
						metav1.OwnerReference{
							APIVersion: "extensions/v1beta1",
							Kind:       "Deployment",
							Name:       "testdeployment",
							UID:        "123e4567-e89b-12d3-a456-426655440000",
						},
					},
				},
			},
			apiVersion: schema.GroupVersion{Group: "extensions", Version: "v1beta1"},
			kind:       "Deployment",
			expected: &metav1.OwnerReference{
				APIVersion: "extensions/v1beta1",
				Kind:       "Deployment",
				Name:       "testdeploy",
				UID:        "123e4567-e89b-12d3-a456-426655440000",
			},
		},
		{
			name: "multipe references",
			obj: &testObject{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
					OwnerReferences: []metav1.OwnerReference{
						metav1.OwnerReference{
							APIVersion: "extensions/v1beta1",
							Kind:       "Deployment",
							Name:       "testdeployment",
							UID:        "123e4567-e89b-12d3-a456-426655440000",
						},
						metav1.OwnerReference{
							APIVersion: "v1",
							Kind:       "ConfigMap",
							Name:       "configmap",
							UID:        "00112233-4455-6677-8899-aabbccddeeff",
						},
					},
				},
			},
			apiVersion: schema.GroupVersion{Group: "extensions", Version: "v1beta1"},
			kind:       "Deployment",
			expected: &metav1.OwnerReference{
				APIVersion: "extensions/v1beta1",
				Kind:       "Deployment",
				Name:       "testdeployment",
				UID:        "123e4567-e89b-12d3-a456-426655440000",
			},
		},
	}

	for _, f := range fixtures {
		t.Run(f.name, func(t *testing.T) {
			ref := getOwnerRefOfKind(f.obj, f.apiVersion, f.kind)
			assertOwnerRef(t, ref, f.expected)
		})
	}
}
