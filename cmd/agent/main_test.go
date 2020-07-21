package main

import (
	"context"
	"github.com/cirruslabs/cirrus-ci-agent/internal/client"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_DialHTTPS(t *testing.T) {
	assert.Nil(t, checkEndpoint(t, "https://grpc.cirrus-ci.com:443"))
}

func Test_DialNoSchema(t *testing.T) {
	assert.Nil(t, checkEndpoint(t, "grpc.cirrus-ci.com:443"))
}

func checkEndpoint(t *testing.T, endpoint string) error {
	clientConn, err := dialWithTimeout(endpoint)
	if err != nil {
		return err
	}

	defer clientConn.Close()

	client.InitClient(clientConn)
	_, err = client.CirrusClient.Ping(context.Background(), &empty.Empty{})
	return err
}
