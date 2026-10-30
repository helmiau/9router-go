//go:build !windows

package tunnel

import (
	"os/exec"
	"strconv"
	"syscall"
)

// configureDetached puts cloudflared in its own session, so it survives the
// dashboard exiting and is never signalled by a terminal the operator closed.
//
// Setsid is the unix equivalent of CREATE_NEW_PROCESS_GROUP: the child becomes
// a session leader with no controlling terminal, which is what detaches it
// from the shell that started the gateway.
func configureDetached(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}

// killCloudflaredByPort terminates every cloudflared whose command line
// targets port.
//
// The PID file only covers the process this binary started. A tunnel left
// running by the upstream Node binary, or by an earlier run whose PID file was
// lost, still holds the port and the origin — and the new tunnel cannot bind.
// The regex boundary keeps :20130 from matching :201300 or :202130.
func killCloudflaredByPort(port int) {
	if port <= 0 {
		return
	}
	pattern := "cloudflared.*:" + strconv.Itoa(port) + "([^0-9]|$)"
	// pkill exits non-zero when nothing matched, which is the normal case on
	// a machine with no tunnel running. That is not an error worth surfacing.
	_ = exec.Command("pkill", "-f", pattern).Run()
}
