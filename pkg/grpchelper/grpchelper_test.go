package grpchelper_test

import (
	"github.com/cirruslabs/cirrus-ci-agent/pkg/grpchelper"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_SecurityDefault(t *testing.T) {
	target, insecure := grpchelper.TransportSettings("grpc.cirrus-ci.com:443")
	assert.Equal(t, "grpc.cirrus-ci.com:443", target)
	assert.False(t, insecure)
}

func Test_SecurityHTTP(t *testing.T) {
	target, insecure := grpchelper.TransportSettings("http://grpc.cirrus-ci.com:80")
	assert.Equal(t, "grpc.cirrus-ci.com:80", target)
	assert.True(t, insecure)
}

func Test_SecurityUNIX(t *testing.T) {
	target, insecure := grpchelper.TransportSettings("unix:///agent.sock")
	assert.Equal(t, "unix:///agent.sock", target)
	assert.True(t, insecure)
}
