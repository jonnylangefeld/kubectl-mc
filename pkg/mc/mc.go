package mc

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"sigs.k8s.io/yaml"
)

const (
	// YAML represents the string for yaml
	YAML = "yaml"
	// JSON represents the string for json
	JSON = "json"
)

var (
	logger  *zap.Logger
	outputs = map[string]bool{
		YAML: true,
		JSON: true,
	}

	errUnknownOutput      = fmt.Errorf("this output format is unknown. Choose one of %s", outputsString())
	errCouldntParseOutput = fmt.Errorf("couldn't parse this output. Are you sure your kubectl command allows for json output? Run command with -d to see debug output")
)

// MC contains the options of the command
type MC struct {
	Cmd        *cobra.Command
	Regex      string
	NegRegex   string
	Namespaces string
	ListOnly   bool
	MaxProc    int
	Debug      bool
	Output     string

	// to allow dependency injection
	getListContextsCmd func() Cmd
	getKubectlCmd      func(args []string, context string, namespace string) Cmd
}

// Cmd is an interface for exec.Cmd to allow for dependency injection
//go:generate go run -mod=mod github.com/golang/mock/mockgen --build_flags=-mod=mod -destination=./mocks/cmd.go -package=mocks -source=./mc.go
type Cmd interface {
	CombinedOutput() ([]byte, error)
}

// New registers the default mc command
func New(version string) *MC {
	mc := &MC{}

	// to allow dependency injection
	mc.getListContextsCmd = func() Cmd {
		return exec.Command("kubectl", []string{"config", "get-contexts", "-o", "name"}...)
	}
	mc.getKubectlCmd = func(args []string, context string, namespace string) Cmd {
		return exec.Command("kubectl", getLocalArgs(args, context, namespace)...)
	}

	cmd := &cobra.Command{
		Use:   "mc [flags] -- [kubectl command]",
		Short: "Run kubectl commands against multiple clusters at once",
		Example: `
# list all kind contexts
mc -r kind -l

# list the pods in the kube-system namespace of all dev clusters
mc -r dev -- get pods -n kube-system

# run a debug container on every kind cluster in the context
mc --regex kind -- run debug --image=markeijsermans/debug --command -- sleep infinity

# list all contexts with 'dev' in the name, but not '-test-' in the name
mc -r dev -l -n '-test-'

# list all pods with label 'app.kubernetes.io/name=audit' in the 'default' namespace from all clusters with 'gke' in the name, but not 'dev'
# run max 5 processes in parallel and enable debug output
mc -r gke -n 'dev' -p 5 -d -- get pods -n gatekeeper-system -l app.kubernetes.io/name=audit

# print the context and the pod names in kube-system using jq
mc -r kind -o json -- get pods -n kube-system | jq 'keys[] as $k | "\($k) \(.[$k] | .items[].metadata.name)"'`,
		SilenceUsage: true,
		Version:      version,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, _ = zap.NewProduction()
			if mc.Debug {
				logger, _ = zap.NewDevelopment()
			}
			defer logger.Sync()
			if mc.Output != "" {
				if _, ok := outputs[mc.Output]; !ok {
					return errUnknownOutput
				}
				args = append(args, "-o", "json")
			}
			if len(args) == 0 && !mc.ListOnly {
				cmd.Usage()
				return nil
			}
			return mc.run(args)
		},
	}

	cmd.Flags().StringVarP(&mc.Regex, "regex", "r", mc.Regex, "a regex to filter the list of context names in kubeconfig. If not given all contexts are used")
	cmd.Flags().StringVarP(&mc.NegRegex, "negative-regex", "x", mc.NegRegex, "a regex to exclude matches from the result set. Evaluated succeeding to the including regex filter")
	cmd.Flags().StringVarP(&mc.Namespaces, "namespaces", "n", mc.Namespaces, "comma-separated list of namespaces. Overrides namespace(s) specified in kubectl command. The default is the current namespace of the context")
	cmd.Flags().BoolVarP(&mc.ListOnly, "list-only", "l", mc.ListOnly, "just list the contexts matching the regex. Good for testing your regex")
	cmd.Flags().IntVarP(&mc.MaxProc, "max-processes", "p", 5, "max amount of parallel kubectl to be executed. Can be used to limit cpu activity")
	cmd.Flags().BoolVarP(&mc.Debug, "debug", "d", mc.Debug, "enable debug output")
	cmd.Flags().StringVarP(&mc.Output, "output", "o", mc.Output, fmt.Sprintf("specify the output format. Useful for parsing with another tool like jq or yq. One of %s", outputsString()))

	mc.Cmd = cmd

	return mc
}

// run executed the main command by listing matched kubernetes contexts and executing the
// given kubectl args against every context in parallel
// The `do` parameter allows for dependency injection
func (mc *MC) run(args []string) error {
	contexts, err := mc.listContexts(mc.getListContextsCmd())
	if err != nil {
		return err
	}

	if mc.ListOnly {
		for _, c := range contexts {
			fmt.Fprintln(mc.Cmd.OutOrStdout(), c)
		}
		return nil
	}

	logger.Debug("preparing wait group", zap.Int("max-processes", mc.MaxProc))
	parallelProc := make(chan bool, mc.MaxProc)
	for i := 0; i < mc.MaxProc; i++ {
		parallelProc <- true
	}

	done := make(chan bool)
	wait := make(chan bool)
	var mutex = &sync.Mutex{}

	namespaces := strings.Split(mc.Namespaces, ",")

	logger.Debug("start wait group")
	go func() {
		for i := 0; i < len(contexts)*len(namespaces); i++ {
			<-done
			parallelProc <- true
		}
		logger.Debug("wait group finished")
		wait <- true
	}()

	output := map[string]json.RawMessage{}
	for _, c := range contexts {
		for _, ns := range namespaces {
			logger.Debug("waiting for next free spot", zap.String("context", c), zap.String("namespace", ns))
			<-parallelProc
			logger.Debug("executing", zap.String("context", c), zap.String("namespace", ns))
			go do(done, c, ns, output, mc.Output == "", mc.Cmd.OutOrStdout(), mc.getKubectlCmd(args, c, ns), mutex)
		}
	}
	<-wait
	if mc.Output != "" {
		logger.Debug("parsing output...")
		o, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			logger.Debug("failed to parse output", zap.String("retrieved", fmt.Sprintf("%s", output)))
			return errCouldntParseOutput
		}
		switch mc.Output {
		case JSON:
			fmt.Fprintf(mc.Cmd.OutOrStdout(), "%s", o)
		case YAML:
			o, err := yaml.JSONToYAML(o)
			if err != nil {
				return err
			}
			fmt.Fprintf(mc.Cmd.OutOrStdout(), "%s", o)
		}
	}
	logger.Debug("done")

	return nil
}

// list context builds a list of context based on a given regex
func (mc *MC) listContexts(cmd Cmd) (contexts []string, err error) {
	r, err := regexp.Compile(mc.Regex)
	if err != nil {
		return nil, err
	}
	nr, err := regexp.Compile(mc.NegRegex)
	if err != nil {
		return nil, err
	}

	stdout, err := kubectl(cmd)
	if err != nil {
		return nil, err
	}

	s := bufio.NewScanner(bytes.NewReader(stdout))
	for s.Scan() {
		context := s.Bytes()
		if r.Match(context) {
			if mc.NegRegex != "" && nr.Match(context) {
				continue
			}
			contexts = append(contexts, string(context))
		}
	}

	return
}

// do executes a command against kubectl and sends a bool to the done channel when done
func do(done chan bool, context string, namespace string, output map[string]json.RawMessage, writeToStdout bool, out io.Writer, cmd Cmd, mutex *sync.Mutex) {
	stdout, err := kubectl(cmd)
	if err != nil {
		stdout = []byte(err.Error())
	}
	mutex.Lock()

	cns := context
	if namespace != "" {
		cns += ": " + namespace
	}
	output[cns] = stdout
	mutex.Unlock()
	if writeToStdout {
		fmt.Fprint(out, formatContext(context, namespace, stdout))
	}
	done <- true
}

// kubectl executes a kubectl command
func kubectl(cmd Cmd) ([]byte, error) {
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf(strings.Replace(strings.Replace(string(out), "error: ", "", -1), "Error: ", "", -1))
	}
	return out, nil
}

// getLocalArgs transforms kubectl args slice by injecting the context flag into the right position.
// if the kubectl command contained `--` (for instance for a `kubectl exec` command, we inject the context flag before
// that.
func getLocalArgs(args []string, context string, namespace string) (localArgs []string) {

	var skipContext bool
	for _, arg := range args {
		if arg == "--" {
			// If this is given, we need to insert the context before this arg
			localArgs = append(localArgs, "--context", context)
			if len(namespace) > 0 {
				localArgs = append(localArgs, "--namespace", namespace)
			}
			skipContext = true
		}
		localArgs = append(localArgs, arg)
	}
	if !skipContext {
		localArgs = append(localArgs, "--context", context)
		if len(namespace) > 0 {
			localArgs = append(localArgs, "--namespace", namespace)
		}
	}
	return
}

// outputStrings is a helper function to transform the output option map keys into a string separated by `|`
// It can be used for helpful docstrings
func outputsString() string {
	keys := make([]string, 0, len(outputs))
	for k := range outputs {
		keys = append(keys, k)
	}
	return strings.Join(keys, "|")
}

// formatContext returns a formated strings with the context has header, separated from the contents by a divider
func formatContext(context string, namespace string, stdout []byte) string {
	if namespace != "" {
		namespace = ": " + namespace
	}
	return fmt.Sprintf("\n%s%s\n%s\n%s", context, namespace, strings.Repeat("-", len(context)+len(namespace)), string(stdout))
}
