package connector_test

import (
	"context"
	"github.com/cirruslabs/cirrus-ci-agent/pkg/connector"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net"
	"sync"
	"testing"
	"time"
)

func writeAndReadAll(conn net.Conn, contents []byte) (result []byte, err error) {
	_, err = conn.Write(contents)
	if err != nil {
		return nil, err
	}

	// We need this so that ioutil.ReadAll() on the other side won't block forever
	if err := conn.(*net.TCPConn).CloseWrite(); err != nil {
		return nil, err
	}

	return ioutil.ReadAll(conn)
}

func TestMaleToMale(t *testing.T) {
	left, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}

	right, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}

	doneChan, errChan := connector.MaleToMale(context.Background(), left.Addr().String(), right.Addr().String())

	leftConn, err := left.Accept()
	if err != nil {
		t.Fatal(err)
	}

	rightConn, err := right.Accept()
	if err != nil {
		t.Fatal(err)
	}

	var (
		sendToLeft = []byte("some bytes sent to the left socket")
		receivedFromLeft []byte

		sendToRight = []byte("some bytes sent to the right socket")
		receivedFromRight []byte
	)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		result, err := writeAndReadAll(leftConn, sendToLeft)
		if err != nil {
			t.Error(err)
			return
		}

		receivedFromLeft = result

		wg.Done()
	}()

	go func() {
		result, err := writeAndReadAll(rightConn, sendToRight)
		if err != nil {
			t.Error(err)
			return
		}

		receivedFromRight = result

		wg.Done()
	}()

	wg.Wait()

	assert.NotEmpty(t, doneChan)
	assert.Empty(t, errChan)
	assert.Equal(t, string(sendToLeft), string(receivedFromRight))
	assert.Equal(t, string(sendToRight), string(receivedFromLeft))
}

func TestMaleToMaleCancellation(t *testing.T) {
	// Create listeners that don't accept anything
	leftListener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	defer leftListener.Close()

	rightListener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	defer rightListener.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1 * time.Second)
	defer cancel()

	doneChan, _ := connector.MaleToMale(ctx, leftListener.Addr().String(), rightListener.Addr().String())

	<-doneChan
}
