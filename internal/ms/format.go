package ms

import (
	"fmt"

	"github.com/midbel/sweet/internal/lang"
)

type tsqlFormatter struct{}

func (_ tsqlFormatter) Quote(str string) string {
	return fmt.Sprintf("[%s]", str)
}

func GetFormatter() lang.Formatter {
	return tsqlFormatter{}
}
