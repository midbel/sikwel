package ast

type TransactionMode int

const (
	ModeReadWrite TransactionMode = 1 << (iota + 1)
	ModeReadOnly
)

type TransactionLevel int

const (
	LevelReadRepeat TransactionLevel = 1 << (iota + 1)
	LevelReadCommit
	LevelReadUncommit
	LevelSerializable
)

type SetTransaction struct {
	Mode  TransactionMode
	Level TransactionLevel
}

func (_ SetTransaction) Keyword() (string, error) {
	return "SET TRANSACTION", nil
}

type StartTransaction struct {
	Mode TransactionMode
	Body Statement
	End  Statement
}

func (_ StartTransaction) Keyword() (string, error) {
	return "START TRANSACTION", nil
}

type Savepoint struct {
	Name string
}

func (_ Savepoint) Keyword() (string, error) {
	return "SAVEPOINT", nil
}

type ReleaseSavepoint struct {
	Name string
}

func (_ ReleaseSavepoint) Keyword() (string, error) {
	return "RELEASE SAVEPOINT", nil
}

type RollbackSavepoint struct {
	Name string
}

func (_ RollbackSavepoint) Keyword() (string, error) {
	return "ROLLBACK TO SAVEPOINT", nil
}

type Commit struct{}

func (_ Commit) Keyword() (string, error) {
	return "COMMIT", nil
}

type Rollback struct{}

func (_ Rollback) Keyword() (string, error) {
	return "ROLLBACK", nil
}
