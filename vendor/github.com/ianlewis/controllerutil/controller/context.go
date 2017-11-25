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

package controller

import (
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"

	"github.com/ianlewis/controllerutil/informers"
	"github.com/ianlewis/controllerutil/logging"
)

// Context defines a controller context. Each controller is given a context in it's constructor which provides it with various objects set up my the controller manager.
type Context struct {
	// An instance of a Kubernetes API client
	Client clientset.Interface

	// SharedInformers allows controllers to register shared informer objects.
	SharedInformers informers.SharedInformers

	// Recorder is used for recording Kubernetes events
	Recorder record.EventRecorder

	// Logger is a logger that can be used by the controller to log info and error messages
	Logger *logging.Logger
}
