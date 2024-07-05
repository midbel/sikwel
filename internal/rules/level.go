package rules

type Level int

const (
	Default Level = iota
	Debug
	Info
	Warning
	Error
	Unknown
)

func (e Level) String() string {
	switch e {
	case Debug:
		return "debug"
	case Info:
		return "info"
	case Warning:
		return "warning"
	case Error:
		return "error"
	default:
		return "unknown"
	}
}

func GetLevelFromName(name string) Level {
	switch name {
	case "", "default":
		return Default
	case "info":
		return Info
	case "warning":
		return Warning
	case "error":
		return Error
	default:
		return Unknown
	}
}
