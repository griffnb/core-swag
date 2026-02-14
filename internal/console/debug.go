package console

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
