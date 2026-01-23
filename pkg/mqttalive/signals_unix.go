//go:build !windows

package mqttalive

import (
	"os"
	"syscall"
)

func getSignals() []os.Signal {
	return []os.Signal{os.Interrupt, syscall.SIGTERM}
}
