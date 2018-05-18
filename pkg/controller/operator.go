// Copyright 2017 Google LLC
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

package controller

import (
	"fmt"
	"sort"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	appsv1listers "k8s.io/client-go/listers/apps/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"

	"github.com/ianlewis/memcached-operator/pkg/apis/ianlewis.org/v1alpha1"
)

// GetDeploymentsForProxy returns the deployment for the given proxy as well as a list of any other deployments owned by the proxy. Under normal circumstances this list is empty.
func GetDeploymentsForProxy(dLister appsv1listers.DeploymentLister, p *v1alpha1.MemcachedProxy) (*appsv1.Deployment, []*appsv1.Deployment, error) {
	dList, err := dLister.Deployments(p.Namespace).List(labels.Everything())
	if err != nil {
		return nil, []*appsv1.Deployment{}, nil
	}

	var result []*appsv1.Deployment
	for _, d := range dList {
		if metav1.IsControlledBy(d, p) {
			result = append(result, d)
		}
	}

	// Sort deployments by creation timestamp in reverse order.
	sort.Slice(result, func(i, j int) bool {
		return result[j].CreationTimestamp.UTC().Before(result[i].CreationTimestamp.UTC())
	})

	if len(result) == 0 {
		return nil, []*appsv1.Deployment{}, nil
	}

	return result[0], result[1:], nil
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
