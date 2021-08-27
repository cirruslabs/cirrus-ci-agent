// +build windows,arm

package metrics

import (
	"context"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/sirupsen/logrus"
)

func Run(ctx context.Context, logger logrus.FieldLogger) (chan *api.ResourceUtilization, chan error) {
	resultChan := make(chan *api.ResourceUtilization, 1)
	errChan := make(chan error, 1)

	resultChan <- nil

	return resultChan, errChan
}
