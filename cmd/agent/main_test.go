package main

import (
	"context"
	"github.com/cirruslabs/cirrus-ci-agent/internal/client"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_SecurityDefault(t *testing.T) {
	target, insecure := transportSettings("grpc.cirrus-ci.com:443")
	assert.Equal(t, "grpc.cirrus-ci.com:443", target)
	assert.False(t, insecure)
}

func Test_SecurityHTTP(t *testing.T) {
	target, insecure := transportSettings("http://grpc.cirrus-ci.com:80")
	assert.Equal(t, "grpc.cirrus-ci.com:80", target)
	assert.True(t, insecure)
}

func Test_SecurityUNIX(t *testing.T) {
	target, insecure := transportSettings("unix:///agent.sock")
	assert.Equal(t, "unix:///agent.sock", target)
	assert.True(t, insecure)
}

func Test_DialHTTPS(t *testing.T) {
	assert.Nil(t, checkEndpoint("https://grpc.cirrus-ci.com:443"))
}

func Test_DialNoSchema(t *testing.T) {
	assert.Nil(t, checkEndpoint("grpc.cirrus-ci.com:443"))
}

func checkEndpoint(endpoint string) error {
	clientConn, err := dialWithTimeout(endpoint)
	if err != nil {
		return err
	}

	defer clientConn.Close()

	client.InitClient(clientConn)
	_, err = client.CirrusClient.Ping(context.Background(), &empty.Empty{})
	return err
}
