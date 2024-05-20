package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/midbel/sweet/internal/lang"
)

func main() {
	flag.Parse()

	r, err := os.Open(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	defer r.Close()

	w := lang.NewWriter(os.Stdout)
	if err := w.Format(r); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
