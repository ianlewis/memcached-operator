// Copyright 2017 Google Inc.
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

package memcachedcluster

import (
	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
)

func (cc *Controller) addPod(obj interface{}) {
	pod := obj.(*v1.Pod)

	// Verify ownership
	c, err := cc.getClusterForPod(pod)
	switch {
	case errors.IsNotFound(err):
		// MemcachedCluster has been deleted.
		return
	case err != nil:
		if glog.V(9) {
			glog.Errorf("Error getting cluster for Pod %#v: %#v", pod, err)
		} else {
			glog.Errorf("Error getting cluster for Pod %q: %v", pod.Name, err)
		}
	}

	if c == nil {
		// Pod not owned by a MemcachedCluster
		return
	}

	logObject("Adding Pod: ", pod)

	cc.enqueueCluster(c)
}

func (cc *Controller) updatePod(old, cur interface{}) {
	oldPod := old.(*v1.Pod)
	curPod := cur.(*v1.Pod)

	// Verify that this is a deployment owned by the cluster.
	c, err := cc.getClusterForPod(curPod)
	switch {
	case errors.IsNotFound(err):
		// MemcachedCluster has been deleted.
		return
	case err != nil:
		if glog.V(9) {
			glog.Errorf("Error getting cluster for Pod %#v: %#v", curPod, err)
		} else {
			glog.Errorf("Error getting cluster for Pod %q: %v", curPod.Name, err)
		}
		return
	}

	if c == nil {
		// Pod not owned by a MemcachedCluster
		return
	}

	logObject("Updating Pod: ", oldPod, curPod)

	cc.enqueueCluster(c)
}

func (cc *Controller) deletePod(obj interface{}) {
	pod, ok := obj.(*v1.Pod)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			glog.Errorf("Couldn't get object from tombstone %#v", obj)
			return
		}
		pod, ok = tombstone.Obj.(*v1.Pod)
		if !ok {
			glog.Errorf("Tombstone contained object that is not a Pod %#v", obj)
			return
		}
	}

	// Verify ownership
	c, err := cc.getClusterForPod(pod)
	switch {
	case errors.IsNotFound(err):
		// MemcachedCluster has been deleted.
		return
	case err != nil:
		if glog.V(9) {
			glog.Errorf("Error getting cluster for Pod %#v: %#v", pod, err)
		} else {
			glog.Errorf("Error getting cluster for Pod %q: %v", pod.Name, err)
		}
	}

	if c == nil {
		// Pod not owned by a MemcachedCluster
		return
	}

	logObject("Deleting Pod: ", pod)

	cc.enqueueCluster(c)
}
