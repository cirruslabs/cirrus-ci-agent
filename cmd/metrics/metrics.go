package main

import (
	"context"
	"github.com/cirruslabs/cirrus-ci-agent/internal/executor/metrics"
	"github.com/sirupsen/logrus"
)

func main() {
	logger := logrus.New()

	resultChan := metrics.Run(context.Background(), logger)

	result := <-resultChan

	if len(result.Errors()) != 0 {
		for _, err := range result.Errors() {
			logrus.Fatalf("metrics failed: %v", err)
		}
	}
}
