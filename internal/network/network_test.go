package network_test

import (
	"github.com/cirruslabs/cirrus-ci-agent/internal/network"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
	"time"
)

const (
	maxAllowedWaitTime  = 60 * time.Second
	maxExpectedWaitTime = 10 * time.Second
)

// TestEarlyExit ensures that WaitForLocalPort() will exit before the full waitDuration
// if the port becomes available in the first loop iteration.
func TestEarlyExit(t *testing.T) {
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	defer lis.Close()

	port := lis.Addr().(*net.TCPAddr).Port

	start := time.Now()
	network.WaitForLocalPort(port, maxAllowedWaitTime)
	stop := time.Now()

	assert.WithinDuration(t, stop, start, maxExpectedWaitTime)
}
