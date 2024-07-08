package ast

func SplitWhereLiteral(where Statement) Statement {
	var (
		split       func(Statement) Statement
		hasConstant func(Statement) bool
	)

	hasConstant = func(stmt Statement) bool {
		b, ok := stmt.(Binary)
		if !ok {
			return true
		}
		if b.IsRelation() {
			return hasConstant(b)
		}
		_, ok1 := b.Left.(Value)
		_, ok2 := b.Right.(Value)
		return ok1 || ok2
	}

	split = func(stmt Statement) Statement {
		b, ok := stmt.(Binary)
		if !ok {
			return stmt
		}
		if b.IsRelation() {
			b.Left = split(b.Left)
			b.Right = split(b.Right)
			if b.Left != nil && b.Right != nil {
				return b
			}
			if b.Left == nil {
				return b.Right
			}
			if b.Right == nil {
				return b.Left
			}
			return nil
		}
		if hasConstant(b) {
			return b
		}
		return nil
	}
	return split(where)
}

func SplitWhere(where Statement) Statement {
	var (
		split   func(Statement) Statement
		discard func(Statement) bool
		isValue func(Statement) bool
	)

	isValue = func(stmt Statement) bool {
		_, ok := stmt.(Value)
		return ok
	}

	discard = func(stmt Statement) bool {
		b, ok := stmt.(Binary)
		if !ok {
			return true
		}
		return isValue(b.Left) || isValue(b.Right)
	}

	split = func(stmt Statement) Statement {
		b, ok := stmt.(Binary)
		if !ok && !b.IsRelation() {
			return b
		}
		if !discard(b.Left) && !discard(b.Right) {
			return b
		}
		if !discard(b.Left) {
			return split(b.Left)
		}
		return split(b.Right)
	}
	return split(where)
}
