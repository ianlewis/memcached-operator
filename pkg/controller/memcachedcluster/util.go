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
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/golang/glog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// makeName() creates an object name (usually Pod) by appending the given prefix to a hash of the given values (such as another object's UID).
func makeName(prefix string, hashValues []string) string {
	hasher := sha1.New()
	for _, val := range hashValues {
		fmt.Fprintf(hasher, "%s", val)
	}
	// Hashes are shortened to 5 characters to keep them reasonably manageable to
	// work with. Names of objects in Kubernetes must be lowercase alphanumeric so
	// the hash value is encoded to hex and changed to lowercase.
	hash := hex.EncodeToString(hasher.Sum(nil))
	return prefix + "-" + strings.ToLower(hash[2:7])
}

func logObject(msg string, o ...metav1.ObjectMetaAccessor) {
	if glog.V(9) {
		obj := make([]interface{}, len(o))
		extra := make([]string, len(o))
		for i, a := range o {
			obj[i] = a
			extra[i] = "%#v"
		}
		glog.Infof(msg+strings.Join(extra, ", "), obj...)
	} else if glog.V(2) {
		names := make([]interface{}, len(o))
		extra := make([]string, len(o))
		for i, a := range o {
			names[i] = a.GetObjectMeta().GetName()
			extra[i] = "%q"
		}
		glog.Infof(msg+strings.Join(extra, ", "), names...)
	}
}

func getOwnerRefOfKind(obj metav1.Object, version schema.GroupVersion, kind string) *metav1.OwnerReference {
	ownerRefs := obj.GetOwnerReferences()
	apiVersion := version.Group + "/" + version.Version
	for _, ref := range ownerRefs {
		if ref.APIVersion == apiVersion && ref.Kind == kind {
			return &ref
		}
	}

	return nil
}
