package ast

type PrimaryKeyConstraint struct {
	Columns []string
}

func (_ PrimaryKeyConstraint) Keyword() (string, error) {
	return "PRIMARY KEY", nil
}

type ForeignKeyConstraint struct {
	Locals   []string
	Remotes  []string
	Table    string
	OnDelete Statement
	OnUpdate Statement
}

func (c ForeignKeyConstraint) Keyword() (string, error) {
	if len(c.Locals) == 0 {
		return "REFERENCES", nil
	}
	return "FOREIGN KEY", nil
}

type NotNullConstraint struct {
	Column string
}

func (_ NotNullConstraint) Keyword() (string, error) {
	return "NOT NULL", nil
}

type UniqueConstraint struct {
	Columns []string
}

func (_ UniqueConstraint) Keyword() (string, error) {
	return "UNIQUE", nil
}

type CheckConstraint struct {
	Expr Statement
}

func (_ CheckConstraint) Keyword() (string, error) {
	return "CHECK", nil
}

type DefaultConstraint struct {
	Expr Statement
}

func (_ DefaultConstraint) Keyword() (string, error) {
	return "DEFAULT", nil
}

type GeneratedConstraint struct {
	Expr Statement
}

func (_ GeneratedConstraint) Keyword() (string, error) {
	return "GENERATED ALWAYS AS", nil
}

type Constraint struct {
	Name string
	Statement
}
