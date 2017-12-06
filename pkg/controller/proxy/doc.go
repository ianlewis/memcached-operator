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
