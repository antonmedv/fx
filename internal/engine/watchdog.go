package engine

import (
	"fmt"
	"sync"
	"time"

	"github.com/dop251/goja"
)

// previewTimeout bounds a live preview, which runs while the user types and
// may be a half-typed expression that never ends or allocates without end.
// A query the user runs has no limit: Esc cancels it. A variable for tests.
var previewTimeout = 2 * time.Second

// watchdog stops a run on cancel or after its timeout: it interrupts the
// JS and closes abort, which the Go side of the run checks.
type watchdog struct {
	vm     *goja.Runtime
	abort  chan struct{}
	once   sync.Once
	reason string // Set before abort is closed, empty on cancel.
}

// startWatchdog watches the run until finished is closed. A zero timeout is
// no timeout.
func startWatchdog(vm *goja.Runtime, cancel <-chan struct{}, finished <-chan struct{}, timeout time.Duration) *watchdog {
	w := &watchdog{vm: vm, abort: make(chan struct{})}
	go func() {
		var expired <-chan time.Time
		if timeout > 0 {
			timer := time.NewTimer(timeout)
			defer timer.Stop()
			expired = timer.C
		}
		select {
		case <-finished:
		case <-cancel:
			w.stop("")
		case <-expired:
			w.stop(fmt.Sprintf("Query stopped: it took longer than %v", timeout))
		}
	}()
	return w
}

func (w *watchdog) stop(reason string) {
	w.once.Do(func() {
		w.reason = reason
		close(w.abort)
		w.vm.Interrupt("cancelled")
	})
}

// limitError returns the error of a run stopped by its timeout, or nil.
func (w *watchdog) limitError() error {
	select {
	case <-w.abort:
		if w.reason != "" {
			return &Error{w.reason}
		}
	default:
	}
	return nil
}

func (w *watchdog) stopped() bool {
	select {
	case <-w.abort:
		return true
	default:
		return false
	}
}
