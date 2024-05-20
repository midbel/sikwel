package lang

import (
	"fmt"
)

type stack[T prefixFunc | infixFunc] struct {
	values []*funcSet[T]
}

func emptyStack[T prefixFunc | infixFunc]() *stack[T] {
	return &stack[T]{}
}

func (s *stack[T]) Push(v *funcSet[T]) {
	s.values = append(s.values, v)
}

func (s *stack[T]) Pop() *funcSet[T] {
	if z := s.Len(); z > 0 {
		t := s.values[z-1]
		s.values = s.values[:z-1]
		return t
	}
	return nil
}

func (s *stack[T]) Len() int {
	return len(s.values)
}

func (s *stack[T]) Register(literal string, kind rune, fn T) {
	n := s.Len()
	if n == 0 {
		return
	}
	s.values[n-1].Register(literal, kind, fn)
}

func (s *stack[T]) Unregister(literal string, kind rune) {
	n := s.Len()
	if n == 0 {
		return
	}
	s.values[n-1].Unregister(literal, kind)
}

func (s *stack[T]) Get(sym symbol) (T, error) {
	var (
		n = s.Len()
		t T
	)
	if n == 0 {
		return t, fmt.Errorf("undefined function %+v", sym)
	}
	return s.values[n-1].Get(sym)
}

type prefixFunc func() (Statement, error)

type infixFunc func(Statement) (Statement, error)

type funcSet[T prefixFunc | infixFunc] struct {
	disabled bool
	funcs    map[symbol]T
}

func newFuncSet[T prefixFunc | infixFunc]() *funcSet[T] {
	return &funcSet[T]{
		funcs: make(map[symbol]T),
	}
}

func (s *funcSet[T]) Get(sym symbol) (T, error) {
	if s.disabled {
		return nil, fmt.Errorf("undefined function")
	}
	fn, ok := s.funcs[sym]
	if !ok {
		return nil, fmt.Errorf("undefined function")
	}
	return fn, nil
}

func (s *funcSet[T]) Toggle() {
	s.disabled = !s.disabled
}

func (s *funcSet[T]) Register(literal string, kind rune, fn T) {
	s.funcs[symbolFor(kind, literal)] = fn
}

func (s *funcSet[T]) Unregister(literal string, kind rune) {
	delete(s.funcs, symbolFor(kind, literal))
}
