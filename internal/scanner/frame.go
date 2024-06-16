package scanner

import (
	"io"
	"path/filepath"

	"github.com/midbel/sweet/internal/keywords"
	"github.com/midbel/sweet/internal/token"
)

type FrameFactory interface {
	Create(r io.Reader) (*Frame, error)
}

type keywordsFactory struct {
	keywords.Set
}

func FactoryFromKeywords(set keywords.Set) FrameFactory {
	return keywordsFactory{
		Set: set,
	}
}

func (k keywordsFactory) Create(r io.Reader) (*Frame, error) {
	return Create(r, k.Set)
}

type Frame struct {
	scan *Scanner
	set  keywords.Set

	file string
	curr token.Token
	peek token.Token
}

func Create(r io.Reader, set keywords.Set) (*Frame, error) {
	scan, err := Scan(r, set)
	if err != nil {
		return nil, err
	}
	f := Frame{
		scan: scan,
		set:  set,
	}
	if n, ok := r.(interface{ Name() string }); ok {
		f.file = n.Name()
	}
	f.Next()
	f.Next()
	return &f, nil
}

func (f *Frame) Keywords() keywords.Set {
	return f.set
}

func (f *Frame) File() string {
	return f.file
}

func (f *Frame) Base() string {
	return filepath.Dir(f.file)
}

func (f *Frame) Curr() token.Token {
	return f.curr
}

func (f *Frame) Peek() token.Token {
	return f.peek
}

func (f *Frame) GetCurrLiteral() string {
	return f.curr.Literal
}

func (f *Frame) GetPeekLiteral() string {
	return f.peek.Literal
}

func (f *Frame) GetCurrType() rune {
	return f.curr.Type
}

func (f *Frame) GetPeekType() rune {
	return f.peek.Type
}

func (f *Frame) Next() {
	f.curr = f.peek
	f.peek = f.scan.Scan()
}

func (f *Frame) Done() bool {
	return f.Is(token.EOF)
}

func (f *Frame) Is(kind rune) bool {
	return f.curr.Type == kind
}

func (f *Frame) PeekIs(kind rune) bool {
	return f.peek.Type == kind
}
