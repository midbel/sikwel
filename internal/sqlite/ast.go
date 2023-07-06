package sqlite

import (
	"github.com/midbel/sweet/internal/lang"
)

type Order struct {
	lang.Order
	Collate string
}
