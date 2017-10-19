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
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/golang/glog"

	multierror "github.com/hashicorp/go-multierror"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"

	"github.com/IanLewis/memcached-operator/pkg/api/ianlewis.org/v1alpha1"
)

const podIPsAnnotationName = "v1alpha1.ianlewis.org/memcachedPodIPs"
const revisionAnnotationName = "v1alpha1.ianlewis.org/revision"

func (cc *Controller) addConfigMap(obj interface{}) {
	cm := obj.(*v1.ConfigMap)

	// Verify that this is a ConfigMap owned by the cluster.
	c, err := cc.getClusterForObject(cm)
	if err != nil {
		if errors.IsNotFound(err) {
			// MemcachedCluster has been deleted.
			return
		}

		if glog.V(9) {
			glog.Errorf("Error getting cluster for ConfigMap %#v: %#v", cm, err)
		} else {
			glog.Errorf("Error getting cluster for ConfigMap %q: %s", cm.Name, err)
		}
		return
	}

	if c == nil {
		// ConfigMap not owned by a MemcachedCluster
		return
	}

	logObject("Adding ConfigMap: ", cm)

	cc.enqueueCluster(c)
}

func (cc *Controller) updateConfigMap(old, cur interface{}) {
	oldCm := old.(*v1.ConfigMap)
	curCm := cur.(*v1.ConfigMap)

	// Verify that this is a ConfigMap owned by the cluster.
	c, err := cc.getClusterForObject(curCm)
	if err != nil {
		if errors.IsNotFound(err) {
			// MemcachedCluster has been deleted.
			return
		}

		if glog.V(9) {
			glog.Errorf("Error getting cluster for ConfigMap %#v: %#v", curCm, err)
		} else {
			glog.Errorf("Error getting cluster for ConfigMap %q: %s", curCm.Name, err)
		}
		return
	}

	if c == nil {
		// ConfigMap not owned by a MemcachedCluster
		return
	}

	logObject("Updating ConfigMap: ", oldCm, curCm)

	cc.enqueueCluster(c)
}

func (cc *Controller) deleteConfigMap(obj interface{}) {
	cm, ok := obj.(*v1.ConfigMap)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			glog.Errorf("Couldn't get object from tombstone %#v", obj)
			return
		}
		cm, ok = tombstone.Obj.(*v1.ConfigMap)
		if !ok {
			glog.Errorf("Tombstone contained object that is not a ConfigMap %#v", obj)
			return
		}
	}

	// Verify that this is a ConfigMap owned by the cluster.
	c, err := cc.getClusterForObject(cm)
	if err != nil {
		if errors.IsNotFound(err) {
			// MemcachedCluster has been deleted.
			return
		}

		if glog.V(9) {
			glog.Errorf("Error getting cluster for ConfigMap %#v: %#v", cm, err)
		} else {
			glog.Errorf("Error getting cluster for ConfigMap %q: %s", cm.Name, err)
		}
		return
	}

	if c == nil {
		// ConfigMap not owned by a MemcachedCluster
		return
	}

	logObject("Deleting ConfigMap: ", cm)

	cc.enqueueCluster(c)
}

func (cc *Controller) getTwemproxyConfigMapsForCluster(c *v1alpha1.MemcachedCluster) ([]*v1.ConfigMap, error) {
	rawCMList, err := cc.configMapLister.ConfigMaps(c.Namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}

	// Add ConfigMaps who are owned by the given cluster to the result.
	cmList := []*v1.ConfigMap{}
	for _, cm := range rawCMList {
		cmCluster, err := cc.getClusterForObject(cm)
		switch {
		case errors.IsNotFound(err):
			// Cluster for this ConfigMap doesn't exist.
			glog.V(3).Infof("MemcachedCluster for ConfigMap %q doesn't exist.", cm.Name)
			continue
		case err != nil:
			return nil, err
		}

		if cmCluster != nil && c.UID == cmCluster.UID {
			cmList = append(cmList, cm)
		}
	}

	return cmList, nil
}

func (cc *Controller) getLatestTwemproxyConfigMapForCluster(c *v1alpha1.MemcachedCluster) (*v1.ConfigMap, error) {
	cmList, err := cc.getTwemproxyConfigMapsForCluster(c)
	if err != nil {
		return nil, err
	}

	// Start at zero so we can pick up ConfigMaps at revision 1
	latestRev := int64(0)
	var latestCm *v1.ConfigMap
	for _, cm := range cmList {
		rev, err := cc.getRevisionForConfigMap(cm)
		if err != nil {
			glog.V(3).Infof("Failed to get revision for ConfigMap %q: %s", cm.Name, err)
			continue
		}

		if rev > latestRev {
			latestRev = rev
			latestCm = cm
		}
	}

	return latestCm, nil
}

// getRevisionForConfigMap gets the ConfigMaps revision.
func (cc *Controller) getRevisionForConfigMap(cm *v1.ConfigMap) (int64, error) {
	rawRev, ok := cm.Annotations[revisionAnnotationName]
	if !ok {
		return 0, fmt.Errorf("configmap %q has no revision", cm.Name)
	}

	rev, err := strconv.ParseInt(rawRev, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("configmap %q has invalid revision %q", cm.Name, rawRev)
	}

	return rev, nil
}

// getConfigMapData() generates the Twemproxy config.yaml as ConfigMap data
func (cc *Controller) getConfigMapData(c *v1alpha1.MemcachedCluster, podIPs []string) map[string]string {
	configYamlData := struct {
		Hash               string
		Distribution       string
		ServerRetryTimeout int64
		ServerFailureLimit int32
		Servers            []string
	}{
		c.Spec.Proxy.TwemproxyProxySpec.Hash,
		c.Spec.Proxy.TwemproxyProxySpec.Distribution,
		*c.Spec.Proxy.TwemproxyProxySpec.ServerRetryTimeout,
		*c.Spec.Proxy.TwemproxyProxySpec.ServerFailureLimit,
		podIPs,
	}

	buf := bytes.NewBuffer([]byte{})
	cc.configYamlTemplate.Execute(buf, configYamlData)
	return map[string]string{
		"config.yaml": buf.String(),
	}
}

func configMapNameFromCluster(c *v1alpha1.MemcachedCluster, revision int64) string {
	// TODO: Use hash of cluster name and collision count
	return makeName(fmt.Sprintf("%s-twemproxy", c.Name), []string{c.Name, strconv.FormatInt(revision, 10)})
}
func configMapNameFromClusterKey(key string, revision int64) (string, error) {
	_, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return "", err
	}
	return makeName(fmt.Sprintf("%s-twemproxy", name), []string{name, strconv.FormatInt(revision, 10)}), nil
}

// syncTwemproxyConfigMap creates a new ConfigMap if necessary and saves it to the Kubernetes API.
func (cc *Controller) syncTwemproxyConfigMap(c *v1alpha1.MemcachedCluster) error {
	// TODO: Periodically garbage collect old ConfigMaps
	cm, err := cc.getLatestTwemproxyConfigMapForCluster(c)
	if err != nil {
		return fmt.Errorf("Error getting latest ConfigMap for MemcachedCluster %q: %v", c.Name, err)
	}

	glog.V(9).Infof("Syncing Twemproxy ConfigMap %#v", cm)

	d, err := cc.getMemcachedDeploymentForCluster(c)
	if err != nil {
		return fmt.Errorf("failed to get Deployment for MemcachedCluster %q: %v", c.Name, err)
	}

	podIPs, err := cc.getMemcachedPodIPs(d)
	if err != nil {
		return fmt.Errorf("failed getting Pod IPs for cluster %q: %v", c.Name, err)
	}

	if twemproxyConfigMapNeedsUpdate(cm, podIPs) {
		// If the ConfigMap doesn't exist yet then the rev is incremented to 1 below.
		rev := int64(0)
		if cm != nil {
			rev, err = cc.getRevisionForConfigMap(cm)
			if err != nil {
				return fmt.Errorf("failed to get revision for Twemproxy ConfigMap %q: %v", cm.Name, err)
			}
		}
		rev = rev + 1

		configMap := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				// Increment the revision for the ConfigMap
				Name:            configMapNameFromCluster(c, rev),
				Namespace:       c.Namespace,
				OwnerReferences: []metav1.OwnerReference{*newClusterOwnerRef(c)},
				Annotations: map[string]string{
					// Set Pod IPs to an annotation so it can be easily compared later.
					// TODO: Sync our own EndPoints object?
					podIPsAnnotationName:   strings.Join(podIPs, ","),
					revisionAnnotationName: strconv.FormatInt(rev, 10),
				},
			},
			Data: cc.getConfigMapData(c, podIPs),
		}

		_, err := cc.client.CoreV1().ConfigMaps(c.Namespace).Create(configMap)
		if err != nil {
			return fmt.Errorf("failed to create new ConfigMap for MemcachedCluster %q: %v", c.Name, err)
		}
	}

	return nil
}

// getMemcachedPodIPs returns a sorted list of Pod IPs for ready Pods. Non-ready pods are excluded.
func (cc *Controller) getMemcachedPodIPs(d *v1beta1.Deployment) ([]string, error) {
	selector, err := metav1.LabelSelectorAsSelector(d.Spec.Selector)
	if err != nil {
		return nil, fmt.Errorf("invalid label selector: %v", err)
	}

	pods, err := cc.podLister.Pods(d.Namespace).List(selector)
	if err != nil {
		return nil, err
	}

	podIPs := []string{}

	for _, pod := range pods {
		// Pod may still be Pending so it may not yet have an IP.
		// Also filter out Pods that are being deleted.
		if pod.Status.PodIP == "" || pod.DeletionTimestamp != nil {
			continue
		}

		podIPs = append(podIPs, strings.Trim(pod.Status.PodIP, " "))
	}
	sort.Strings(podIPs)
	return podIPs, nil
}

func twemproxyConfigMapNeedsUpdate(cm *v1.ConfigMap, newPodIPs []string) bool {
	// If the ConfigMap doesn't exist return true indicating a ConfigMap needs
	// to be created.
	if cm == nil {
		return true
	}

	oldPodIPs := podIPsForConfigMap(cm)
	if len(oldPodIPs) != len(newPodIPs) {
		return true
	}
	for i := range oldPodIPs {
		if newPodIPs[i] != oldPodIPs[i] {
			return true
		}
	}
	return false
}

func podIPsForConfigMap(cm *v1.ConfigMap) []string {
	rawPodIPs := strings.Split(cm.Annotations[podIPsAnnotationName], ",")

	podIPs := []string{}
	for _, rawIP := range rawPodIPs {
		ip := strings.Trim(rawIP, " ")
		if ip != "" {
			podIPs = append(podIPs, ip)
		}
	}
	sort.Strings(podIPs)
	return podIPs
}

func (cc *Controller) deleteTwemproxyConfigMaps(key string) error {
	namespace, clusterName, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	rawCMList, err := cc.configMapLister.ConfigMaps(namespace).List(labels.Everything())
	if err != nil {
		return err
	}

	// Add ConfigMaps who are owned by the given cluster to the result.
	cmList := []*v1.ConfigMap{}
	for _, cm := range rawCMList {
		ref := getOwnerRefOfKind(cm, v1alpha1.SchemeGroupVersion, "MemcachedCluster")
		if ref != nil && ref.Name == clusterName {
			cmList = append(cmList, cm)
		}
	}

	// Attempt to delete all ConfigMaps but return an error if any one fails.
	var result *multierror.Error
	for _, cm := range cmList {
		// Delete the Twemproxy ConfigMap.
		if err := cc.client.CoreV1().ConfigMaps(cm.Namespace).Delete(cm.Name, nil); err != nil {
			result = multierror.Append(result, fmt.Errorf("failed to delete Twemproxy ConfigMap %q: %v", cm.Name, err))
		}
		glog.V(2).Infof("ConfigMap %q deleted", cm.Name)
	}

	return result.ErrorOrNil()
}
