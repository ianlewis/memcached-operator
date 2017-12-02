package controller

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	corev1listers "k8s.io/client-go/listers/core/v1"
	v1beta1listers "k8s.io/client-go/listers/extensions/v1beta1"

	"github.com/ianlewis/memcached-operator/pkg/apis/ianlewis.org/v1alpha1"
)

// GetDeploymentsForProxy returns all configmaps owned by the proxy
func GetDeploymentsForProxy(dLister v1beta1listers.DeploymentLister, p *v1alpha1.MemcachedProxy) ([]*v1beta1.Deployment, error) {
	dList, err := dLister.Deployments(p.Namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}

	var result []*v1beta1.Deployment
	for _, d := range dList {
		if metav1.IsControlledBy(d, p) {
			result = append(result, d)
		}
	}
	return result, nil
}

// GetConfigMapForProxy returns the configmap owned by the proxy or an error if multiple are found.
func GetConfigMapForProxy(cmLister corev1listers.ConfigMapLister, p *v1alpha1.MemcachedProxy) (*corev1.ConfigMap, error) {
	cmList, err := GetConfigMapsForProxy(cmLister, p)
	if err != nil {
		return nil, err
	}

	if len(cmList) > 1 {
		return nil, fmt.Errorf("found multiple configmaps for %q", p.Namespace+"/"+p.Name)
	}

	if len(cmList) == 0 {
		return nil, fmt.Errorf("configmap for %q not found", p.Namespace+"/"+p.Name)
	}

	return cmList[0], nil
}

// GetConfigMapsForProxy returns all configmaps owned by the proxy
func GetConfigMapsForProxy(cmLister corev1listers.ConfigMapLister, p *v1alpha1.MemcachedProxy) ([]*corev1.ConfigMap, error) {
	cmList, err := cmLister.ConfigMaps(p.Namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}

	var result []*corev1.ConfigMap
	for _, cm := range cmList {
		if metav1.IsControlledBy(cm, p) {
			result = append(result, cm)
		}
	}
	return result, nil
}
