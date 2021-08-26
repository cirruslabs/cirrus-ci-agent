package resolver

import "github.com/cirruslabs/cirrus-ci-agent/internal/executor/metrics/source/cgroup/subsystem"

type Resolver interface {
	Resolve(subsystemName subsystem.SubsystemName) (string, string, error)
}
