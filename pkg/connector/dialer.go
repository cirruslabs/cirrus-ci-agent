package connector

import (
	"context"
	"net"
	"time"
)

func repeatedlyDial(ctx context.Context, network string, address string) (net.Conn, error) {
	for {
		dialer := &net.Dialer{Timeout: time.Second}
		conn, err := dialer.DialContext(ctx, network, address)
		if err == nil {
			// Connected!
			return conn, nil
		}

		// Wait before the next attempt
		select {
		case <-time.After(1 * time.Second):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}
