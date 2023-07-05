package rest

import (
	"bytes"
	"io"
	"net/http"
	"strconv"

	"github.com/midbel/sweet/internal/format"
)

const MaxBodySize = (1 << 16) - 1

const SqlContent = "application/sql"

func Format(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if r.Header.Get("content-type") != SqlContent || r.Header.Get("accept") != SqlContent {
		w.WriteHeader(http.StatusExpectationFailed)
		return
	}
	var (
		ws bytes.Buffer
		rs = io.LimitReader(r.Body, MaxBodySize)
	)
	if err := format.WriteAnsi(rs, &ws); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, err.Error())
		return
	}
	w.Header().Set("content-type", "application/sql")
	w.Header().Set("content-length", strconv.Itoa(ws.Len()))
	io.Copy(w, &ws)
}
