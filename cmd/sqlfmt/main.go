package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/midbel/sweet/internal/dialect"
)

func main() {
	vendor := flag.String("d", "", "dialect")
	flag.Parse()

	err := runFormat(flag.Arg(0), *vendor)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runFormat(file, vendor string) error {
	r, err := os.Open(file)
	if err != nil {
		return err
	}
	defer r.Close()

	w, err := dialect.NewWriter(os.Stdout, vendor)
	if err != nil {
		return err
	}
	return w.Format(r)
}
