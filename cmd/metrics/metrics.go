package main

import (
	"context"
	"github.com/cirruslabs/cirrus-ci-agent/internal/executor/metrics"
	"github.com/sirupsen/logrus"
)

func main() {
	logger := logrus.New()

	_, errChan := metrics.Run(context.Background(), logger)

	if err := <-errChan; err != nil {
		logrus.Fatalf("metrics failed: %v", err)
	}
}
