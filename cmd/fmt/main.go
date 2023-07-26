package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/midbel/sweet/internal/dialect"
)

var (
	vendor   = flag.String("d", "", "dialect")
	compact  = flag.Bool("c", false, "compact statement")
	upper    = flag.Bool("u", false, "uppercase keyword")
	tab      = flag.Bool("t", false, "use tab as indent character")
	count    = flag.Int("n", 1, "number of space for indent")
	file     = flag.String("f", "", "write sql to file")
	keep     = flag.Bool("k", false, "keep comment in output")
	colorize = flag.Bool("C", false, "colorize output")
)

func main() {
	flag.Parse()

	var out io.Writer = os.Stdout
	if *file != "" {
		w, err := os.Create(*file)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		defer w.Close()
		out = w
	}

	w, err := prepareWriter(out, *vendor)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	for _, f := range flag.Args() {
		if err := formatFile(w, f); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}

func prepareWriter(ws io.Writer, vendor string) (dialect.Writer, error) {
	w, err := dialect.NewWriter(ws, vendor)
	if err != nil {
		return nil, err
	}
	w.SetKeywordUppercase(*upper)
	w.SetFunctionUppercase(*upper)
	w.SetKeepComments(*keep)
	w.SetCompact(*compact)
	w.ColorizeOutput(*colorize)
	if *tab {
		w.SetIndent("\t")
	} else if *count > 0 {
		w.SetIndent(strings.Repeat(" ", *count))
	}
	return w, nil
}

func formatFile(w dialect.Writer, file string) error {
	r, err := os.Open(file)
	if err != nil {
		return err
	}
	defer r.Close()
	return w.Format(r)
}
