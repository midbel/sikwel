package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/midbel/sweet/internal/db2"
	"github.com/midbel/sweet/internal/lang"
	"github.com/midbel/sweet/internal/lang/complexity"
	"github.com/midbel/sweet/internal/lang/lint"
	"github.com/midbel/sweet/internal/lang/parser"
	"github.com/midbel/sweet/internal/scanner"
	"github.com/midbel/sweet/internal/token"
)

func main() {
	flag.Parse()

	var (
		err error
		cmd func([]string) error
	)
	switch n := flag.Arg(0); n {
	case "scan":
		cmd = runScan
	case "parse":
		cmd = runParse
	case "format", "fmt":
		cmd = runFormat
	case "lint", "check", "verify":
		cmd = runLint
	case "debug", "ast":
		cmd = runDebug
	case "cyclo", "measure":
		cmd = runCyclo
	default:
		err = fmt.Errorf("unknown command %s", n)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	args := flag.Args()
	if err = cmd(args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runParse(args []string) error {
	var (
		set     = flag.NewFlagSet("parse", flag.ExitOnError)
		dialect string
	)
	set.StringVar(&dialect, "dialect", "", "SQL dialect")
	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {

		}
		return err
	}
	r, err := os.Open(set.Arg(0))
	if err != nil {
		return err
	}
	defer r.Close()

	var ps lang.Parser
	switch dialect {
	case "db2":
		ps, err = db2.Parse(r)
	default:
		ps, err = parser.NewParser(r)
	}
	if err != nil {
		return err
	}
	for {
		stmt, err := ps.Parse()
		if errors.Is(err, io.EOF) {
			break
		}
		fmt.Printf("%+v\n", stmt)
	}
	return nil
}

func runScan(args []string) error {
	var (
		set     = flag.NewFlagSet("scan", flag.ExitOnError)
		dialect string
	)
	set.StringVar(&dialect, "dialect", "", "SQL dialect")
	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {

		}
		return err
	}
	r, err := os.Open(set.Arg(0))
	if err != nil {
		return err
	}
	defer r.Close()

	var scan *scanner.Scanner
	switch dialect {
	case "db2":
		scan, err = db2.Scan(r)
	default:
		scan, err = scanner.Scan(r, lang.GetKeywords())
	}
	if err != nil {
		return err
	}
	for !scan.Done() {
		tok := scan.Scan()
		if tok.Type == token.EOF {
			break
		}
		fmt.Println(tok)
	}
	return nil
}

func runLint(args []string) error {
	var (
		set     = flag.NewFlagSet("lint", flag.ExitOnError)
		linter  = lint.NewLinter()
		dialect string
		config  string
		init    bool
	)
	set.StringVar(&config, "config", "", "linter configuration")
	set.StringVar(&dialect, "dialect", "", "SQL dialect")
	set.BoolVar(&init, "init", false, "create linter configuration file")
	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {

		}
		return err
	}
	if init {
		return runInit()
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

func runInit() error {
	return nil
}

func runCyclo(files []string) error {
	run := func(f string) (int, error) {
		r, err := os.Open(f)
		if err != nil {
			return 0, err
		}
		defer r.Close()
		return complexity.Complexity(r)
	}
	for _, f := range files {
		n, err := run(f)
		if err != nil {
			return err
		}
		fmt.Printf("%s: %d", f, n)
		fmt.Println()
	}
	return nil
}

func runDebug(files []string) error {
	for _, f := range files {
		if err := printTree(f); err != nil {
			return err
		}
	}
	return nil
}

func printTree(file string) error {
	r, err := os.Open(file)
	if err != nil {
		return err
	}
	defer r.Close()

	p, err := parser.NewParser(r)
	if err != nil {
		return err
	}
	for {
		stmt, err := p.Parse()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
		_ = stmt
	}
	return nil
}
