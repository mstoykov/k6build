// Package main contains CLI documentation generator tool.
package main

import (
	"github.com/grafana/clireadme"
	"github.com/grafana/k6build/cmd"
)

func main() {
	clireadme.Main(cmd.New(), 0)
}
