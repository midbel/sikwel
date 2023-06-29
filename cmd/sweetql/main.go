package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/midbel/sweet/sql"
)

func main() {
	var (
		scan    = flag.Bool("s", false, "scanning mode")
		jsonify = flag.Bool("j", false, "jsonify")
	)
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
		err = parseReader(r, *jsonify)
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

func parseReader(r io.Reader, jsonify bool) error {
	p, err := sql.NewParser(r)
	if err != nil {
		return err
	}
	e := json.NewEncoder(os.Stdout)
	e.SetIndent("", strings.Repeat(" ", 2))
	for {
		stmt, err := p.Parse()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
		if !jsonify {
			fmt.Printf("%#v", stmt)
			fmt.Println()
			continue
		}
		if err = e.Encode(stmt); err != nil {
			return err
		}
	}
	return nil
}
