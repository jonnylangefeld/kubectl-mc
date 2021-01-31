package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var (
	version string
	logger  *zap.Logger
)

func main() {
	mc := NewMC()
	if err := mc.Execute(); err != nil {
		os.Exit(1)
	}
}

// MC contains the options of the command
type MC struct {
	Regex    string
	NegRegex string
	ListOnly bool
	MaxProc  int
	Debug    bool
}

// NewMC registers the default mc command
func NewMC() *cobra.Command {
	mc := &MC{}

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
mc -r gke -n 'dev' -p 5 -d -- get pods -n gatekeeper-system -l app.kubernetes.io/name=audit`,
		SilenceUsage: true,
		Version:      version,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, _ = zap.NewProduction()
			if mc.Debug {
				logger, _ = zap.NewDevelopment()
			}
			defer logger.Sync()
			if len(args) == 0 && !mc.ListOnly {
				cmd.Usage()
				return nil
			}
			return mc.run(args)
		},
	}

	cmd.Flags().StringVarP(&mc.Regex, "regex", "r", mc.Regex, "a regex to filter the list of context names in kubeconfig. If not given all contexts are used")
	cmd.Flags().StringVarP(&mc.NegRegex, "negative-regex", "n", mc.NegRegex, "a regex to exclude matches from the result set. Evaluated succeeding to the including regex filter")
	cmd.Flags().BoolVarP(&mc.ListOnly, "list-only", "l", mc.ListOnly, "just list the contexts matching the regex. Good for testing your regex")
	cmd.Flags().IntVarP(&mc.MaxProc, "max-processes", "p", 100, "max amount of parallel kubectl to be exectued. Can be used to limit cpu activity")
	cmd.Flags().BoolVarP(&mc.Debug, "debug", "d", mc.Debug, "enable debug output")

	return cmd
}

// run executed the main command by listing matched kubernetes contexts and executing the
// given kubectl args against every context in parallel
func (mc *MC) run(args []string) error {
	contexts, err := mc.listContexts()
	if err != nil {
		return err
	}

	if mc.ListOnly {
		for _, c := range contexts {
			fmt.Println(c)
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

	logger.Debug("start wait group")
	go func() {
		for i := 0; i < len(contexts); i++ {
			<-done
			parallelProc <- true
		}
		logger.Debug("wait group finished")
		wait <- true
	}()

	for _, c := range contexts {
		logger.Debug("waiting for next free spot", zap.String("context", c))
		<-parallelProc
		logger.Debug("executing", zap.String("context", c))
		go do(done, c, args)
	}
	<-wait
	logger.Debug("done")

	return nil
}

// list context builds a list of context based on a given regex
func (mc *MC) listContexts() (contexts []string, err error) {
	r, err := regexp.Compile(mc.Regex)
	if err != nil {
		return nil, err
	}
	nr, err := regexp.Compile(mc.NegRegex)
	if err != nil {
		return nil, err
	}

	args := []string{"config", "get-contexts", "-o", "name"}

	stdout, err := kubectl(args)
	if err != nil {
		return nil, err
	}

	s := bufio.NewScanner(stdout)
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
func do(done chan bool, c string, args []string) {
	var localArgs []string
	var skipContext bool
	for _, arg := range args {
		if arg == "--" {
			// If this is given, we need to insert the context before this arg
			localArgs = append(localArgs, "--context", c)
			skipContext = true
		}
		localArgs = append(localArgs, arg)
	}
	if !skipContext {
		localArgs = append(localArgs, "--context", c)
	}
	stdout, err := kubectl(localArgs)
	if err != nil {
		stdout = bytes.NewBuffer([]byte(err.Error()))
	}
	fmt.Printf("\n%s\n%s\n%s", c, strings.Repeat("-", len(c)), string(stdout.Bytes()))
	done <- true
}

// kubectl executes a kubectl command
func kubectl(args []string) (*bytes.Buffer, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("kubectl", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf(strings.Replace(strings.Replace(stderr.String(), "error: ", "", -1), "Error:", "", -1))
	}
	return &stdout, nil
}
