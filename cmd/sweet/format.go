package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/midbel/sweet/internal/config"
	"github.com/midbel/sweet/internal/lang"
	"github.com/midbel/sweet/internal/lang/format"
	"github.com/midbel/sweet/internal/ms"
	"github.com/midbel/sweet/internal/my"
	// "github.com/midbel/sweet/internal/db2"
)

func runFormat(args []string) error {
	var (
		set    = flag.NewFlagSet("format", flag.ExitOnError)
		writer = format.NewWriter(os.Stdout)
	)
	set.BoolVar(&writer.Compact, "compact", writer.Compact, "produces compact SQL queries")
	set.BoolVar(&writer.UseAs, "use-as", writer.UseAs, "always use as to define alias")
	set.BoolVar(&writer.UseQuote, "use-quote", writer.UseQuote, "quote all identifier")
	set.IntVar(&writer.UseIndent, "use-indent", writer.UseIndent, "number of space to use to indent SQL")
	set.BoolVar(&writer.UseSpace, "use-space", writer.UseSpace, "use tabs instead of space to indent SQL")
	set.BoolVar(&writer.UseColor, "use-color", writer.UseColor, "colorify SQL keywords, identifiers")
	set.BoolVar(&writer.UseCrlf, "use-crlf", writer.UseCrlf, "use crlf for newline")
	set.BoolVar(&writer.PrependComma, "prepend-comma", writer.PrependComma, "write comma before expressions")
	set.BoolVar(&writer.KeepComment, "keep-comment", writer.KeepComment, "keep comments")

	set.Func("dialect", "SQL dialect", func(value string) error {
		formatter, err := getFormatterForDialect(value)
		if err == nil {
			writer.Formatter = formatter
		}
		return err
	})
	set.Func("rewrite", "rewrite rules to apply", rewriteRules(writer))
	set.Func("upper", "upperize mode", upperizeRules(writer))
	set.Func("config", "formatter configuration file", configureRules(writer))

	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {

		}
		return err
	}
	process := func(file string) error {
		r, err := os.Open(file)
		if err != nil {
			return err
		}
		defer r.Close()
		return writer.Format(r)
	}
	for _, f := range set.Args() {
		if err := process(f); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
	return nil
}

func configureRules(writer *format.Writer) func(string) error {
	var (
		rewrite  = rewriteRules(writer)
		upperize = upperizeRules(writer)
	)
	return func(file string) error {
		r, err := os.Open(file)
		if err != nil {
			return err
		}
		cfg, err := config.Load(r)
		if err != nil {
			return err
		}
		cfg = cfg.Sub("format")
		var (
			syntax = cfg.Sub("syntax")
			indent = cfg.Sub("indent")
		)
		writer.Compact = cfg.GetBool("compact")
		writer.UseQuote = syntax.GetBool("quote")
		writer.UseAs = syntax.GetBool("as")
		writer.UseIndent = int(indent.GetInt("count"))
		writer.UseSpace = indent.GetBool("space")
		cfg.Apply("comma", func(value any) error {
			switch value.(string) {
			case "before", "prepend":
				writer.PrependComma = true
			case "after", "":
			default:
			}
			return nil
		})
		cfg.Apply("comment", func(value any) error {
			switch value.(string) {
			case "keep":
				writer.KeepComment = true
			case "discard", "":
			default:
			}
			return nil
		})
		cfg.Apply("newline", func(value any) error {
			switch value.(string) {
			case "crlf":
				writer.UseCrlf = true
			case "nl", "lf", "":
			default:
			}
			return nil
		})
		for _, r := range cfg.GetStrings("upperize") {
			upperize(strings.ReplaceAll(r, "_", "-"))
		}
		for _, r := range cfg.GetStrings("rewrite") {
			rewrite(strings.ReplaceAll(r, "_", "-"))
		}
		return nil
	}
}

func upperizeRules(writer *format.Writer) func(string) error {
	return func(value string) error {
		switch value {
		case "all", "":
			writer.Upperize |= format.UpperId | format.UpperKw | format.UpperFn | format.UpperType
		case "keyword", "kw":
			writer.Upperize |= format.UpperKw
		case "function", "fn":
			writer.Upperize |= format.UpperFn
		case "identifier", "ident", "id":
			writer.Upperize |= format.UpperId
		case "type":
			writer.Upperize |= format.UpperType
		case "none":
			writer.Upperize = format.UpperNone
		default:
		}
		return nil
	}
}

func rewriteRules(writer *format.Writer) func(string) error {
	return func(value string) error {
		switch value {
		case "all", "":
			writer.Rules |= format.RewriteAll
		case "use-std-op":
			writer.Rules |= format.RewriteStdOp
		case "use-std-expr":
			writer.Rules |= format.RewriteStdExpr
		case "missing-cte-alias":
			writer.Rules |= format.RewriteMissCteAlias
		case "missing-view-alias":
			writer.Rules |= format.RewriteMissViewAlias
		case "subquery-as-cte":
			writer.Rules |= format.RewriteWithCte
		case "cte-as-subquery":
			writer.Rules |= format.RewriteWithSubqueries
		case "join-as-subquery":
			writer.Rules |= format.RewriteJoinSubquery
		case "join-without-literal":
			writer.Rules |= format.RewriteJoinPredicate
		default:
		}
		return nil
	}
}

func getFormatterForDialect(name string) (lang.Formatter, error) {
	switch name {
	case "my", "mysql":
		return my.GetFormatter(), nil
	case "mssql":
		return ms.GetFormatter(), nil
	case "ansi", "pg", "postgres", "sqlite", "lite":
		return format.GetFormatter(), nil
	case "db2":
		return format.GetFormatter(), nil
	default:
		return nil, fmt.Errorf("%s unsupported dialect", name)
	}
}
