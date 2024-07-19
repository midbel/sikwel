package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

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
		// var pserr parser.ParseError
		// if errors.As(err, &pserr) {
		// 	fmt.Println(pserr.Query)
		// 	fmt.Printf(">> %+v\n", pserr)
		// }
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

	scan, err := scanner.Scan(r, lang.GetKeywords())
	if err != nil {
		return err
	}
	for !scan.Done() {
		tok := scan.Scan()
		if tok.Type == token.EOL || tok.Type == token.EOF {
			fmt.Println(">>", scan.Query())
		}
		if tok.Type == token.EOF {
			break
		}
		fmt.Println(tok)
	}
	return nil
}
