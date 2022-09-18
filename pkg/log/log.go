package log

var g logger

func init() {
	g = &terminalLogger{}
}

func NewLogger(debug bool) {
	if debug == true {
		g = newDetailLogger()
	} else {
		g = &terminalLogger{}
	}
}

func Donef(format string, args ...interface{}) {
	g.Donef(format, args...)
}

func Infof(format string, args ...interface{}) {
	g.Infof(format, args...)
}

func Warnf(format string, args ...interface{}) {
	g.Warnf(format, args...)
}

func Fatalf(format string, args ...interface{}) {
	g.Fatalf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	g.Errorf(format, args...)
}

func StartWait(message string) {
	g.StartWait(message)
}

func StopWait() {
	g.StopWait()
}

func Sync() {
	g.Sync()
}

func SetExitCallback(callback func()) {
	g.SetExitCallback(callback)
}
