package sqlite

import (
	"fmt"

	"github.com/midbel/sweet/internal/lang"
)

type Order struct {
	lang.Order
	Collate string
}

type VacuumStatement struct {
	Schema string
	File   string
}

func (s VacuumStatement) Keyword() (string, error) {
	return "VACUUM", nil
}

type InsertStatement struct {
	lang.Statement
	Action string
}

func (s InsertStatement) Keyword() (string, error) {
	var kw string
	switch s.Action {
	case "":
		kw = "INSERT INTO"
	case "ABORT":
		kw = "INSERT OR ABORT INTO"
	case "FAIL":
		kw = "INSERT OR FAIL INTO"
	case "IGNORE":
		kw = "INSERT OR IGNORE INTO"
	case "REPLACE":
		kw = "REPLACE INTO"
	case "ROLLBACK":
		kw = "INSERT OR ROLLBACK INTO"
	default:
		return "", fmt.Errorf("invalid action")
	}
	return kw, nil
}

type UpdateStatement struct {
	lang.Statement
	Action string
}

func (s UpdateStatement) Keyword() (string, error) {
	var kw string
	switch s.Action {
	case "":
		kw = "UPDATE"
	case "ABORT":
		kw = "UPDATE OR ABORT"
	case "FAIL":
		kw = "UPDATE OR FAIL"
	case "IGNORE":
		kw = "UPDATE OR IGNORE"
	case "REPLACE":
		kw = "UPDATE OR REPLACE"
	case "ROLLBACK":
		kw = "UPDATE OR ROLLBACK"
	default:
		return "", fmt.Errorf("invalid action")
	}
	return kw, nil
}
