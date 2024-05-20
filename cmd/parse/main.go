package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/midbel/sweet/internal/lang"
)

func main() {
	var (
		jsonify   = flag.Bool("j", false, "jsonify parsed query")
		inlineCte = flag.Bool("i", false, "inline cte in query")
	)
	flag.Parse()

	r, err := os.Open(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	defer r.Close()

	if err := parseReader(r, *inlineCte, *jsonify); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func parseReader(r io.Reader, inlineCte, jsonify bool) error {
	p, err := lang.NewParser(r)
	if err != nil {
		return err
	}
	p.SetInline(inlineCte)
	for i := 1; ; i++ {
		stmt, err := p.Parse()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("parsing query #%d fails - %s", i, err)
		}
		if jsonify {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "    ")
			if err := enc.Encode(stmt); err != nil {
				return err
			}
		} else {
			fmt.Printf("%d: %#v", i, stmt)
			fmt.Println()
		}
	}
	return nil
}
