package network

import (
	"context"
	"fmt"
	"github.com/avast/retry-go"
	"math"
	"net"
	"time"
)

func WaitForLocalPort(ctx context.Context, port int) error {
	dialer := net.Dialer{
		Timeout: 10 * time.Second,
	}

	var conn net.Conn
	var err error

	return retry.Do(
		func() error {
			conn, err = dialer.DialContext(ctx, "tcp", fmt.Sprintf("localhost:%d", port))
			if err != nil {
				return err
			}

			_ = conn.Close()

			return nil
		},
		retry.Delay(1*time.Second), retry.MaxDelay(1*time.Second),
		retry.Attempts(math.MaxUint32), retry.LastErrorOnly(true),
		retry.Context(ctx),
	)
}
