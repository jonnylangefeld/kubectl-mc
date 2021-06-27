[![Go Report Card](https://goreportcard.com/badge/github.com/jonnylangefeld/kubectl-mc)](https://goreportcard.com/report/github.com/jonnylangefeld/kubectl-mc)
[![codecov](https://codecov.io/gh/jonnylangefeld/kubectl-mc/branch/master/graph/badge.svg?token=Q38GREUVRF)](https://codecov.io/gh/jonnylangefeld/kubectl-mc)
[![Build](https://img.shields.io/github/workflow/status/jonnylangefeld/kubectl-mc/Main)](https://github.com/jonnylangefeld/kubectl-mc/actions/workflows/main.yml)
[![Twitter](https://img.shields.io/badge/twitter-@jonnylangefeld-blue.svg)](http://twitter.com/jonnylangefeld)
[![GitHub release](https://img.shields.io/github/release/jonnylangefeld/kubectl-mc.svg)](https://github.com/jonnylangefeld/kubectl-mc/releases)
![GitHub](https://img.shields.io/github/license/jonnylangefeld/kubectl-mc)

# A Multi cluster kubectl Client

Run kubectl commands against multiple clusters at once.

If you work at a company or organization that maintains multiple Kubernetes clusters it is fairly common to connect to multiple different kubernetes clusters throughout your day. And sometimes you want to execute a command against multiple clusters at once. For instance to get the status of a deployment across all `staging` clusters. You could run your `kubectl` command in a bash loop. That does not only require some bash logic, but also it'll take a while to get your results because every loop iteration is an individual API round trip executed successively.

`kubectl-mc` (short for multi cluster) supports this workflow and significantly reduces the return time by executing the necessary API requests in parallel go routines.

# Installation

Run `go get github.com/jonnylangefeld/kubectl-mc`.  
This will make the binary available on your path and allows you to run it as `kubectl` addon via `kubectl mc`. More [here](https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins).

### Via [`krew`](https://krew.sigs.k8s.io/)

Krew is the plugin manager for the `kubectl` command-line tool.  
Run `krew install mc` to install mc. You can then use it via `kubectl mc`.

# Usage

Run `kubectl mc` for help.

## Using `kubectl mc` in automation with jq and yq

The `kubectl mc` command supports native json and yaml output. This allows for effective usage for automations and inventory run scenarios in multicluster setups using `jq` and `yq`. 

Note: the `-o output` must be specified for the `kubectl mc` command and not the kubectl argument, as it produces different output. When specified as a kubectl argument, the output is not machine parsable as it does not present a valid json / yaml object.

Wrong usage for automation:

```
$ kubectl mc -r testcluster -- get node --sort-by="{.metadata.creationTimestamp}" -o yaml

```

Example yaml output:

```
$ kubectl mc -r testcluster -o yaml -- get node --sort-by="{.metadata.creationTimestamp}"
testcluster1:
  apiVersion: v1
  items:
  - apiVersion: v1
    kind: Node
	...
  - apiVersion: v1
    kind: Node
testcluster2:
  apiVersion: v1
  items:
  - apiVersion: v1
    kind: Node
	...
  - apiVersion: v1
    kind: Node
```

Example json output:

```
$ kubectl mc -r testcluster -o json -- get node --sort-by="{.metadata.creationTimestamp}"
{
  "testcluster1": {
    "apiVersion": "v1",
    "items": [
      {
        "apiVersion": "v1",
        "kind": "Node",
		...
      {
        "apiVersion": "v1",
        "kind": "Node",
		...
	]
  },
  "testcluster2": {
    "apiVersion": "v1",
    "items": [
      {
        "apiVersion": "v1",
        "kind": "Node",
		...
      {
        "apiVersion": "v1",
        "kind": "Node",
		...
	]
  }
}
```

In this example the key `testcluster1` is the context name in your kubectl context file. The value of the hash is the results from the kubectl call as you are used to. This can easily be used in automations.

* Access a single cluster


```
# Get all nodes with testclusters matching "lab"
kubectl mc -r testcluster -o json -- get node --sort-by="{.metadata.creationTimestamp}" | jq -r '.testcluster1.items[] | .metadata.name'
# Example when context names contain special characters
kubectl mc -r testcluster -o json -- get node --sort-by="{.metadata.creationTimestamp}" | jq -r '."aws:region1:testcluster".items[] | .metadata.name'
```

* Access a few clusters following a pattern

```
# Get all nodes with testclusters matching "lab"
kubectl mc -r testcluster -o json -- get node --sort-by="{.metadata.creationTimestamp}" | jq -r 'to_entries[] | select(.key | test("lab")) | .value.items[] | .metadata.name'
```


# UX

```bash
$ kubectl mc -r kind -- get pods -n kube-system

kind-kind
---------
NAME                                         READY   STATUS    RESTARTS   AGE
coredns-f9fd979d6-q7gnm                      1/1     Running   0          99m
coredns-f9fd979d6-zd4jn                      1/1     Running   0          99m
etcd-kind-control-plane                      1/1     Running   0          99m
kindnet-8qd8p                                1/1     Running   0          99m
kube-apiserver-kind-control-plane            1/1     Running   0          99m
kube-controller-manager-kind-control-plane   1/1     Running   0          99m
kube-proxy-nb55k                             1/1     Running   0          99m
kube-scheduler-kind-control-plane            1/1     Running   0          99m

kind-another-kind-cluster
-------------------------
NAME                                                         READY   STATUS    RESTARTS   AGE
coredns-f9fd979d6-l2xdb                                      1/1     Running   0          91s
coredns-f9fd979d6-m99fx                                      1/1     Running   0          91s
etcd-another-kind-cluster-control-plane                      1/1     Running   0          92s
kindnet-jlrqg                                                1/1     Running   0          91s
kube-apiserver-another-kind-cluster-control-plane            1/1     Running   0          92s
kube-controller-manager-another-kind-cluster-control-plane   1/1     Running   0          92s
kube-proxy-kq2tr                                             1/1     Running   0          91s
kube-scheduler-another-kind-cluster-control-plane            1/1     Running   0          92s
```

## Speed Comparison

To demonstrate the advantages in speed, here is the same task once executed as bash script and once using `kubectl-mc` for a list of  cluster:

```bash
$ kubectl mc -r 'prod' -l | wc -l
      13

$ time kubectl mc -r 'prod' -- get pods -n gatekeeper-system -l gatekeeper.sh/operation=audit > /dev/null
kubectl mc -r 'prod' -- get pods -n gatekeeper-system -l  > /dev/null  1.68s user 1.03s system 123% cpu 2.191 total

$ print "for c in \$(kubectx | grep -E 'prod'); do echo $c && kubectl get pods -n gatekeeper-system -l gatekeeper.sh/operation=audit --context $c ; done" > /tmp/loop.sh && \
chmod +x /tmp/loop.sh

$ time /tmp/loop.sh > /dev/null
/tmp/loop.sh > /dev/null  1.41s user 1.17s system 33% cpu 7.801 total
```

While the execution via bash loop over 13 clusters took 7.9 seconds, `kubectl-mc` only took 2.2 seconds in this non-empirical test.
