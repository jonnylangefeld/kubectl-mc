package mc

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"sync"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/jonnylangefeld/kubectl-mc/pkg/mc/mocks"

	"github.com/stretchr/testify/assert"
)

func TestMC_ExecuteCommand(t *testing.T) {
	tests := map[string]struct {
		args               []string
		listContextsReturn []byte
		kubectlReturns     [][]byte
		wantContains       []string
		wantErr            error
	}{
		"get pods from one cluster": {
			args:               []string{"-r", "kind", "--", "get", "pods", "-n", "kube-system,"},
			listContextsReturn: []byte("kind-kind\nkind-kind1\n"),
			kubectlReturns: [][]byte{
				[]byte("NAME                                         READY   STATUS    RESTARTS   AGE\ncoredns-66bff467f8-4lnsg                     1/1     Running   1          22h\n"),
				[]byte("NAME                                         READY   STATUS    RESTARTS   AGE\ncoredns-66bff467f8-4lnsg                     1/1     Running   1          22h\n"),
			},
			wantContains: []string{`
kind-kind1
----------
NAME                                         READY   STATUS    RESTARTS   AGE
coredns-66bff467f8-4lnsg                     1/1     Running   1          22h
`,
				`
kind-kind
---------
NAME                                         READY   STATUS    RESTARTS   AGE
coredns-66bff467f8-4lnsg                     1/1     Running   1          22h
`,
			},
		},
		"exec": {
			args:               []string{"-r", "kind", "--", "exec", "deployment/local-path-provisioner", "-n", "local-path-storage", "-it", "--", "ls", "/usr"},
			listContextsReturn: []byte("kind-kind\nkind-kind1\n"),
			kubectlReturns: [][]byte{
				[]byte(directories),
				[]byte(directories),
			},
			wantContains: []string{directoriesReturn, directoriesReturn1},
		},
		"list contexts": {
			args:               []string{"-r", "kind", "-l"},
			listContextsReturn: []byte("kind-kind\nkind-kind1\nfoo\nbar\n"),
			wantContains:       []string{"kind-kind\n", "kind-kind1\n"},
		},
		"json": {
			args:               []string{"-r", "kind", "-o", "json", "--", "get", "pods", "-n", "kube-system,"},
			listContextsReturn: []byte("kind-kind\nkind-kind1\n"),
			kubectlReturns: [][]byte{
				kubectlReturnSA,
				kubectlReturnSA,
			},
			wantContains: []string{jsonReturn},
		},
		"yaml": {
			args:               []string{"-r", "kind", "-o", "yaml", "--", "get", "pods", "-n", "kube-system,"},
			listContextsReturn: []byte("kind-kind\nkind-kind1\n"),
			kubectlReturns: [][]byte{
				kubectlReturnSA,
				kubectlReturnSA,
			},
			wantContains: []string{yamlReturn},
		},
		"unknown output": {
			args:    []string{"-r", "kind", "-o", "foo", "--", "get", "pods", "-n", "kube-system,"},
			wantErr: errUnknownOutput,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := mocks.NewMockCmd(ctrl)
			m.EXPECT().CombinedOutput().Return(test.listContextsReturn, nil)
			for _, r := range test.kubectlReturns {
				m.EXPECT().CombinedOutput().Return(r, nil)
			}
			mc := New("")
			mc.getListContextsCmd = func() Cmd {
				return m
			}
			mc.getKubectlCmd = func(args []string, context string) Cmd {
				return m
			}
			b := bytes.NewBuffer([]byte(``))
			errB := bytes.NewBuffer([]byte(``))
			mc.Cmd.SetOut(b)
			mc.Cmd.SetErr(errB)
			mc.Cmd.SetArgs(test.args)
			err := mc.Cmd.Execute()
			if test.wantErr != nil {
				assert.Equal(t, test.wantErr, err)
			} else {
				assert.NoError(t, err)
			}
			got, err := ioutil.ReadAll(b)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range test.wantContains {
				assert.Contains(t, string(got), want)
			}
		})
	}
}

func TestMC_ListContexts(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := mocks.NewMockCmd(ctrl)

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
			mc := MC{
				Regex:    test.regex,
				NegRegex: test.negRegex,
			}
			got, err := mc.listContexts(m)
			assert.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestDo(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := mocks.NewMockCmd(ctrl)

	m.EXPECT().CombinedOutput().Return(kubectlReturn, nil)

	done := make(chan bool, 1)
	var mutex = &sync.Mutex{}
	output := map[string]json.RawMessage{}
	do(done, context, output, false, nil, m, mutex)
	assert.True(t, <-done)
	assert.Equal(t, map[string]json.RawMessage{context: kubectlReturn}, output)
}

func TestKubectl(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := mocks.NewMockCmd(ctrl)

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
			args: []string{"exec", "deployment/local-path-provisioner", "-n", "local-path-storage", "-it", "--", "ls", "/usr"},
			want: []string{"exec", "deployment/local-path-provisioner", "-n", "local-path-storage", "-it", "--context", context, "--", "ls", "/usr"},
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
