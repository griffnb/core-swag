package console

import (
	"fmt"
	"os"
)

type logger struct {
	DebugLevel int
}

var Logger = &logger{
	DebugLevel: 0,
}

func (this *logger) Debug(format string, args ...any) {
	if this.DebugLevel >= 1 {
		printf(format+"\n", args...)
	}
}

// Info writes a status message to stdout unconditionally.
// Used for high-level phase markers ("Loaded N files", "Wrote swagger.json").
// Supports the same $Keyword{...} template syntax as Sprintf.
func (this *logger) Info(format string, args ...any) {
	fmt.Fprint(os.Stdout, Sprintf(format+"\n", args...))
}

// Error writes an error message to stderr unconditionally.
// Supports the same $Keyword{...} template syntax as Sprintf.
func (this *logger) Error(format string, args ...any) {
	fmt.Fprint(os.Stderr, Sprintf(format+"\n", args...))
}
