package main

import (
	"os"

	_ "github.com/golang/mock/mockgen/model"
	"github.com/jonnylangefeld/kubectl-mc/pkg/mc"
)

var (
	version string
)

func main() {
	mc := mc.New(version)
	if err := mc.Cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
