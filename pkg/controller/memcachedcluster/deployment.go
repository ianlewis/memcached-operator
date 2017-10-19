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

	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"

	"github.com/IanLewis/memcached-operator/pkg/api/ianlewis.org/v1alpha1"
)

func (cc *Controller) addDeployment(obj interface{}) {
	d := obj.(*v1beta1.Deployment)

	// Verify that this is a deployment owned by the cluster.
	c, err := cc.getClusterForObject(d)
	if err != nil {
		if errors.IsNotFound(err) {
			// MemcachedCluster has been deleted.
			return
		}

		if glog.V(9) {
			glog.Errorf("Error getting cluster for deployment %#v: %#v", d, err)
		} else {
			glog.Errorf("Error getting cluster for deployment %q: %s", d.Name, err)
		}
		return
	}

	if c == nil {
		// Deployment not owned by a MemcachedCluster
		return
	}

	logObject("Adding deployment: ", d)

	cc.enqueueCluster(c)
}

func (cc *Controller) updateDeployment(old, cur interface{}) {
	oldD := old.(*v1beta1.Deployment)
	curD := cur.(*v1beta1.Deployment)

	// TODO: Sync old cluster if cluster owner changed?

	// Verify that this is a deployment owned by the cluster.
	c, err := cc.getClusterForObject(curD)
	if err != nil {
		if errors.IsNotFound(err) {
			// MemcachedCluster has been deleted.
			return
		}

		if glog.V(9) {
			glog.Errorf("Error getting cluster for deployment %#v: %#v", curD, err)
		} else {
			glog.Errorf("Error getting cluster for deployment %q: %s", curD.Name, err)
		}
		return
	}

	if c == nil {
		// Deployment not owned by a MemcachedCluster
		return
	}

	logObject("Updating deployment: ", oldD, curD)

	cc.enqueueCluster(c)
}

func (cc *Controller) deleteDeployment(obj interface{}) {
	d, ok := obj.(*v1beta1.Deployment)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			glog.Errorf("Couldn't get object from tombstone %#v", obj)
			return
		}
		d, ok = tombstone.Obj.(*v1beta1.Deployment)
		if !ok {
			glog.Errorf("Tombstone contained object that is not a Deployment %#v", obj)
			return
		}
	}

	// Verify that this is a deployment owned by the cluster.
	c, err := cc.getClusterForObject(d)
	if err != nil {
		if errors.IsNotFound(err) {
			// MemcachedCluster has been deleted.
			return
		}

		if glog.V(9) {
			glog.Errorf("Error getting cluster for deployment %#v: %#v", d, err)
		} else {
			glog.Errorf("Error getting cluster for deployment %q: %s", d.Name, err)
		}
		return
	}

	if c == nil {
		// Deployment not owned by a MemcachedCluster
		return
	}

	logObject("Deleting deployment: ", d)

	cc.enqueueCluster(c)
}

func (cc *Controller) syncTwemproxyDeployment(c *v1alpha1.MemcachedCluster) error {
	d, err := cc.getTwemproxyDeploymentForCluster(c)
	switch {
	case errors.IsNotFound(err):
		glog.V(2).Infof("Twemproxy Deployment for MemcachedCluster %q not found. Creating.", c.Name)
		d, _, err := cc.createTwemproxyDeployment(c)
		if err != nil {
			return fmt.Errorf("failed to create Deployment for MemcachedCluster %q: %v", c.Name, err)
		}
		logObject("Deployment created: ", d)

		return nil
	case err != nil:
		// Some other error occurred.
		return fmt.Errorf("failed to get Twemproxy Deployment for MemcachedCluster %q: %v", c.Name, err)
	}

	glog.V(9).Infof("Syncing Twemproxy Deployment %#v", d)

	_, _, err = cc.updateTwemproxyDeployment(c, d)
	if err != nil {
		return err
	}

	return nil
}

func (cc *Controller) calcTwemproxyDeploymentReplicas(c *v1alpha1.MemcachedCluster) (int32, error) {
	// Get the ConfigMap and check the number of PodIPs registered there. They should be updated
	// by syncTwemproxyConfigMap.

	cm, err := cc.getLatestTwemproxyConfigMapForCluster(c)
	if err != nil {
		return 0, fmt.Errorf("failed to get ConfigMap for Cluster %q: %v", c.Name, err)
	}

	// Scale TwemproxyDeployment to zero if there are Zero Pod IPs in the latest configmap.
	// TODO: Auto-Scale Twemproxy Deployment. Twemroxy is not likely CPU bound so how?
	replicas := 0
	if cm != nil {
		// Scale TwemproxyDeployment to 1 if there are Zero Pod IPs in the latest configmap.
		// Twemproxy crashes if no IPs are listed in "servers:" so simply scale it to zero.
		podIPs := podIPsForConfigMap(cm)
		if len(podIPs) > 0 {
			replicas = 1
		}
	}

	return int32(replicas), nil
}

func (cc *Controller) createTwemproxyDeployment(c *v1alpha1.MemcachedCluster) (*v1beta1.Deployment, bool, error) {
	name := twemproxyNameFromCluster(c)

	// TODO: Allow changing the number of replicas. Maybe autoscale by default?

	// Make initial replicas 0 as twemproxy does not handle config with 0 backend servers.
	// When the memcached Pod IPs are retrieved we will scale this up.
	replicas, err := cc.calcTwemproxyDeploymentReplicas(c)
	if err != nil {
		return nil, false, fmt.Errorf("failed to calc Twemproxy Deployment replicas for cluster %q: %v", c.Name, err)
	}

	cm, err := cc.getLatestTwemproxyConfigMapForCluster(c)
	if err != nil {
		return nil, false, err
	}

	if cm == nil {
		return nil, false, fmt.Errorf("No ConfigMap for MemcachedCluster %q", c.Name)
	}

	deployment := &v1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       c.Namespace,
			OwnerReferences: []metav1.OwnerReference{*newClusterOwnerRef(c)},
		},
		Spec: v1beta1.DeploymentSpec{
			// Use a single replica for now.
			Replicas: &replicas,
			// Set the DeploymentStrategy.
			// This is tricky since we want to update memcached server pod IP addresses promptly
			// but with large enough clusters there will constantly be churn and rolling updates may
			// not be able to keep up. Needs testing.
			Strategy: v1beta1.DeploymentStrategy{
				Type: v1beta1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &v1beta1.RollingUpdateDeployment{
					MaxSurge: &intstr.IntOrString{
						Type:   intstr.String,
						StrVal: "25%",
					},
					// Be fairly conservative with unavailable pods for the time being.
					MaxUnavailable: &intstr.IntOrString{
						Type:   intstr.String,
						StrVal: "0%",
					},
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						// TODO: Use a better selector.
						"twemproxy-name": name,
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						v1.Container{
							Name:  "twemproxy",
							Image: c.Spec.Proxy.TwemproxyProxySpec.Image,
							Args: []string{
								"--conf-file=/conf/config.yaml",
							},
							Ports: []v1.ContainerPort{
								v1.ContainerPort{
									Name:          "twemproxy",
									ContainerPort: memcachedPort,
								},
								// v1.ContainerPort{
								// 	Name:          "stats",
								// 	ContainerPort: 22222,
								// },
							},
							VolumeMounts: []v1.VolumeMount{
								v1.VolumeMount{
									Name:      "config-volume",
									MountPath: "/conf",
								},
							},
							ReadinessProbe: &v1.Probe{
								Handler: v1.Handler{
									TCPSocket: &v1.TCPSocketAction{
										Port: intstr.IntOrString{
											IntVal: memcachedPort,
										},
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       10,
							},
						},
					},
					Volumes: []v1.Volume{
						v1.Volume{
							Name: "config-volume",
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{
										Name: cm.Name,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	result, err := cc.client.Extensions().Deployments(c.Namespace).Create(deployment)
	if err != nil {
		// Return the deployment we attempted to created w/ the error
		return deployment, false, err
	}

	// Return the created deployment
	return result, true, nil
}

// updateTwemproxyDeployment updates the TwemproxyDeployment if necessary.
func (cc *Controller) updateTwemproxyDeployment(c *v1alpha1.MemcachedCluster, d *v1beta1.Deployment) (*v1beta1.Deployment, bool, error) {
	// Check if the Deployment needs updating.
	needsUpdate := false

	// This deployment is owned by the cluster.
	// Scale the Deployment if necessary.
	replicas, err := cc.calcTwemproxyDeploymentReplicas(c)
	if err != nil {
		return nil, false, fmt.Errorf("failed to calc Twemproxy Deployment replicas for cluster %q: %v", c.Name, err)
	}

	if *d.Spec.Replicas != replicas {
		needsUpdate = true
	}

	cm, err := cc.getLatestTwemproxyConfigMapForCluster(c)
	if err != nil {
		return nil, false, err
	}

	if cm == nil {
		return nil, false, fmt.Errorf("No ConfigMap for MemcachedCluster %q", c.Name)
	}

	// Check if we are using the latest ConfigMap
	found := false
	for _, v := range d.Spec.Template.Spec.Volumes {
		if v.Name == "config-volume" {
			found = true
			if v.VolumeSource.ConfigMap == nil || v.VolumeSource.ConfigMap.Name != cm.Name {
				needsUpdate = true
			}
			break
		}
	}
	if !found {
		// If no ConfigMap Volume found then we need to update.
		needsUpdate = true
	}

	if needsUpdate {
		glog.V(3).Infof("Updating Twemproxy Deployment %q.", d.Name)

		objCopy, err := scheme.Scheme.Copy(d)
		dCopy := objCopy.(*v1beta1.Deployment)

		dCopy.Spec.Replicas = &replicas

		dCopy.Spec.Template.Spec.Volumes = []v1.Volume{
			v1.Volume{
				Name: "config-volume",
				VolumeSource: v1.VolumeSource{
					ConfigMap: &v1.ConfigMapVolumeSource{
						LocalObjectReference: v1.LocalObjectReference{
							Name: cm.Name,
						},
					},
				},
			},
		}

		*dCopy.Spec.Replicas = replicas
		d, err = cc.client.Extensions().Deployments(dCopy.Namespace).Update(dCopy)
		if err != nil {
			return nil, false, fmt.Errorf("failed to update Twemproxy Deployment %q: %v", dCopy.Name, err)
		}

		return d, true, nil

	}

	return nil, false, nil
}

func (cc *Controller) deleteTwemproxyDeployment(key string) error {
	namespace, clusterName, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	name, err := twemproxyNameFromClusterKey(key)
	if err != nil {
		return err
	}

	d, err := cc.deploymentLister.Deployments(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			// If IsNotFound() then Deployment has already been deleted.
			return nil
		}

		// Another error occurred
		return err
	}

	// verify owner ref
	ref := getOwnerRefOfKind(d, v1alpha1.SchemeGroupVersion, "MemcachedCluster")
	if ref == nil || ref.Name != clusterName {
		// Log the error and return nil since this is not a temporary error.
		glog.Errorf("deployment %q ownerreference is %q; expected %q", name, ref.Name, clusterName)
		return nil
	}

	// Delete the Twemproxy Deployment. We will rely on Garbage collection here.
	deletePolicy := metav1.DeletePropagationForeground
	deleteOptions := &metav1.DeleteOptions{PropagationPolicy: &deletePolicy}
	if err := cc.client.ExtensionsV1beta1().Deployments(namespace).Delete(name, deleteOptions); err != nil {
		return fmt.Errorf("failed to delete Twemproxy Deployment %q: %v", name, err)
	}
	glog.V(2).Infof("Deployment %q deleted", name)

	return nil
}

func twemproxyNameFromCluster(c *v1alpha1.MemcachedCluster) string {
	// TODO: Use hash of cluster name and collision count
	return makeName(fmt.Sprintf("%s-twemproxy", c.Name), []string{c.Name})
}
func twemproxyNameFromClusterKey(key string) (string, error) {
	_, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return "", err
	}
	return makeName(fmt.Sprintf("%s-twemproxy", name), []string{name}), nil
}

func (cc *Controller) syncMemcachedDeployment(c *v1alpha1.MemcachedCluster) error {
	d, err := cc.getMemcachedDeploymentForCluster(c)
	switch {
	case errors.IsNotFound(err):
		glog.V(3).Infof("Memcached deployment for MemcachedCluster %q not found. Creating.", c.Name)
		d, err = cc.createMemcachedDeployment(c)
		if err != nil {
			return fmt.Errorf("failed to create Memcached Deployment for MemcachedCluster %q: %v", c.Name, err)
		}

		logObject("Deployment created: ", d)

		return nil
	case err != nil:
		// Some other error occurred.
		return fmt.Errorf("failed to get Memcached Deployment for MemcachedCluster %q: %v", c.Name, err)
	}

	glog.V(9).Infof("Syncing Memcached Deployment %#v", d)

	// This deployment is owned by the cluster.
	// Scale the Deployment if necessary.
	if *c.Spec.Replicas != *d.Spec.Replicas {
		_, err := cc.scaleDeployment(d, *c.Spec.Replicas)
		if err != nil {
			return err
		}
	}

	return nil
}

func (cc *Controller) createMemcachedDeployment(c *v1alpha1.MemcachedCluster) (*v1beta1.Deployment, error) {
	name := memcachedNameFromCluster(c)

	progressDeadlineSeconds := int32(10)
	deployment := &v1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       c.Namespace,
			OwnerReferences: []metav1.OwnerReference{*newClusterOwnerRef(c)},
		},
		Spec: v1beta1.DeploymentSpec{
			// Use a single replica for now.
			Replicas:                c.Spec.Replicas,
			ProgressDeadlineSeconds: &progressDeadlineSeconds,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						// TODO: Use a better selector.
						"memcached-name": name,
					},
				},
				Spec: v1.PodSpec{
					// TODO: Set the DeploymentStrategy.
					// DeploymentStrategy would be used when upgrading the memcached version, for instance.
					Containers: []v1.Container{
						v1.Container{
							Name:  "memcached",
							Image: c.Spec.MemcachedImage,
							Ports: []v1.ContainerPort{
								v1.ContainerPort{
									Name:          "memcached",
									ContainerPort: memcachedPort,
								},
							},
							ReadinessProbe: &v1.Probe{
								Handler: v1.Handler{
									TCPSocket: &v1.TCPSocketAction{
										Port: intstr.IntOrString{
											IntVal: memcachedPort,
										},
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       10,
							},
						},
					},
				},
			},
		},
	}

	result, err := cc.client.Extensions().Deployments(c.Namespace).Create(deployment)
	if err != nil {
		// Return the deployment we attempted to created w/ the error
		return deployment, err
	}

	// Return the created deployment
	return result, nil
}

func (cc *Controller) deleteMemcachedDeployment(key string) error {
	namespace, clusterName, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	// Delete the Memcached Deployment. We will rely on Garbage collection here.
	name, err := memcachedNameFromClusterKey(key)
	if err != nil {
		return err
	}

	d, err := cc.deploymentLister.Deployments(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			// If IsNotFound() then Deployment has already been deleted.
			return nil
		}

		// Another error occurred
		return err
	}

	// verify owner ref
	ref := getOwnerRefOfKind(d, v1alpha1.SchemeGroupVersion, "MemcachedCluster")
	if ref == nil || ref.Name != clusterName {
		// Log the error and return nil since this is not a temporary error.
		glog.Errorf("deployment %q ownerreference is %q; expected %q", name, ref.Name, clusterName)
		return nil
	}

	// Delete the Twemproxy Deployment. We will rely on Garbage collection here.
	deletePolicy := metav1.DeletePropagationForeground
	deleteOptions := &metav1.DeleteOptions{PropagationPolicy: &deletePolicy}
	if err := cc.client.ExtensionsV1beta1().Deployments(namespace).Delete(name, deleteOptions); err != nil {
		return fmt.Errorf("failed to delete Memcached Deployment %q: %v", name, err)
	}
	glog.V(2).Infof("Deployment %q deleted", name)

	return nil
}

func memcachedNameFromCluster(c *v1alpha1.MemcachedCluster) string {
	// TODO: Use hash of cluster name and collision count
	return makeName(fmt.Sprintf("%s-memcached", c.Name), []string{c.Name})
}
func memcachedNameFromClusterKey(key string) (string, error) {
	_, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return "", err
	}
	return makeName(fmt.Sprintf("%s-memcached", name), []string{name}), nil
}

func (cc *Controller) scaleDeployment(d *v1beta1.Deployment, replicas int32) (*v1beta1.Deployment, error) {
	objCopy, err := scheme.Scheme.Copy(d)
	if err != nil {
		return nil, fmt.Errorf("failed to copy Deployment %q: %v", d.Name, err)
	}
	dCopy := objCopy.(*v1beta1.Deployment)

	*dCopy.Spec.Replicas = replicas
	d, err = cc.client.Extensions().Deployments(dCopy.Namespace).Update(dCopy)
	if err != nil {
		return nil, fmt.Errorf("failed to scale Twemproxy Deployment %q: %v", dCopy.Name, err)
	}

	return d, nil
}
