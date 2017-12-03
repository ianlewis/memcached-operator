package proxyconfigmap

import (
	"fmt"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/cache"

	"github.com/ianlewis/memcached-operator/pkg/apis/ianlewis.org/v1alpha1"
)

// getAffectedProxies returns a set of string keys for each proxy that is afffected
// by a change in the given endpoints for a referenced service.
func (c *Controller) getAffectedProxies(ep *corev1.Endpoints) (sets.String, error) {
	set := sets.String{}

	// Get the services for each memcached proxy
	proxyMap, err := c.getAllProxyServiceSpecs()
	if err != nil {
		return set, fmt.Errorf("failed to get services for memcached proxies: %v", err)
	}

	// Find proxies that reference services that have the same name as the endpoints object.
	// We don't query the API here because the services may have been deleted.
	for key, serviceSpecs := range proxyMap {
		for _, serviceSpec := range serviceSpecs {
			if serviceSpec.Namespace == ep.Namespace && serviceSpec.Name == ep.Name {
				set.Insert(key)
			}
		}
	}

	return set, nil
}

// getAllProxyServices returns a map of memcached proxy to affected services. The
// key is a string key as used in the workqueue.
func (c *Controller) getAllProxyServiceSpecs() (map[string][]*v1alpha1.ServiceSpec, error) {
	// Only watch the namespace given to the controller constructor
	proxies, err := c.pLister.MemcachedProxies(c.namespace).List(labels.Everything())
	if err != nil {
		return nil, fmt.Errorf("failed to get memcached proxies for namespace %q", c.namespace)
	}

	result := make(map[string][]*v1alpha1.ServiceSpec)
	for _, p := range proxies {
		key, err := KeyFunc(p)
		if err != nil {
			return nil, fmt.Errorf("failed to get key for proxy: %q: %v", p.Namespace+"/"+p.Name, err)
		}

		result[key] = c.getProxyServiceSpecs(p)
	}

	return result, nil
}

// getProxyServiceSpecs returns a list of unique service specs referenced by the given proxy
func (c *Controller) getProxyServiceSpecs(p *v1alpha1.MemcachedProxy) []*v1alpha1.ServiceSpec {
	serviceMap := c.serviceSpecsForRule(p.Spec.Rules)

	services := make([]*v1alpha1.ServiceSpec, len(serviceMap))
	index := 0
	for _, s := range serviceMap {
		services[index] = s
		index++
	}

	return services
}

// serviceSpecsForRule returns the service names referenced by the proxy rule spec.
// A map is returned to maitain a unique set across during recursion
func (c *Controller) serviceSpecsForRule(r v1alpha1.RuleSpec) map[string]*v1alpha1.ServiceSpec {
	result := make(map[string]*v1alpha1.ServiceSpec)
	// If the service spec is set then simply return that service
	if r.Service != nil {
		result[r.Service.Namespace+"/"+r.Service.Name] = r.Service
		return result
	}

	// ... otherwise return the set of all services returned by child rules
	for _, childRule := range r.Children {
		services := c.serviceSpecsForRule(childRule)
		for k, spec := range services {
			result[k] = spec
		}
	}

	return result
}

// addEndpoints enqueues MemcachedProxy objects that reference services
// that are associated with the endpoints object
func (c *Controller) addEndpoints(obj interface{}) {
	// verify that a service exists for the endpoint
	endpoints := obj.(*corev1.Endpoints)

	keys, err := c.getAffectedProxies(endpoints)
	if err != nil {
		c.l.Error.Printf("failed to get affected proxies for endpoints %q: %v", endpoints.Namespace+"/"+endpoints.Name, err)
		return
	}

	for key := range keys {
		c.queue.Add(key)
	}
}

// Update endpoint checks if there was a relavent change and then
// enqueues MemcachedProxy objects that reference services
// that are associated with the endpoints object
func (c *Controller) updateEndpoints(old, new interface{}) {
	oldEndpoints := old.(*corev1.Endpoints)
	newEndpoints := new.(*corev1.Endpoints)
	if oldEndpoints.ResourceVersion == newEndpoints.ResourceVersion {
		// Periodic resync will send update events for all known endpoints.
		// Two different versions of the same endpoints will always have different RVs.
		return
	}
	if !reflect.DeepEqual(oldEndpoints.Subsets, newEndpoints.Subsets) {
		// We need to enqueue proxies that reference services for the endpoints.
		// This just happens to be exactly what we do in addEndpoints
		c.addEndpoints(newEndpoints)
	}
}

// deleteEndpoints enqueues MemcachedProxy objects that reference services
// that are associated with the endpoints object
func (c *Controller) deleteEndpoints(obj interface{}) {
	if _, ok := obj.(*corev1.Endpoints); ok {
		// We need to enqueue proxies that reference services for the endpoints.
		// This just happens to be exactly what we do in addEndpoints
		c.addEndpoints(obj)
		return
	}
	// If we reached here it means the endpoints was deleted but its final state is unrecorded.
	tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
	if !ok {
		c.l.Error.Printf("failed to get object from tombstone %#v", obj)
		return
	}
	endpoints, ok := tombstone.Obj.(*corev1.Endpoints)
	if !ok {
		c.l.Error.Printf("tombstone contained object that is not a endpoints: %#v", obj)
		return
	}
	c.addEndpoints(endpoints)

}
