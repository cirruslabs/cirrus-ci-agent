package network

import (
	"fmt"
	"net"
	"time"
)

func WaitForLocalPort(port int, waitDuration time.Duration) {
	endTime := time.Now().Add(waitDuration)
	for time.Now().Before(endTime) {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 10*time.Second)
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		if conn != nil {
			_ = conn.Close()
		}
		break
	}
}
