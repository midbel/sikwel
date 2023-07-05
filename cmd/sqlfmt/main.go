package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/midbel/sweet/internal/format"
	"github.com/midbel/sweet/internal/lang"
	"github.com/midbel/sweet/rest"
)

func main() {
	var (
		dialect = flag.String("d", "", "dialect")
		listen  = flag.Bool("l", false, "listen")
		err     error
	)
	flag.Parse()

	if *listen {
		err = runServe(flag.Arg(0))
	} else {
		err = runFormat(flag.Arg(0), *dialect)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runServe(addr string) error {
	http.HandleFunc("/format", rest.Format)
	return http.ListenAndServe(addr, nil)
}

func runFormat(file, dialect string) error {
	r, err := os.Open(file)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	defer r.Close()

	w := format.NewWriter(os.Stdout)
	return w.Format(r, lang.GetKeywords())
}
