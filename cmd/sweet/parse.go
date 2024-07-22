package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/midbel/sweet/internal/lang"
	"github.com/midbel/sweet/internal/lang/parser"
	"github.com/midbel/sweet/internal/scanner"
	"github.com/midbel/sweet/internal/token"
)

func runParse(args []string) error {
	var (
		set     = flag.NewFlagSet("parse", flag.ExitOnError)
		dialect string
	)
	set.StringVar(&dialect, "dialect", "", "SQL dialect")
	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	r, err := os.Open(set.Arg(0))
	if err != nil {
		return err
	}
	defer r.Close()

	ps, err := parser.NewParser(r)
	if err != nil {
		return err
	}
	for {
		stmt, err := ps.Parse()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			reportError(err)
			continue
		}
		fmt.Printf("%+v\n", stmt)
	}
	return nil
}

func reportError(err error) {
	var pserr parser.ParseError
	if !errors.As(err, &pserr) {
		fmt.Println(err)
		return
	}
	var (
		parts = strings.Split(pserr.Query, "\n")
		pos   = pserr.Position()
		first = pos.Line - 3
	)
	if pos.Line < len(parts) {
		parts = parts[:pos.Line]
	}

	for i := range parts {
		var (
			lino = pos.Line - len(parts) + i + 1
			line = strings.TrimSpace(parts[i])
		)
		if lino < first {
			continue
		}
		fmt.Printf("%03d | %s", lino, line)
		fmt.Println()
	}
	fmt.Print(strings.Repeat(" ", 6+pos.Column-1))
	if str := pserr.Literal(); len(str) > 0 {
		fmt.Println(strings.Repeat("^", len(str)))
	} else {
		fmt.Println("^")
	}
	fmt.Println(pserr)
	fmt.Println()
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

	scan, err := scanner.Scan(r, lang.GetKeywords())
	if err != nil {
		return err
	}
	for !scan.Done() {
		tok := scan.Scan()
		if tok.Type == token.EOF {
			break
		}
		pos := tok.Position
		fmt.Printf("%d:%d, %s", pos.Line, pos.Column, tok)
		fmt.Println()
	}
	return nil
}
