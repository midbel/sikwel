package rules

import (
	"slices"
)

type RuleFunc[T any] func(T) ([]LintMessage[T], error)

type RegisteredRule[T any] struct {
	Name     string
	Func     RuleFunc[T]
	Priority int
}

type Map[T any] map[string]RegisteredRule[T]

func (r Map[T]) Get() []RuleFunc[T] {
	var (
		tmp []RegisteredRule[T]
		all []RuleFunc[T]
	)
	for _, fn := range r {
		tmp = append(tmp, fn)
	}
	slices.SortFunc(tmp, func(a, b RegisteredRule[T]) int {
		return a.Priority - b.Priority
	})
	for i := range tmp {
		all = append(all, tmp[i].Func)
	}
	return all
}

func (r Map[T]) Register(name string, priority int, fn RuleFunc[T]) {
	r[name] = RegisteredRule[T]{
		Name:     name,
		Func:     fn,
		Priority: priority,
	}
}
