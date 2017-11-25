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

package controllerutil_test

import (
	"context"
	"log"

	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"

	"github.com/ianlewis/controllerutil"
	"github.com/ianlewis/controllerutil/controller"
	"github.com/ianlewis/controllerutil/logging"
)

func ExampleControllerManager() {
	config, _ := rest.InClusterConfig()
	client, _ := clientset.NewForConfig(config)
	m := controllerutil.New("hoge", client)

	err := m.Run(context.Background())
	if err != nil {
		// handle error
	}
}

type HogeController struct {
	client      clientset.Interface
	recorder    record.EventRecorder
	infoLogger  *logging.Logger
	errorLogger *log.Logger
}

func (c *HogeController) Run(ctx context.Context) error { return nil }

func ExampleControllerManager_Register() {
	config, _ := rest.InClusterConfig()
	client, _ := clientset.NewForConfig(config)
	m := controllerutil.New("hoge", client)

	// Register the hoge controller. ctx is a controller context.
	m.Register("hoge", func(ctx *controller.Context) controller.Interface {
		return &HogeController{
			// ctx.Client is the same client passed to controllerutil.New
			client: ctx.Client,
			// ctx.Recorder is used for recording Kubernetes events.
			recorder: ctx.Recorder,
			// ctx.InfoLogger is a convenient wrapper used for logging.
			infoLogger: ctx.InfoLogger,
			// ctx.ErrorLogger is a convenient wrapper used for error logging.
			errorLogger: ctx.ErrorLogger,
		}
	})
}
