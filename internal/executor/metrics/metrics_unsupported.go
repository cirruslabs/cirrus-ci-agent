//go:build (windows && arm) || (windows && arm64)
// +build windows,arm windows,arm64

package metrics

import (
	"context"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/sirupsen/logrus"
)

type Result struct {
	Errors              []error
	ResourceUtilization api.ResourceUtilization
}

func Run(ctx context.Context, logger logrus.FieldLogger) chan *Result {
	resultChan := make(chan *Result, 1)

	resultChan <- nil

	return resultChan
}
