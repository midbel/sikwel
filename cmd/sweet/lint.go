package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/midbel/sweet/internal/lang/lint"
)

func runLint(args []string) error {
	var (
		set     = flag.NewFlagSet("lint", flag.ExitOnError)
		linter  = lint.NewLinter()
		dialect string
		config  string
	)
	set.StringVar(&config, "config", "", "linter configuration")
	set.StringVar(&dialect, "dialect", "", "SQL dialect")
	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {

		}
		return err
	}
	process := func(file string) ([]lint.LintMessage, error) {
		r, err := os.Open(file)
		if err != nil {
			return nil, err
		}
		return linter.Lint(r)
	}
	for _, f := range set.Args() {
		list, err := process(f)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		for _, m := range list {
			fmt.Fprintf(os.Stdout, "%s (%s): %s", m.Rule, m.Severity, m.Message)
			fmt.Fprintln(os.Stdout)
		}
	}
	return nil
}
