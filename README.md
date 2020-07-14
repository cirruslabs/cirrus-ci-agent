# Agent to execute Cirrus CI tasks

[![Build Status](https://api.cirrus-ci.com/github/cirruslabs/cirrus-ci-agent.svg)](https://cirrus-ci.com/github/cirruslabs/cirrus-ci-agent)

This tiny agent aims only to execute [Cirrus CI Instructions](https://cirrus-ci.org/guide/writing-tasks/#supported-instructions) and streams logs and execution progress via GRPC API. 

This agent can work either with [Cirrus CLI](https://github.com/cirruslabs/cirrus-cli) to run tasks locally in Docker containers or with [Cirrus CI](https://cirrus-ci.org/) to run the same tasks in a distributed fasion across larger variety of environments (containers, VMs on GCP/AWS/Azure, bare metal, etc.).
