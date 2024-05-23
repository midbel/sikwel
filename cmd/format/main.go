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
	flag.BoolVar(&w.AllUpper, "upper-all", w.AllUpper, "all identifiers and keywords to uppercase")
	flag.BoolVar(&w.WithAs, "with-as", w.WithAs, "set as keyword to define alias")
	flag.BoolVar(&w.InlineCte, "inline-cte", w.InlineCte, "inline cte")
	flag.BoolVar(&w.QuoteIdent, "quote-ident", w.QuoteIdent, "quote identifier")
	flag.BoolVar(&w.UseNames, "use-names", w.UseNames, "use names from select to create fields list of view and/or cte")

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
