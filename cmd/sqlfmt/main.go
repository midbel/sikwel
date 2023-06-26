package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/midbel/sikwel"
	"github.com/midbel/sikwel/rest"
)

func main() {
	var (
		dialect = flag.String("d", "", "dialect")
		listen  = flag.Bool("l", false, "listen")
		err     error
	)
	flag.Parse()

	if *listen {
		err = serve(flag.Arg(0))
	} else {
		err = format(flag.Arg(0), *dialect)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func serve(addr string) error {
	http.HandleFunc("/format", rest.Format)
	return http.ListenAndServe(addr, nil)
}

func format(file, dialect string) error {
	r, err := os.Open(file)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	defer r.Close()

	w := sikwel.NewWriter(os.Stdout)
	return w.Format(r, sikwel.KeywordsForDialect(dialect))
}
