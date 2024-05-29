package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/midbel/sweet/internal/config"
)

func main() {
	flag.Parse()

	r, err := os.Open(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	defer r.Close()

	ps, err := config.New(r)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if err := ps.Parse(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// scan, err := config.Scan(r)
	// if err != nil {
	// 	fmt.Fprintln(os.Stderr, err)
	// 	os.Exit(1)
	// }
	// for {
	// 	tok := scan.Scan()
	// 	if tok.Type == config.Invalid {
	// 		fmt.Fprintf(os.Stderr, "invalid token found at %s", tok.Position)
	// 		fmt.Fprintln(os.Stderr)
	// 		os.Exit(1)
	// 	}
	// 	if tok.Type == config.EOF {
	// 		break
	// 	}
	// 	fmt.Println(tok.Position, tok)
	// }
}
