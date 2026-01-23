//go:build windows

package mqttalive

import "os"

func getSignals() []os.Signal {
	return []os.Signal{os.Interrupt}
}
