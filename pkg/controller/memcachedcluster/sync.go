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
	"fmt"
	"reflect"
	"time"

	"github.com/golang/glog"

	multierror "github.com/hashicorp/go-multierror"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"

	"github.com/IanLewis/memcached-operator/pkg/api/ianlewis.org/v1alpha1"
)

// Syncs a MemcachedCluster. This method is not meant to be run concurrently for a single key.
func (cc *Controller) sync(key string) error {
	// TODO: Add cluster status and mechanism to update cluster objects

	startTime := time.Now()
	glog.V(2).Infof("Started syncing MemcachedCluster for key %q (%v)", key, startTime)
	defer func() {
		glog.V(2).Infof("Finished syncing MemcachedCluster for key %q (%v)", key, time.Now().Sub(startTime))
	}()

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	cOrig, err := cc.clusterLister.MemcachedClusters(namespace).Get(name)
	if errors.IsNotFound(err) {
		glog.V(2).Infof("MemcachedCluster for key %q has been deleted", key)
		// NOTE: As of 1.7, Garbage Collection is not supported for ThirdPartyResources or CustomResourceDefinitions.
		// We will have to clean up the cluster here since TPRs and CRDs are not supported for Garbage Collection.
		err = cc.teardownCluster(key)
		if err != nil {
			return err
		}
		return nil
	}
	if err != nil {
		return err
	}

	c, err := clusterDeepCopy(cOrig)
	if err != nil {
		return err
	}

	glog.V(9).Infof("Syncing MemcachedCluster %#v", c)

	// FIXME: Currently defaults not working w/ API client.
	v1alpha1.SetMemcachedClusterDefaults(c)
	if !reflect.DeepEqual(c, cOrig) {
		glog.V(2).Infof("Applying MemcachedCluster defaults for key %q", key)
		cc.syncMemcachedCluster(c)
	}

	// Sync the cluster's owned objects.
	// If errors occur then continue, but
	// collect errors and return them.
	var result *multierror.Error

	if err = cc.syncTwemproxyDeployment(c); err != nil {
		result = multierror.Append(result, err)
	}

	if err = cc.syncMemcachedDeployment(c); err != nil {
		result = multierror.Append(result, err)
	}

	if err = cc.syncTwemproxyConfigMap(c); err != nil {
		result = multierror.Append(result, err)
	}

	// Sync cluster status.
	if err = cc.syncMemcachedClusterStatus(c); err != nil {
		multierror.Append(result, err)
	}

	return result.ErrorOrNil()
}

func clusterDeepCopy(c *v1alpha1.MemcachedCluster) (*v1alpha1.MemcachedCluster, error) {
	objCopy, err := scheme.Scheme.DeepCopy(c)
	if err != nil {
		return nil, err
	}
	copied, ok := objCopy.(*v1alpha1.MemcachedCluster)
	if !ok {
		return nil, fmt.Errorf("expected MemcachedCluster , got %#v", objCopy)
	}
	return copied, nil
}
