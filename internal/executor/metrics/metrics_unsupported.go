//go:build openbsd || netbsd

package metrics

import (
	"context"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/sirupsen/logrus"
)

type Result struct {
	ResourceUtilization *api.ResourceUtilization
}

func (Result) Errors() []error {
	return []error{}
}

func Run(ctx context.Context, logger logrus.FieldLogger) chan *Result {
	resultChan := make(chan *Result, 1)

	resultChan <- nil

	return resultChan
}
