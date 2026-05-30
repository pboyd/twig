package main

import (
	"context"
	"fmt"
	"os"

	"github.com/pboyd/twig/services/twig/internal/cli"
	"github.com/pboyd/twig/services/twig/internal/tui"
	"golang.org/x/term"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 && term.IsTerminal(int(os.Stdout.Fd())) {
		if err := tui.Run(context.Background()); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	os.Exit(cli.Run(args))
}
