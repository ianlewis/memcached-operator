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

package controllerutil

import (
	"context"

	"golang.org/x/sync/errgroup"

	"k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"

	// Required for Auth with GKE/Azure/OIDC
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/ianlewis/controllerutil/controller"
	"github.com/ianlewis/controllerutil/informers"
	"github.com/ianlewis/controllerutil/logging"
)

// ControllerManager manages a set of controllers registered to it. It manages their lifecycle and the lifecycle of informers that controllers use.
type ControllerManager struct {
	name        string
	client      clientset.Interface
	controllers map[string]controller.Constructor

	// INFO is a wrapper around glog that allows the easy creation of loggers at certain log levels.
	l *logging.Logger
}

// New creates a new controller manager.
func NewControllerManager(name string, client clientset.Interface) *ControllerManager {
	return &ControllerManager{
		name:        name,
		client:      client,
		controllers: make(map[string]controller.Constructor),

		l: logging.New("[" + name + "] "),
	}
}

// Register registers a controller created by the given constructor by the given name. The given name should be unique to the controller and is used in Kubernetes event recorders and logging.
func (m *ControllerManager) Register(name string, c controller.Constructor) {
	m.controllers[name] = c
}

// Run starts all controllers and informers registered to the ControllerManager instance. ControllerManager assumes all registered controllers are essential to proper functioning. ControllerManager cancels all controllers and returns the first error returned by the controllers in the event of a failure. ControllerManager will not attempt to restart controllers and will simply return. As a best practice the calling process should log the returned error message and exit. It is assumed that the controller manager will be run by a supervisor process like supervisord or the Kubernetes kubelet and will be restarted gracefully if fatal errors occur.
//
// The ControllerManager starts shared informers and waits for their caches to be synced before starting controllers. Controllers do not need to wait for informers to sync.
func (m *ControllerManager) Run(ctx context.Context) error {
	m.l.Info.V(4).Print("starting controller manager")

	wg, ctx := errgroup.WithContext(ctx)

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(m.l.Info.V(4).Printf)
	eventBroadcaster.StartRecordingToSink(&corev1.EventSinkImpl{Interface: m.client.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: m.name})

	sharedInformers := informers.NewSharedInformers()

	// First create all controllers and register shared informers
	controllers := make(map[string]controller.Interface)
	for name, fn := range m.controllers {
		context := &controller.Context{
			Client:          m.client,
			SharedInformers: sharedInformers,

			Recorder: recorder,

			Logger: logging.New("[" + name + "] "),
		}
		controllers[name] = fn(context)
	}

	// Start shared informers that were registered by controller constructors.
	wg.Go(func() error {
		return sharedInformers.Run(ctx)
	})

	m.l.Info.V(4).Print("waiting for cache sync")
	if err := sharedInformers.WaitForCacheSync(ctx); err != nil {
		return err
	}

	// Start all controllers
	for name, c := range controllers {
		// wg.Go does not allow us to define parameters to the goroutine function so we
		// encapsulate the wg.Go call in a function so that we can capture the necessary variables.
		// See: https://golang.org/doc/faq#closures_and_goroutines
		func(name string, c controller.Interface) {
			wg.Go(func() error {
				defer m.l.Info.V(4).Printf("controller %q stopped", name)
				m.l.Info.V(4).Printf("starting controller %q", name)
				return c.Run(ctx)
			})
		}(name, c)
	}

	m.l.Info.V(4).Print("controller manager started")

	return wg.Wait()
}
