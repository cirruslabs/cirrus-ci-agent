package connector

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
)

func MaleToMale(ctx context.Context, leftAddr string, rightAddr string) (doneChan chan struct{}, errChan chan error) {
	doneChan = make(chan struct{}, 1)
	errChan = make(chan error, 1)

	go func() {
		// A way to signal that connector has finished
		defer func() {
			doneChan <- struct{}{}
		}()

		// Create a cancellable sub-context and defer it's cancellation to ensure that:
		// * parent context will be able to cancel left and right handlers spawned below
		//   (they will error out once we close the connections)
		// * connections are cleaned up when this goroutine exits
		subCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		// Create connection pairs
		leftConn, err := repeatedlyDial(subCtx, "tcp", leftAddr)
		if err != nil {
			errChan <- err
			return
		}
		go func() {
			<-subCtx.Done()
			_ = leftConn.Close()
		}()

		rightConn, err := repeatedlyDial(subCtx, "tcp", rightAddr)
		if err != nil {
			errChan <- err
			return
		}
		go func() {
			<-subCtx.Done()
			_ = rightConn.Close()
		}()

		// Launch handlers that copy data in each direction
		var wg sync.WaitGroup
		wg.Add(2)

		// Left handler
		leftErrChan := make(chan error, 1)
		go func() {
			defer wg.Done()

			n, err := io.Copy(leftConn, rightConn)
			if err != nil {
				leftErrChan <- err
				return
			}

			fmt.Printf("[+] copied %d bytes to leftConn\n", n)
		}()

		// Right handler
		rightErrChan := make(chan error, 1)
		go func() {
			defer wg.Done()

			n, err := io.Copy(rightConn, leftConn)
			if err != nil {
				rightErrChan <- err
				return
			}

			fmt.Printf("[+] copied %d bytes to rightConn\n", n)

			// This is a way to interrupt io.Copy() in the goroutine above
			err = rightConn.(*net.TCPConn).CloseWrite()
			if err != nil {
				rightErrChan <- err
				return
			}
		}()

		wg.Wait()

		fmt.Println("[+] handlers finished")

		// Propagate handler errors (if any)
		select {
		case leftHandlerErr := <-leftErrChan:
			errChan <- leftHandlerErr
		case rightHandlerErr := <-rightErrChan:
			errChan <- rightHandlerErr
		default:
			// no errors
		}
	}()

	return doneChan, errChan
}
