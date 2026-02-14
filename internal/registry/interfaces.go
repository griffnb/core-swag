package registry

// Debugger provides debug logging interface.
type Debugger interface {
	Printf(format string, v ...interface{})
}
