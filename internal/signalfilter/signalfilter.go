//go:build !windows
// +build !windows

package signalfilter

import (
	"os"
	"syscall"
)

func IsNoisy(sig os.Signal) bool {
	return sig == syscall.SIGURG || sig == syscall.SIGCHLD
}
