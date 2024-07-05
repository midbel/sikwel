package rules

type LintMessage struct {
	Severity Level
	Rule     string
	Message  string
}

type LintInfo struct {
	Rule    string
	Enabled bool
}
