package lang

import (
	"errors"
	"io"
	"slices"
)

func Complexity(r io.Reader) (int, error) {
	p, err := NewParser(r)
	if err != nil {
		return 0, err
	}
	var total int
	for {
		stmt, err := p.Parse()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return total, nil
		}
		total += measureQuery(stmt)
	}
	return total, nil
}

func measureQuery(stmt Statement) int {
	var total int
	switch stmt := stmt.(type) {
	case TruncateStatement:
		total++
	case DeleteStatement:
	case UpdateStatement:
	case InsertStatement:
	case MergeStatement:
	case ValuesStatement:
	case CallStatement:
	case UnionStatement:
	case ExceptStatement:
	case IntersectStatement:
	case SelectStatement:
		total = measureSelect(stmt)
	case WithStatement:
		total = measureWith(stmt)
	case CteStatement:
		total = measureCte(stmt)
	case SetStatement:
	case IfStatement:
	case WhileStatement:
	case Return:
	case Alias:
		total = measureQuery(stmt.Statement)
	case Join:
		total = measureJoin(stmt)
	case Case:
		total = measureCase(stmt)
	case When:
		total = measureWhen(stmt)
	case Group:
		total = measureQuery(stmt.Statement)
	case Binary:
		total = measureBinary(stmt)
	case Unary:
		total = measureQuery(stmt.Right)
	case Exists:
	case Between:
	case All:
	case Any:
	case Is:
	default:
		// pass
	}
	return total
}

func measureBinary(stmt Binary) int {
	var total int
	if stmt.IsRelation() {
		total++
	}
	total += measureQuery(stmt.Left)
	total += measureQuery(stmt.Right)
	return total

}

func measureCase(stmt Case) int {
	var total int
	for _, q := range stmt.Body {
		total++
		total += measureQuery(q)
	}
	if stmt.Else != nil {
		total++
		total += measureQuery(stmt.Else)
	}
	return total
}

func measureWhen(stmt When) int {
	return measureQuery(stmt.Cdt) + measureQuery(stmt.Body)
}

func measureJoin(stmt Join) int {
	var total int
	if _, ok := stmt.Table.(Name); ok {
		total++
	} else if a, ok := stmt.Table.(Alias); ok {
		stmt.Table = a.Statement
		return measureJoin(stmt)
	} else {
		total += measureQuery(stmt.Table)
	}
	return total + measureQuery(stmt.Where)
}

func measureSelect(stmt SelectStatement) int {
	var (
		total int
		list  []Statement
	)
	list = slices.Concat(list, stmt.Tables, stmt.Columns)
	for _, q := range list {
		total += measureQuery(q)
	}
	return total + 1
}

func measureWith(stmt WithStatement) int {
	return 0
}

func measureCte(stmt CteStatement) int {
	return 0
}
