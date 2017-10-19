package controller

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/golang/glog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// MakeName creates an object name (usually Pod) by appending the given prefix to a hash of the given values (such as another object's UID).
func MakeName(prefix string, hashValues []string) string {
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

// GetOwnerRefOfKind returns the OwnerReference of the given kind for the given object
func GetOwnerRefOfKind(obj metav1.Object, version schema.GroupVersion, kind string) *metav1.OwnerReference {
	ownerRefs := obj.GetOwnerReferences()
	apiVersion := version.Group + "/" + version.Version
	for _, ref := range ownerRefs {
		if ref.APIVersion == apiVersion && ref.Kind == kind {
			return &ref
		}
	}

	return nil
}

// LogObject logs given objects with all fields if log level >=9 otherwise it logs by name
func LogObject(msg string, o ...metav1.ObjectMetaAccessor) {
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
