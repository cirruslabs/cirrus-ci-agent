// +build !linux

package resolver

import (
	"github.com/cirruslabs/cirrus-ci-agent/internal/executor/metrics/source/cgroup"
)

func New() (Resolver, error) {
	return nil, cgroup.ErrUnsupportedPlatform
}
