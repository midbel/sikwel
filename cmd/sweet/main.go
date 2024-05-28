package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/midbel/sweet/internal/lang"
)

func main() {
	flag.Parse()

	var err error
	switch n := flag.Arg(0); n {
	case "format", "fmt":
		err = runFormat(flag.Args())
	case "lint", "check", "verify":
		err = runLint(flag.Args())
	default:
		err = fmt.Errorf("unknown command %s", n)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func runFormat(args []string) error {
	var (
		set flag.FlagSet
		wtr = lang.NewWriter(os.Stdout)
	)
	if err := set.Parse(args); err != nil {
		return err
	}
	process := func(file string) error {
		r, err := os.Open(file)
		if err != nil {
			return err
		}
		defer r.Close()
		return wtr.Format(r)
	}
	for _, f := range set.Args() {
		if err := process(f); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
	return nil
}

func runLint(args []string) error {
	var (
		set flag.FlagSet
		ltr = lang.NewLinter()
	)
	if err := set.Parse(args); err != nil {
		return err
	}
	process := func(file string) ([]lang.LintMessage, error) {
		r, err := os.Open(file)
		if err != nil {
			return nil, err
		}
		return ltr.Lint(r)
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
