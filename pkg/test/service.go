package test

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/uuid"
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
