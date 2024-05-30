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

	cfg, err := config.Load(r)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	sub := cfg.Sub("indent")
	fmt.Println("space", sub.GetBool("space"))
	fmt.Println("count", sub.GetInt("count"))
}
