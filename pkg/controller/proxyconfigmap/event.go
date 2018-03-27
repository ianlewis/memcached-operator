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

package proxyconfigmap

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ianlewis/memcached-operator/pkg/apis/ianlewis.org/v1alpha1"
)

func (c *Controller) recordConfigMapEvent(verb string, p *v1alpha1.MemcachedProxy, cm *corev1.ConfigMap, err error) {
	c.recordEvent(verb, p, "ConfigMap", cm, err)
}

func (c *Controller) recordEvent(verb string, p *v1alpha1.MemcachedProxy, kind string, obj metav1.ObjectMetaAccessor, err error) {
	if err == nil {
		var msg string
		reason := fmt.Sprintf("Successful%s", strings.Title(verb))
		if obj == nil {
			msg = fmt.Sprintf("%s %s for MemcachedProxy %q successful",
				strings.ToLower(verb), kind, p.Name)
		} else {
			objName := obj.GetObjectMeta().GetName()
			msg = fmt.Sprintf("%s %s %q for MemcachedProxy %q successful",
				strings.ToLower(verb), kind, objName, p.Name)
		}
		c.recorder.Event(p, corev1.EventTypeNormal, reason, msg)
	} else {
		var msg string
		reason := fmt.Sprintf("Failed%s", strings.Title(verb))
		if obj == nil {
			msg = fmt.Sprintf("%s %s for MemcachedProxy %q failed error: %s",
				strings.ToLower(verb), kind, p.Name, err)
		} else {
			objName := obj.GetObjectMeta().GetName()
			msg = fmt.Sprintf("%s %s %q for MemcachedProxy %q failed error: %s",
				strings.ToLower(verb), kind, objName, p.Name, err)
		}
		c.recorder.Event(p, corev1.EventTypeWarning, reason, msg)
	}
}
