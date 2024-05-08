**NOTE**: The agent now resides in the [Cirrus CLI's repository](https://github.com/cirruslabs/cirrus-cli) and can be invoked as `cirrus agent`.

# Agent to execute Cirrus CI tasks

[![Build Status](https://api.cirrus-ci.com/github/cirruslabs/cirrus-ci-agent.svg)](https://cirrus-ci.com/github/cirruslabs/cirrus-ci-agent)

This agent is used by [Cirrus CLI](https://github.com/cirruslabs/cirrus-cli) to run tasks locally and by [Cirrus CI](https://cirrus-ci.org/) to run the same tasks in a distributed fashion across larger variety of environments (containers, VMs on GCP/AWS/Azure, bare metal, etc.).

This tiny agent aims only to execute [Cirrus CI Instructions](https://cirrus-ci.org/guide/writing-tasks/#supported-instructions) and streams logs and execution progress to a service via gRPC API. Both [Cirrus CLI](https://github.com/cirruslabs/cirrus-cli) and [Cirrus CI](https://cirrus-ci.org/) implement the same gRPC API which makes it possible to seamlessly use the agent with either of them.
