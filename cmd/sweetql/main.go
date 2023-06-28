package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/midbel/sweet/sql"
)

func main() {
	flag.Parse()

	r, err := os.Open(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	defer r.Close()

	if err = scanReader(r); err != nil {
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
