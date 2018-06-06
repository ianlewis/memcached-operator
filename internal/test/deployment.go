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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"

	"github.com/ianlewis/memcached-operator/internal/apis/ianlewis.org/v1alpha1"
	"github.com/ianlewis/memcached-operator/internal/controller"
)

func NewMemcachedClusterDeployment(p *v1alpha1.MemcachedProxy, cm *corev1.ConfigMap) *appsv1.Deployment {
	replicas := int32(1)
	d := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			UID:         uuid.NewUUID(),
			Name:        fmt.Sprintf("%s-memcached", p.Name),
			Namespace:   p.Namespace,
			Annotations: make(map[string]string),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(p, v1alpha1.SchemeGroupVersion.WithKind("MemcachedCluster")),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: controller.GetProxyServiceSelector(p),
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "mcrouter",
							Image:   p.Spec.McRouter.Image,
							Command: []string{"mcrouter"},
							Args: []string{
								"-p", fmt.Sprint(*p.Spec.McRouter.Port),
								"--config-file=/etc/mcrouter/config.json",
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "mcrouter",
									ContainerPort: *p.Spec.McRouter.Port,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/etc/mcrouter",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: cm.Name,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return d
}
