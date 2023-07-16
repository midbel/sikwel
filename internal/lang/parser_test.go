package lang_test

import (
	"errors"
	"io"
	"os"
	"testing"

	"github.com/midbel/sweet/internal/lang"
)

func TestParser(t *testing.T) {
	t.Run("select", testSelect)
}

func testSelect(t *testing.T) {
	p, err := createParser("testdata/queries.sql")
	if err != nil {
		t.Errorf("fail to create parser: %s", err)
		return
	}
	for {
		_, err := p.Parse()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			t.Errorf("error parsing statement: %s", err)
			continue
		}
	}
}

func createParser(file string) (*lang.Parser, error) {
	r, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	return lang.NewParser(r)
}
