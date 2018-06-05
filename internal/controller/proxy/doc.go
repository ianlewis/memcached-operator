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

package proxy

// The proxy controller is responsible for watching for new`MemcachedProxy`
// objects and applying default values to the objects. Because custom resource
// definitions don't currently have the ability to automatically apply default
// values and structure to objects, it must be done by this controller. This makes
// it easier for users to create new objects because they don't have to fill in
// each and every field in the object.
//
// The proxy controller is also responsible for examining the cluster state and
// updating the status for each MemcachedProxy object.
