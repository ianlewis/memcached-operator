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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/uuid"

	"github.com/ianlewis/memcached-operator/internal/apis/ianlewis.org/v1alpha1"
	"github.com/ianlewis/memcached-operator/internal/controller"
)

func NewMemcachedService(name string) (*corev1.Service, *corev1.Endpoints) {
	s := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			UID:         uuid.NewUUID(),
			Name:        name,
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "memcached",
			},
			Type: "ClusterIP",
			Ports: []corev1.ServicePort{
				{
					Name:       "memcached",
					Protocol:   "TCP",
					Port:       11211,
					TargetPort: intstr.FromInt(11211),
				},
			},
		},
	}

	ep := &corev1.Endpoints{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			UID:         uuid.NewUUID(),
			Name:        name,
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
		},
	}

	return s, ep
}

func NewMemcachedClusterService(p *v1alpha1.MemcachedProxy) *corev1.Service {
	s := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			UID:         uuid.NewUUID(),
			Name:        fmt.Sprintf("%s-memcached", p.Name),
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(p, v1alpha1.SchemeGroupVersion.WithKind("MemcachedCluster")),
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: controller.GetProxyServiceSelector(p),
			Type:     "ClusterIP",
			Ports: []corev1.ServicePort{
				{
					Name:       "mcrouter",
					Protocol:   "TCP",
					Port:       11211,
					TargetPort: intstr.FromInt(11211),
				},
			},
		},
	}

	return s
}
