package sqlite

import (
	"github.com/midbel/sweet/internal/lang"
)

type Order struct {
	lang.Order
	Collate string
}

type InsertStatement struct {
	lang.Statement
	Action string
}
