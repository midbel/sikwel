package sikwel

import (
	"bufio"
	"io"
)

type Writer struct {
	inner  *bufio.Writer
	prefix int
	indent string
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{
		inner:  bufio.NewWriter(w),
		indent: " ",
	}
}
