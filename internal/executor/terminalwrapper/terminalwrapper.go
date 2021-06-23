package terminalwrapper

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/avast/retry-go"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-ci-agent/internal/client"
	"github.com/cirruslabs/terminal/pkg/host"
	"math"
	"time"
)

type Wrapper struct {
	ctx           context.Context
	operationChan chan Operation
	terminalHost  *host.TerminalHost
}

func New(ctx context.Context, taskIdentification *api.TaskIdentification) *Wrapper {
	wrapper := &Wrapper{
		ctx:           ctx,
		operationChan: make(chan Operation),
	}

	// A trusted secret that grants ability to spawn shells on the terminal host we start below
	trustedSecret, err := generateTrustedSecret()
	if err != nil {
		wrapper.operationChan <- &LogOperation{Message: fmt.Sprintf("Unable to generate a trusted secret needed to"+
			" initialize a terminal host: %v", err)}
		return wrapper
	}

	// A callback that will be called once the terminal host connects and registers on the terminal server
	locatorCallback := func(locator string) error {
		_, err := client.CirrusClient.ReportTerminalAttached(ctx, &api.ReportTerminalAttachedRequest{
			TaskIdentification: taskIdentification,
			Locator:            locator,
			TrustedSecret:      trustedSecret,
		})
		return err
	}

	wrapper.terminalHost, err = host.New(
		host.WithTrustedSecret(trustedSecret),
		host.WithLocatorCallback(locatorCallback),
	)
	if err != nil {
		wrapper.operationChan <- &LogOperation{Message: fmt.Sprintf("Failed to initialize a terminal host: %v", err)}
		return wrapper
	}

	go func() {
		_ = retry.Do(
			func() error {
				subCtx, cancel := context.WithCancel(ctx)
				defer cancel()

				return wrapper.terminalHost.Run(subCtx)
			},
			retry.OnRetry(func(n uint, err error) {
				wrapper.operationChan <- &LogOperation{Message: fmt.Sprintf("Terminal host failed: %v", err)}
			}),
			retry.Context(ctx),
			retry.Delay(1*time.Second), retry.MaxDelay(1*time.Second),
			retry.Attempts(math.MaxUint32), retry.LastErrorOnly(true),
		)
	}()

	return wrapper
}

func (wrapper *Wrapper) Wait() chan Operation {
	go func() {
		const minIdleDuration = 1 * time.Minute

		if wrapper.terminalHost == nil {
			wrapper.operationChan <- &ExitOperation{Success: false}

			return
		}

		message := fmt.Sprintf("Waiting for the terminal session to be inactive for at least %v...",
			minIdleDuration)
		wrapper.operationChan <- &LogOperation{Message: message}

		for {
			durationSinceLastActivity := time.Since(wrapper.terminalHost.LastActivity())

			if durationSinceLastActivity >= minIdleDuration {
				break
			}

			// Here the durationSinceLastActivity is less than minIdleDuration (see the check above),
			// so we account for the former to sleep the minimal reasonable duration possible
			timeToWait := minIdleDuration - durationSinceLastActivity

			select {
			case <-time.After(timeToWait):
				message := fmt.Sprintf("Waited %v, but there's still activity. Perhaps there are"+
					" terminal sessions open which generate the input/output?", timeToWait)
				wrapper.operationChan <- &LogOperation{Message: message}

				continue
			case <-wrapper.ctx.Done():
				wrapper.operationChan <- &ExitOperation{Success: true}
			}
		}
	}()

	return wrapper.operationChan
}

func generateTrustedSecret() (string, error) {
	buf := make([]byte, 32)

	_, err := rand.Read(buf)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(buf), nil
}
