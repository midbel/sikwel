package format_test

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/midbel/sweet/internal/lang/format"
)

func TestFormat(t *testing.T) {
	files, err := os.ReadDir("testdata")
	if err != nil {
		t.Errorf("not able to read testdata %s", err)
		return
	}
	for _, e := range files {
		t.Logf("formatting %s", e.Name())
		testFile(t, e.Name())
	}
}

func testFile(t *testing.T, file string) {
	t.Helper()
	input, want, err := getSQL(file)
	if err != nil {
		t.Errorf("error loading input SQL from %s: %s", file, err)
		return
	}
	var (
		ws strings.Builder
		wf = format.NewWriter(&ws)
	)
	if err := wf.Format(strings.NewReader(input)); err != nil {
		t.Errorf("error formatting input SQL: %s", err)
		return
	}
	got := strings.TrimSpace(ws.String())
	if got != want {
		t.Errorf("output SQL mismatched!")
		t.Logf("got : %s", strings.ReplaceAll(got, " ", "_"))
		t.Logf("want: %s", strings.ReplaceAll(want, " ", "_"))
	}
}

func getSQL(file string) (string, string, error) {
	r, err := os.Open(filepath.Join("testdata", file))
	if err != nil {
		return "", "", err
	}
	var (
		lines []string
		buf   strings.Builder
		scan  = bufio.NewScanner(io.TeeReader(r, &buf))
	)
	for scan.Scan() {
		str := scan.Text()
		lines = append(lines, strings.TrimSpace(str))
	}

	var (
		sql   = strings.ReplaceAll(buf.String(), "\t", "    ")
		parts = strings.Split(sql, "--")
	)
	if len(parts) > 1 {
		sql = parts[len(parts)-1]
	}
	sql = strings.TrimSpace(sql)
	return strings.Join(lines, " "), sql, scan.Err()
}
