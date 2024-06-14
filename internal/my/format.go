package my

import (
	"fmt"

	"github.com/midbel/sweet/internal/lang"
)

type mysqlFormatter struct{}

func (_ mysqlFormatter) Quote(str string) string {
	return fmt.Sprintf("`%s`", str)
}

func GetFormatter() lang.Formatter {
	return mysqlFormatter{}
}
