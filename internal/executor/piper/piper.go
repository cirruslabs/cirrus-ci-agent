package piper

import (
	"errors"
	"io"
	"os"
)

type Piper struct {
	r, w    *os.File
	errChan chan error
}

func New(output io.Writer) (*Piper, error) {
	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	piper := &Piper{
		r:       r,
		w:       w,
		errChan: make(chan error),
	}

	go func() {
		_, err := io.Copy(output, r)
		piper.errChan <- err
	}()

	return piper, nil
}

func (piper *Piper) Input() *os.File {
	return piper.w
}

func (piper *Piper) Close() (result error) {
	// Cancel the Goroutine started in New()
	result = piper.r.Close()

	if err := <-piper.errChan; err != nil && !errors.Is(err, os.ErrClosed) && result == nil {
		result = err
	}

	if err := piper.w.Close(); err != nil && !errors.Is(err, os.ErrClosed) && result == nil {
		result = err
	}

	return result
}
