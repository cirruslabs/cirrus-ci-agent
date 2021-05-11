package main

import (
	"context"
	"github.com/cirruslabs/cirrus-ci-agent/internal/client"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_DialHTTPS(t *testing.T) {
	assert.Nil(t, checkEndpoint("https://grpc.cirrus-ci.com:443"))
}

func Test_DialNoSchema(t *testing.T) {
	assert.Nil(t, checkEndpoint("grpc.cirrus-ci.com:443"))
}

func checkEndpoint(endpoint string) error {
	clientConn, err := dialWithTimeout(context.Background(), endpoint)
	if err != nil {
		return err
	}

	defer clientConn.Close()

	client.InitClient(clientConn)

	return err
}
