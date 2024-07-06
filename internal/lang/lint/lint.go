package lint

import (
	"errors"
	"fmt"
	"io"
	"slices"

	"github.com/midbel/sweet/internal/config"
	"github.com/midbel/sweet/internal/lang/ast"
	"github.com/midbel/sweet/internal/lang/parser"
	"github.com/midbel/sweet/internal/rules"
)

var ErrNa = errors.New("not applicable")

type Linter struct {
	MinLevel   rules.Level
	Max        int
	AbortOnErr bool
	rules      rules.Map[ast.Statement]
}

func NewLinter() *Linter {
	i := Linter{
		MinLevel: rules.Info,
		Max:      0,
		rules:    getDefaultRules(),
	}
	return &i
}

func (i *Linter) Rules() []rules.LintInfo {
	var infos []rules.LintInfo
	for _, n := range GetRuleNames() {
		g := rules.LintInfo{
			Rule: n,
		}
		_, g.Enabled = i.rules[n]
		infos = append(infos, g)
	}
	return infos
}

func (i *Linter) Lint(r io.Reader) ([]rules.LintMessage, error) {
	p, err := parser.NewParser(r)
	if err != nil {
		return nil, err
	}
	if ps, ok := p.(*parser.Parser); ok {
		i.configure(ps.Config)
	}
	var list []rules.LintMessage
	for {
		stmt, err := p.Parse()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		others, err := i.LintStatement(stmt)
		if err != nil {
			return nil, err
		}
		list = append(list, others...)
		if i.Max > 0 && len(list) >= i.Max {
			list = list[:i.Max]
			break
		}
	}
	return list, nil
}

func (i *Linter) LintStatement(stmt ast.Statement) ([]rules.LintMessage, error) {
	var list []rules.LintMessage
	for _, r := range i.rules.Get() {
		res, err := r(stmt)
		if err != nil {
			if errors.Is(err, ErrNa) {
				continue
			}
			return nil, err
		}
		list = append(list, res...)
	}
	return list, nil
}

func (i *Linter) configure(cfg *config.Config) error {
	for _, k := range cfg.Keys() {
		set, err := getRulesByName(k)
		if err != nil {
			return err
		}
		var (
			enabled  bool
			level    rules.Level
			priority = defaultPriority
		)

		v := cfg.Get(k)
		if b, ok := v.(bool); ok {
			enabled = b
		} else if x, ok := v.(*config.Config); ok {
			level = rules.GetLevelFromName(x.GetString("level"))
			if level < rules.Default {
				return fmt.Errorf("unknown level %q", x.GetString("level"))
			}
			priority = int(x.GetInt("priority"))
			enabled = true
		}
		for _, fn := range set {
			fn.Func = customizeRule(fn.Func, enabled, level)
			i.rules.Register(fn.Name, priority, fn.Func)
		}
	}
	return nil
}

func checkJoin(stmt ast.Statement) ([]rules.LintMessage, error) {
	switch stmt := stmt.(type) {
	case ast.SelectStatement:
		return handleSelectStatement(stmt, checkJoin)
	case ast.UnionStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkJoin)
	case ast.IntersectStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkJoin)
	case ast.ExceptStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkJoin)
	case ast.WithStatement:
		return handleWithStatement(stmt, checkJoin)
	case ast.CteStatement:
		return checkJoin(stmt.Statement)
	case ast.Join:
		return joinWithConstant(stmt)
	case ast.Group:
		return checkJoin(stmt.Statement)
	default:
		return nil, ErrNa
	}
}

func joinWithConstant(stmt ast.Join) ([]rules.LintMessage, error) {
	var check func(ast.Statement) bool

	check = func(stmt ast.Statement) bool {
		switch s := stmt.(type) {
		case ast.Binary:
			return check(s.Left) || check(s.Right)
		case ast.Value:
			return true
		case ast.In:
			return check(s.Ident)
		case ast.Is:
			return check(s.Ident)
		case ast.Between:
			return check(s.Ident)
		default:
			return false
		}
	}
	if check(stmt.Where) {
		return makeArray(constantJoin()), nil
	}
	return nil, nil
}

func checkSubqueriesNotAllow(stmt ast.Statement) ([]rules.LintMessage, error) {
	switch stmt := stmt.(type) {
	case ast.SelectStatement:
		return selectSubqueries(stmt)
	case ast.UnionStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkSubqueriesNotAllow)
	case ast.IntersectStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkSubqueriesNotAllow)
	case ast.ExceptStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkSubqueriesNotAllow)
	case ast.WithStatement:
		return handleWithStatement(stmt, checkSubqueriesNotAllow)
	case ast.CteStatement:
		return checkSubqueriesNotAllow(stmt.Statement)
	case ast.Join:
		return checkSubqueriesNotAllow(stmt.Table)
	case ast.Group:
		return checkSubqueriesNotAllow(stmt.Statement)
	default:
		return nil, ErrNa
	}
}

func selectSubqueries(stmt ast.SelectStatement) ([]rules.LintMessage, error) {
	isSubquery := func(q ast.Statement) bool {
		if a, ok := q.(ast.Alias); ok {
			q = a.Statement
		}
		if g, ok := q.(ast.Group); ok {
			q = g.Statement
		}
		_, ok := q.(ast.SelectStatement)
		return ok
	}
	var list []rules.LintMessage
	for _, c := range stmt.Columns {
		if isSubquery(c) {
			list = append(list, subqueryDisallow())
		}
	}
	for _, t := range stmt.Tables {
		j, ok := t.(ast.Join)
		if !ok {
			continue
		}
		if isSubquery(j.Table) {
			list = append(list, subqueryDisallow())
		}
	}
	others, err := handleSelectStatement(stmt, checkSubqueriesNotAllow)
	return slices.Concat(list, others), err
}

func subqueryDisallow() rules.LintMessage {
	return rules.LintMessage{
		Severity: rules.Error,
		Message:  "subquery is not allowed",
		Rule:     ruleSubqueryNotAllow,
	}
}

func constantJoin() rules.LintMessage {
	return rules.LintMessage{
		Severity: rules.Error,
		Message:  "join expression composed of constant values",
		Rule:     ruleConstExprJoin,
	}
}

func handleCompoundStatement(q1, q2 ast.Statement, check RuleFunc) ([]rules.LintMessage, error) {
	list1, err := check(q1)
	if err != nil && !errors.Is(err, ErrNa) {
		return nil, err
	}
	list2, err := check(q2)
	if err != nil && !errors.Is(err, ErrNa) {
		return nil, err
	}
	return append(list1, list2...), nil
}

func handleWithStatement(with ast.WithStatement, check RuleFunc) ([]rules.LintMessage, error) {
	var list []rules.LintMessage
	for _, q := range with.Queries {
		ms, err := check(q)
		if err != nil {
			if !errors.Is(err, ErrNa) {
				return nil, err
			}
			continue
		}
		list = append(list, ms...)
	}
	ms, err := check(with.Statement)
	if err == nil {
		list = append(list, ms...)
	}
	return list, err
}

func handleSelectStatement(stmt ast.SelectStatement, check RuleFunc) ([]rules.LintMessage, error) {
	var list []rules.LintMessage
	for _, c := range stmt.Columns {
		msg, err := check(c)
		if err != nil && !errors.Is(err, ErrNa) {
			return nil, err
		}
		list = append(list, msg...)
	}
	for _, c := range stmt.Tables {
		msg, err := check(c)
		if err != nil && !errors.Is(err, ErrNa) {
			return nil, err
		}
		list = append(list, msg...)
	}
	return list, nil
}

func makeArray[T rules.LintMessage](el T) []T {
	return []T{el}
}
