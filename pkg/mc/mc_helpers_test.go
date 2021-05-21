package mc

const (
	context   = "kind-kind"
	namespace = "default"
)

var (
	kubectlReturn = []byte(`NAME                                         READY   STATUS    RESTARTS   AGE
coredns-66bff467f8-4lnsg                     1/1     Running   0          14h
coredns-66bff467f8-czsf6                     1/1     Running   0          14h
etcd-kind-control-plane                      1/1     Running   0          14h
kindnet-j682f                                1/1     Running   0          14h
kube-apiserver-kind-control-plane            1/1     Running   0          14h
kube-controller-manager-kind-control-plane   1/1     Running   0          14h
kube-proxy-trbmh                             1/1     Running   0          14h
kube-scheduler-kind-control-plane            1/1     Running   0          14h
`)

	directories       = "bin\nlib\nlocal\nsbin\nshare\n"
	directoriesReturn = `
kind-kind1
----------
bin
lib
local
sbin
share
`
	directoriesReturn1 = `
kind-kind
---------
bin
lib
local
sbin
share
`

	kubectlReturnSA = []byte(`
{
    "apiVersion": "v1",
    "items": [
        {
            "apiVersion": "v1",
            "kind": "ServiceAccount",
            "metadata": {
                "creationTimestamp": "2021-03-21T03:59:54Z",
                "name": "default",
                "namespace": "default",
                "resourceVersion": "372",
                "selfLink": "/api/v1/namespaces/default/serviceaccounts/default",
                "uid": "2600c99a-e702-461d-89f6-c5e020e91d30"
            },
            "secrets": [
                {
                    "name": "default-token-6x8kn"
                }
            ]
        }
    ],
    "kind": "List",
    "metadata": {
        "resourceVersion": "",
        "selfLink": ""
    }
}
`)

	jsonReturn = `{
  "kind-kind": {
    "apiVersion": "v1",
    "items": [
      {
        "apiVersion": "v1",
        "kind": "ServiceAccount",
        "metadata": {
          "creationTimestamp": "2021-03-21T03:59:54Z",
          "name": "default",
          "namespace": "default",
          "resourceVersion": "372",
          "selfLink": "/api/v1/namespaces/default/serviceaccounts/default",
          "uid": "2600c99a-e702-461d-89f6-c5e020e91d30"
        },
        "secrets": [
          {
            "name": "default-token-6x8kn"
          }
        ]
      }
    ],
    "kind": "List",
    "metadata": {
      "resourceVersion": "",
      "selfLink": ""
    }
  },
  "kind-kind1": {
    "apiVersion": "v1",
    "items": [
      {
        "apiVersion": "v1",
        "kind": "ServiceAccount",
        "metadata": {
          "creationTimestamp": "2021-03-21T03:59:54Z",
          "name": "default",
          "namespace": "default",
          "resourceVersion": "372",
          "selfLink": "/api/v1/namespaces/default/serviceaccounts/default",
          "uid": "2600c99a-e702-461d-89f6-c5e020e91d30"
        },
        "secrets": [
          {
            "name": "default-token-6x8kn"
          }
        ]
      }
    ],
    "kind": "List",
    "metadata": {
      "resourceVersion": "",
      "selfLink": ""
    }
  }
}`

	yamlReturn = `kind-kind:
  apiVersion: v1
  items:
  - apiVersion: v1
    kind: ServiceAccount
    metadata:
      creationTimestamp: "2021-03-21T03:59:54Z"
      name: default
      namespace: default
      resourceVersion: "372"
      selfLink: /api/v1/namespaces/default/serviceaccounts/default
      uid: 2600c99a-e702-461d-89f6-c5e020e91d30
    secrets:
    - name: default-token-6x8kn
  kind: List
  metadata:
    resourceVersion: ""
    selfLink: ""
kind-kind1:
  apiVersion: v1
  items:
  - apiVersion: v1
    kind: ServiceAccount
    metadata:
      creationTimestamp: "2021-03-21T03:59:54Z"
      name: default
      namespace: default
      resourceVersion: "372"
      selfLink: /api/v1/namespaces/default/serviceaccounts/default
      uid: 2600c99a-e702-461d-89f6-c5e020e91d30
    secrets:
    - name: default-token-6x8kn
  kind: List
  metadata:
    resourceVersion: ""
    selfLink: ""
`
)
