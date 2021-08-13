// +build windows,arm

package metrics

import (
	"context"
	"github.com/cirruslabs/cirrus-ci-agent/api"
)

func Run(ctx context.Context) (chan *api.ResourceUtilization, chan error) {
	resultChan := make(chan *api.ResourceUtilization, 1)
	errChan := make(chan error, 1)

	resultChan <- nil

	return resultChan, errChan
}
