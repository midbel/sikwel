package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/midbel/sweet/internal/dialect"
)

func main() {
	var (
		vendor  = flag.String("d", "", "dialect")
		compact = flag.Bool("c", false, "compact statement")
		upper   = flag.Bool("u", false, "uppercase keyword")
		tab     = flag.Bool("t", false, "use tab as indent character")
		count   = flag.Int("n", 1, "number of space for indent")
	)
	flag.Parse()

	w, err := dialect.NewWriter(os.Stdout, *vendor)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	w.SetKeywordUppercase(*upper)
	w.SetCompact(*compact)
	if *tab {
		w.SetIndent("\t")
	} else if *count > 0 {
		w.SetIndent(strings.Repeat(" ", *count))
	}
	for _, f := range flag.Args() {
		if err := formatFile(w, f); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}

func formatFile(w dialect.Writer, file string) error {
	r, err := os.Open(file)
	if err != nil {
		return err
	}
	defer r.Close()
	return w.Format(r)
}
