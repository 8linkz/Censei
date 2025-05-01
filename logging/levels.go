package logging

// LogLevel defines the severity of log messages
type LogLevel int

const (
	// DEBUG level is used for detailed debugging messages
	DEBUG LogLevel = iota
	// INFO level is used for informational messages
	INFO
	// ERROR level is used for error messages
	ERROR
)

// LogLevelFromString converts a string to a LogLevel
func LogLevelFromString(level string) (LogLevel, bool) {
	switch level {
	case "DEBUG":
		return DEBUG, true
	case "INFO":
		return INFO, true
	case "ERROR":
		return ERROR, true
	default:
		return INFO, false
	}
}

// String returns the string representation of a LogLevel
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}
