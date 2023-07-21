package lang_test

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/midbel/sweet/internal/lang"
)

func TestParserShouldFail(t *testing.T) {
	queries := []string{
		"select e.dept count(e.id) from employees e where e.salary >= 1000 and e.manager is null group by e.dept;",
	}
	for _, q := range queries {
		p, err := lang.NewParser(strings.NewReader(q))
		if err != nil {
			t.Errorf("fail to create parser for query: %s", q)
			continue
		}
		_, err = p.Parse()
		if err == nil {
			t.Errorf("error expected but query parse properly: %s", q)
		}
	}
}

func TestParser(t *testing.T) {
	files := []string{
		"select.sql",
		"delete.sql",
		"update.sql",
		"insert.sql",
		"transactions.sql",
		"script.sql",
		"call.sql",
	}
	for _, f := range files {
		testFile(t, f)
	}
}

func testFile(t *testing.T, file string) {
	t.Helper()

	r, err := os.Open(filepath.Join("testdata", file))
	if err != nil {
		t.Errorf("fail to open file %s (%s)", file, err)
		return
	}
	defer r.Close()

	p, err := lang.NewParser(r)
	if err != nil {
		t.Errorf("fail to create parser for file %s (%s)", file, err)
		return
	}
	for {
		_, err := p.Parse()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			t.Errorf("error parsing statement in %s: %s", file, err)
			continue
		}
	}
}
