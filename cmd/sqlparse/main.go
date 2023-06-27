package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/midbel/sweet"
)

func main() {
	var (
		dialect = flag.String("d", "", "dialect")
		jsonify = flag.Bool("j", false, "jsonify parsed query")
	)
	flag.Parse()

	r, err := os.Open(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	defer r.Close()

	p, err := sweet.NewParser(r, sweet.KeywordsForDialect(*dialect))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	for i := 1; ; i++ {
		stmt, err := p.Parse()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			fmt.Fprintf(os.Stderr, "parsing query #%d fails - %s", i, err)
			fmt.Fprintln(os.Stderr)
			continue
		}
		if *jsonify {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "    ")
			if err := enc.Encode(stmt); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(2)
			}
		} else {
			fmt.Printf("%d: %#v", i, stmt)
			fmt.Println()
		}
	}
}
