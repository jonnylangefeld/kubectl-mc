package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"

	"github.com/spf13/cobra"
)

var (
	version string
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
	ListOnly bool
}

// NewMC registers the default mc command
func NewMC() *cobra.Command {
	mc := &MC{}

	cmd := &cobra.Command{
		Use:   "mc [flags] -- [kubectl command]",
		Short: "Run kubectl commands against multiple clusters at once",
		Example: `
mc -r kind -l
mc -r dev -- get pods -n kube-system
mc --regex kind -- run debug --image=markeijsermans/debug --command -- sleep infinity`,
		SilenceUsage: true,
		Version:      version,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && !mc.ListOnly {
				cmd.Usage()
				return nil
			}
			return mc.run(args)
		},
	}

	cmd.Flags().StringVarP(&mc.Regex, "regex", "r", mc.Regex, "a regex to filter the list of context names in kubeconfig. If not given all contexts are used")
	cmd.Flags().BoolVarP(&mc.ListOnly, "list-only", "l", mc.ListOnly, "just list the contexts matching the regex. Good for testing your regex")
	return cmd
}

// run executed the main command by listing matched kubernetes contexts and executing the
// given kubectl args against every context in parallel
func (mc *MC) run(args []string) error {
	contexts, err := mc.listContexts()
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	for _, c := range contexts {
		if mc.ListOnly {
			fmt.Println(c)
			continue
		}

		wg.Add(1)
		go func(wg *sync.WaitGroup, c string, args []string) {
			defer wg.Done()
			localArgs := []string{}
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
		}(&wg, c, args)
	}
	wg.Wait()

	return nil
}

// list context builds a list of context based on a given regex
func (mc *MC) listContexts() (contexts []string, err error) {
	r, err := regexp.Compile(mc.Regex)
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
			contexts = append(contexts, string(context))
		}
	}

	return
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
