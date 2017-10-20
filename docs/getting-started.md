# Getting Started

## Deploy memcached-operator

TODO: deploy with pre-built Docker images.

## Running Locally

Start the memcached-operator. The application will create the required `CustomResourceDefinition` if necessary.

    $ memcached-operator -kubeconfig ~/.kube/config -logtostderr -v 3

Create a memcached cluster using helm. Make sure you have initialized helm with `helm init`.

    $ helm install --name example stable/memcached

Create a `MemcachedProxy` CRD.

```
$ cat <<EOF > example-proxy.yaml
> apiVersion: ianlewis.org/v1alpha1
> kind: MemcachedProxy
>   name: example
> spec:
>   serviceName: example-memcached
>   servicePort: 11211
> EOF
$ kubectl apply -f example-proxy.yaml
```

The Memcached Operator will then create a new proxy deployment and service for you. Your application can then access the memcached cluster at `example-memcachedproxy` on the standard memcached port 11211.

For instance here is a sample app that could connect to the memcached cluster if run from within the Kubernetes cluster. Cache data will be sharded across all instances in the memcached cluster.

```
package main
import (
  "fmt" 
	"github.com/bradfitz/gomemcache/memcache"
)

func main() {
  mc := memcache.New("example-memcachedproxy:11211")
	obj, err := mc.Get("some-key")
  if err != nil {
		fmt.Printf("Error reading from memcached: %v", err)
    return
  }
  fmt.Printf("value: %s", obj.Value)
}
```
