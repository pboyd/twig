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

	flagName, flagSet, rest, err := cli.ExtractProfileFlag(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	profileName := cli.ResolveProfileName(flagName, flagSet)

	if len(rest) == 0 && term.IsTerminal(int(os.Stdout.Fd())) {
		if err := tui.Run(context.Background(), profileName); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	os.Exit(cli.Run(profileName, rest))
}
