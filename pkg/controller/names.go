package controller

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ianlewis/memcached-operator/pkg/apis/ianlewis.org/v1alpha1"
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
	return prefix + strings.ToLower(hash[2:7])
}

// GetProxyServiceName returns the name of the proxy service for the given proxy
func GetProxyServiceName(p *v1alpha1.MemcachedProxy) string {
	return MakeName(fmt.Sprintf("%s-memcached-", p.Name), []string{p.Name})
}

// GetProxyServiceSelector returns labels used by the proxy service's selector
func GetProxyServiceSelector(p *v1alpha1.MemcachedProxy) map[string]string {
	return map[string]string{
		"memcached-operator": "true",
		"mcrouter":           MakeName(p.Name+"-", []string{p.Namespace, p.Name}),
	}
}
