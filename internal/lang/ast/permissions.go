package ast

type GrantStatement struct {
	Object     string
	Privileges []string
	Users      []string
}

func (s GrantStatement) Keyword() (string, error) {
	return "GRANT", nil
}

type RevokeStatement struct {
	Object     string
	Privileges []string
	Users      []string
}

func (s RevokeStatement) Keyword() (string, error) {
	return "REVOKE", nil
}
