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

package informers_test

import (
	"time"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1_informers "k8s.io/client-go/informers/apps/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	"github.com/ianlewis/controllerutil/informers"
)

func ExampleSharedInformers() {
	config, _ := rest.InClusterConfig()
	client, _ := clientset.NewForConfig(config)

	i := informers.NewSharedInformers()
	i.InformerFor(
		&appsv1.Deployment{},
		func() cache.SharedIndexInformer {
			return appsv1_informers.NewDeploymentInformer(
				client,
				metav1.NamespaceAll,
				12*time.Hour,
				cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
			)
		},
	)
}
