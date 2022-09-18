package log

import "go.uber.org/zap"

type detailLogger struct {
	logger       *zap.Logger
	exitCallback func()
}

func (l *detailLogger) Debugf(format string, args ...interface{}) {
	l.logger.Sugar().Debugf(format, args...)
}

func (l *detailLogger) Donef(format string, args ...interface{}) {
	l.logger.Sugar().Infof(format, args...)
}

func (l *detailLogger) Infof(format string, args ...interface{}) {
	l.logger.Sugar().Infof(format, args...)
}

func (l *detailLogger) Warnf(format string, args ...interface{}) {
	l.logger.Sugar().Warnf(format, args...)
}

func (l *detailLogger) Fatalf(format string, args ...interface{}) {
	if l.exitCallback != nil {
		l.exitCallback()
	}

	l.logger.Sugar().Fatalf(format, args...)
}

func (l *detailLogger) Errorf(format string, args ...interface{}) {
	l.logger.Sugar().Errorf(format, args...)
}

func (l *detailLogger) StartWait(message string) {
}

func (l *detailLogger) StopWait() {
}

func (l *detailLogger) Sync() {
	l.logger.Sync()
}

func (l *detailLogger) SetExitCallback(callback func()) {
	l.exitCallback = callback
}

func newDetailLogger() *detailLogger {
	logger, _ := zap.NewDevelopment()
	logger = logger.WithOptions(zap.AddCallerSkip(2))

	return &detailLogger{
		logger: logger,
	}
}
