package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/midbel/sweet/internal/dialect"
	"github.com/midbel/sweet/internal/rest"
)

func main() {
	var (
		vendor = flag.String("d", "", "dialect")
		listen = flag.Bool("l", false, "listen")
		err    error
	)
	flag.Parse()

	if *listen {
		err = runServe(flag.Arg(0))
	} else {
		err = runFormat(flag.Arg(0), *vendor)
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

func runFormat(file, vendor string) error {
	r, err := os.Open(file)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	defer r.Close()

	w, err := dialect.NewWriter(os.Stdout, vendor)
	if err != nil {
		return err
	}
	return w.Format(r)
}
