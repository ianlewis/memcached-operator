package test

import (
	"fmt"
	"math/rand"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ianlewis/memcached-operator/pkg/apis/ianlewis.org/v1alpha1"
)

func NewProxyConfigMap(p *v1alpha1.MemcachedProxy) *corev1.ConfigMap {
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			// Generate a new unique name on the API server side
			Name:      fmt.Sprintf("%s-config-%d", p.Name, r1.Intn(10000)),
			Namespace: p.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(p, v1alpha1.SchemeGroupVersion.WithKind("MemcachedProxy")),
			},
		},
	}

	return cm
}
