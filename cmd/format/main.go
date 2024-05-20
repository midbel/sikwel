package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/midbel/sweet/internal/lang"
)

func main() {
	w := lang.NewWriter(os.Stdout)

	flag.BoolVar(&w.Compact, "compact", w.Compact, "compact")
	flag.BoolVar(&w.KwUpper, "upper-kw", w.KwUpper, "sql keyword to uppercase")
	flag.BoolVar(&w.FnUpper, "upper-fn", w.FnUpper, "functions to uppercase")
	flag.BoolVar(&w.WithAs, "with-as", w.WithAs, "set as keyword to define alias")

	flag.Parse()

	r, err := os.Open(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	defer r.Close()

	if err := w.Format(r); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
