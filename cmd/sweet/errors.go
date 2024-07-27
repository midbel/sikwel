package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/midbel/sweet/internal/lang/parser"
)

func reportError(err error) {
	var pserr parser.ParseError
	if !errors.As(err, &pserr) {
		fmt.Println(err)
		return
	}
	var (
		parts = strings.Split(pserr.Query, "\n")
		pos   = pserr.Position()
		first = pos.Line - 3
	)
	if pos.Line < len(parts) {
		parts = parts[:pos.Line]
	}

	for i := range parts {
		var (
			lino = pos.Line - len(parts) + i + 1
			line = strings.TrimSpace(parts[i])
		)
		if lino < first {
			continue
		}
		fmt.Printf("%03d | %s", lino, line)
		fmt.Println()
	}
	fmt.Print(strings.Repeat(" ", 6+pos.Column-1))
	if str := pserr.Literal(); len(str) > 0 {
		fmt.Println(strings.Repeat("^", len(str)))
	} else {
		fmt.Println("^")
	}
	fmt.Println(pserr)
	fmt.Println()
}
