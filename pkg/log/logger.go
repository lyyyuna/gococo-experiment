package log

type logger interface {
	Debugf(format string, args ...interface{})

	Infof(format string, args ...interface{})

	Warnf(format string, args ...interface{})

	Errorf(format string, args ...interface{})

	Fatalf(format string, args ...interface{})

	Donef(format string, args ...interface{})

	StartWait(message string)

	StopWait()

	Sync()

	SetExitCallback(callback func())
}
