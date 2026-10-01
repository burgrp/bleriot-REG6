//go:build tinygo

package reg6

import (
	"runtime/interrupt"
	_ "unsafe"
)

//go:linkname runtimeNanotime runtime.nanotime
func runtimeNanotime() int64

func nanotime() int64 {
	state := interrupt.Disable()
	now := runtimeNanotime()
	interrupt.Restore(state)
	return now
}
