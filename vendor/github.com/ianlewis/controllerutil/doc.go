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

// Package controllerutil implements a ControllerManager type that is
// used to manage and run several Kubernetes controllers as part of a single
// process.
//
// Controller are registered with the ControllerManager via the Register method
// and the group of controllers are run using the Run method on the
// ControllerManager.
package controllerutil
