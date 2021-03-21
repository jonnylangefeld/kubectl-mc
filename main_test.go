package main

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/jonnylangefeld/kubectl-mc/mocks"

	"github.com/stretchr/testify/assert"
)

const (
	context = "kind-kind"
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
)

func TestMC_ListContexts(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := mocks.NewMockCMD(ctrl)

	tests := map[string]struct {
		kubectlReturn []byte
		regex         string
		negRegex      string
		want          []string
	}{
		"only dev clusters": {
			kubectlReturn: []byte(`kind-kind
gke_project-dev_cluster-dev
gke_project-dev-test_cluster-test
gke_project-prod_cluster-prod
`),
			regex: "dev",
			want:  []string{"gke_project-dev_cluster-dev", "gke_project-dev-test_cluster-test"},
		},
		"gke clusters but no dev clusters": {
			kubectlReturn: []byte(`kind-kind
gke_project-dev_cluster-dev
gke_project-dev-test_cluster-test
gke_project-stg_cluster-stg
gke_project-prod_cluster-prod
`),
			regex:    "gke",
			negRegex: "dev",
			want:     []string{"gke_project-stg_cluster-stg", "gke_project-prod_cluster-prod"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			m.EXPECT().CombinedOutput().Return(test.kubectlReturn, nil)
		})
		mc := MC{
			Regex:    test.regex,
			NegRegex: test.negRegex,
		}
		got, err := mc.listContexts(m)
		assert.NoError(t, err)
		assert.Equal(t, test.want, got)
	}
}

func TestDo(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := mocks.NewMockCMD(ctrl)

	m.EXPECT().CombinedOutput().Return(kubectlReturn, nil)

	done := make(chan bool, 1)
	output := map[string]json.RawMessage{}
	do(done, context, output, false, m)
	assert.True(t, <-done)
	assert.Equal(t, map[string]json.RawMessage{context: kubectlReturn}, output)
}

func TestKubectl(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := mocks.NewMockCMD(ctrl)

	m.EXPECT().CombinedOutput().Return(kubectlReturn, nil)

	got, err := kubectl(m)
	assert.NoError(t, err)
	assert.Equal(t, kubectlReturn, got)

	m.EXPECT().CombinedOutput().Return([]byte(`Error: unknown shorthand flag: 'a' in -abc
See 'kubectl get --help' for usage.`), errors.New(""))

	got, err = kubectl(m)
	assert.Nil(t, got)
	assert.Error(t, err)
	assert.Equal(t, "unknown shorthand flag: 'a' in -abc\nSee 'kubectl get --help' for usage.", err.Error())
}

func TestGetLocalArgs(t *testing.T) {
	tests := map[string]struct {
		args []string
		want []string
	}{
		"default": {
			args: []string{"get", "pods", "-n", "kube-system"},
			want: []string{"get", "pods", "-n", "kube-system", "--context", context},
		},
		"exec": {
			args: []string{"exec", "debug", "-it", "--", "bash"},
			want: []string{"exec", "debug", "-it", "--context", context, "--", "bash"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := getLocalArgs(test.args, context)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestOutputsString(t *testing.T) {
	got := outputsString()
	assert.Contains(t, got, "|")
}

func TestFormatContext(t *testing.T) {
	got := formatContext(context, kubectlReturn)
	assert.Equal(t, "\nkind-kind\n---------\n"+string(kubectlReturn), got)
}
