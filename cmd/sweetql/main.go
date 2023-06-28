package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/midbel/sweet/sql"
)

func main() {
	scan := flag.Bool("s", false, "scanning mode")
	flag.Parse()

	r, err := os.Open(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	defer r.Close()

	if *scan {
		err = scanReader(r)
	} else {
		err = parseReader(r)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func scanReader(r io.Reader) error {
	scan, err := sql.Scan(r)
	if err != nil {
		return err
	}
	for {
		tok := scan.Scan()
		if tok.Type == sql.Invalid {
			fmt.Fprintf(os.Stderr, "invalid token detected %s", tok)
			fmt.Fprintln(os.Stderr)
		}
		if tok.Type == sql.EOF {
			break
		}
		fmt.Printf("%d:%d = %s", tok.Line, tok.Column, tok)
		fmt.Println()
	}
	return nil
}

func parseReader(r io.Reader) error {
	p, err := sql.NewParser(r)
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
		fmt.Printf("%#v", stmt)
		fmt.Println()
	}
	return nil
}

