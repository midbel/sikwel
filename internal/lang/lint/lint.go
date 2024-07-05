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

func checkRewriteIn(stmt ast.Statement) ([]rules.LintMessage, error) {
	switch stmt := stmt.(type) {
	case ast.SelectStatement:
		return selectRewriteIn(stmt)
	case ast.UnionStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkRewriteIn)
	case ast.IntersectStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkRewriteIn)
	case ast.ExceptStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkRewriteIn)
	case ast.WithStatement:
		return handleWithStatement(stmt, checkRewriteIn)
	case ast.CteStatement:
		return checkRewriteIn(stmt.Statement)
	case ast.Join:
		return joinRewriteIn(stmt)
	case ast.Group:
		return checkRewriteIn(stmt.Statement)
	case ast.Binary:
		return lintIn(stmt)
	case ast.In:
		return lintIn(stmt)
	default:
		return nil, ErrNa
	}
}

func selectRewriteIn(stmt ast.SelectStatement) ([]rules.LintMessage, error) {
	list, err := lintIn(stmt.Where)
	if err != nil && !errors.Is(err, ErrNa) {
		return nil, err
	}
	others, err := handleSelectStatement(stmt, checkRewriteIn)
	return slices.Concat(list, others), err
}

func joinRewriteIn(stmt ast.Join) ([]rules.LintMessage, error) {
	list, err := checkRewriteIn(stmt.Where)
	if err != nil && !errors.Is(err, ErrNa) {
		return nil, err
	}
	others, err := checkRewriteIn(stmt.Table)
	return slices.Concat(list, others), err
}

func lintIn(stmt ast.Statement) ([]rules.LintMessage, error) {
	switch stmt := stmt.(type) {
	case ast.In:
		vs, ok := stmt.Value.(ast.List)
		if !ok {
			return nil, nil
		}
		if len(vs.Values) == 1 {
			return makeArray(rewriteIn()), nil
		}
		return nil, nil
	case ast.Binary:
		if !stmt.IsRelation() {
			return nil, ErrNa
		}
		l1, err1 := lintIn(stmt.Left)
		if err1 != nil && !errors.Is(err1, ErrNa) {
			return nil, err1
		}
		l2, err2 := lintIn(stmt.Right)
		if err2 != nil && !errors.Is(err2, ErrNa) {
			return nil, err2
		}
		return slices.Concat(l1, l2), nil
	default:
		return nil, ErrNa
	}
}

func checkConstantBinary(stmt ast.Statement) ([]rules.LintMessage, error) {
	switch stmt := stmt.(type) {
	case ast.SelectStatement:
		return selectConstantBinary(stmt)
	case ast.UnionStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkConstantBinary)
	case ast.IntersectStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkConstantBinary)
	case ast.ExceptStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkConstantBinary)
	case ast.WithStatement:
		return handleWithStatement(stmt, checkConstantBinary)
	case ast.CteStatement:
		return checkConstantBinary(stmt.Statement)
	case ast.Join:
		return joinConstantBinary(stmt)
	case ast.Group:
		return checkConstantBinary(stmt.Statement)
	default:
		return nil, ErrNa
	}
}

func selectConstantBinary(stmt ast.SelectStatement) ([]rules.LintMessage, error) {
	var list []rules.LintMessage
	if isConstant(stmt.Where) {
		list = append(list, constantOnlyExpr())
	}
	others, err := handleSelectStatement(stmt, checkConstantBinary)
	return slices.Concat(list, others), err
}

func joinConstantBinary(stmt ast.Join) ([]rules.LintMessage, error) {
	var list []rules.LintMessage
	if isConstant(stmt.Where) {
		list = append(list, constantOnlyExpr())
	}
	others, err := checkConstantBinary(stmt.Table)
	return slices.Concat(list, others), err
}

func isConstant(stmt ast.Statement) bool {
	switch s := stmt.(type) {
	case ast.Value:
		return true
	case ast.List:
		for _, v := range s.Values {
			if !isConstant(v) {
				return false
			}
		}
		return true
	case ast.Binary:
		if s.IsRelation() {
			return isConstant(s.Left) || isConstant(s.Right)
		}
		return isConstant(s.Left) && isConstant(s.Right)
	case ast.Is:
		return isConstant(s.Ident)
	case ast.In:
		return isConstant(s.Ident) && isConstant(s.Value)
	case ast.Between:
		return isConstant(s.Ident) && isConstant(s.Lower) && isConstant(s.Upper)
	default:
		return false
	}
}

func checkRewriteBinary(stmt ast.Statement) ([]rules.LintMessage, error) {
	switch stmt := stmt.(type) {
	case ast.SelectStatement:
		return selectRewriteBinary(stmt)
	case ast.UnionStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkRewriteBinary)
	case ast.IntersectStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkRewriteBinary)
	case ast.ExceptStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkRewriteBinary)
	case ast.WithStatement:
		return handleWithStatement(stmt, checkRewriteBinary)
	case ast.CteStatement:
		return checkRewriteBinary(stmt.Statement)
	case ast.Join:
		return joinRewriteBinary(stmt)
	case ast.Group:
		return checkRewriteBinary(stmt.Statement)
	default:
		return nil, ErrNa
	}
}

func selectRewriteBinary(stmt ast.SelectStatement) ([]rules.LintMessage, error) {
	list, err := lintBinary(stmt.Where)
	if err != nil && !errors.Is(err, ErrNa) {
		return nil, err
	}
	others, err := handleSelectStatement(stmt, checkRewriteBinary)
	return slices.Concat(list, others), err
}

func joinRewriteBinary(stmt ast.Join) ([]rules.LintMessage, error) {
	list, err := checkRewriteBinary(stmt.Where)
	if err != nil && !errors.Is(err, ErrNa) {
		return nil, err
	}
	others, err := checkRewriteBinary(stmt.Table)
	return slices.Concat(list, others), err
}

func lintBinary(stmt ast.Statement) ([]rules.LintMessage, error) {
	bin, ok := stmt.(ast.Binary)
	if !ok {
		return nil, ErrNa
	}
	if bin.IsRelation() {
		l1, err1 := lintBinary(bin.Left)
		if err1 != nil && !errors.Is(err1, ErrNa) {
			return nil, err1
		}
		l2, err2 := lintBinary(bin.Right)
		if err2 != nil && !errors.Is(err2, ErrNa) {
			return nil, err2
		}
		return slices.Concat(l1, l2), nil
	}
	if bin.Op == "=" || bin.Op == "<>" {
		if v, ok := bin.Right.(ast.Value); ok && v.Constant() {
			return makeArray(rewriteBinary()), nil
		}
		if v, ok := bin.Left.(ast.Value); ok && v.Constant() {
			return makeArray(rewriteBinary()), nil
		}
	}
	return nil, ErrNa
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

func rewriteIn() rules.LintMessage {
	return rules.LintMessage{
		Severity: rules.Warning,
		Message:  "in predicate should be rewritten",
		Rule:     ruleRewriteExprIn,
	}
}

func rewriteBinary() rules.LintMessage {
	return rules.LintMessage{
		Severity: rules.Warning,
		Message:  "expression should be rewritten",
		Rule:     ruleRewriteExpr,
	}
}

func constantJoin() rules.LintMessage {
	return rules.LintMessage{
		Severity: rules.Error,
		Message:  "join expression composed of constant values",
		Rule:     ruleConstExprJoin,
	}
}

func constantOnlyExpr() rules.LintMessage {
	return rules.LintMessage{
		Severity: rules.Error,
		Message:  "expression composed of constant values",
		Rule:     ruleConstExprBin,
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
