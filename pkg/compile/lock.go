package compile

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/flock"
)

// compileMutex is used to protect two gococo processes from
// compiling one same project simultaneously
type compileMutex struct {
	flock   *flock.Flock
	timeout time.Duration
	cancel  context.CancelFunc
}

func newCompileMutex(path string, timeout time.Duration) *compileMutex {
	return &compileMutex{
		flock:   flock.New(path),
		timeout: timeout,
	}
}

func (l *compileMutex) Lock() (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), l.timeout)
	l.cancel = cancel

	locked, err := l.flock.TryLockContext(ctx, time.Second)
	if err != nil {
		return
	}

	if !locked {
		return fmt.Errorf("fail to lock the compile")
	}

	return
}

func (l *compileMutex) Unlock() (err error) {
	defer l.cancel()

	return l.flock.Unlock()
}
