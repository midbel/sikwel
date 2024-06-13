package ast

type ParameterMode int

const (
	ModeIn ParameterMode = 1 << (iota + 1)
	ModeOut
	ModeInOut
)

type ProcedureParameter struct {
	Mode    ParameterMode
	Name    string
	Type    Type
	Default Statement
}

type CreateProcedureStatement struct {
	Replace    bool
	Name       string
	Parameters []Statement
	Language   string
	Body       Statement
}

func (s CreateProcedureStatement) Keyword() (string, error) {
	if s.Replace {
		return "CREATE OR REPLACE PROCEDURE", nil
	}
	return "CREATE PROCEDURE", nil
}
