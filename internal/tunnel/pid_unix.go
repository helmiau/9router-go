//go:build !windows

package tunnel

import "os"

// processAlive reports whether a PID names a running process. Signal 0 is the
// portable POSIX probe: it performs the permission and existence checks
// without delivering anything.
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(nil) == nil
}
