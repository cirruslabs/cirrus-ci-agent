FROM golang:latest as builder

WORKDIR /tmp/cirrus-ci-agent
ADD . /tmp/cirrus-ci-agent/

# Install GoReleaser
RUN echo 'deb [trusted=yes] https://repo.goreleaser.com/apt/ /' | tee /etc/apt/sources.list.d/goreleaser.list
RUN apt update && apt -y install goreleaser

RUN goreleaser build --id=agent --single-target --snapshot --timeout 60m

FROM alpine:latest

LABEL org.opencontainers.image.source=https://github.com/cirruslabs/cirrus-ci-agent

RUN apk add --no-cache rsync
COPY --from=builder /tmp/cirrus-ci-agent/dist/agent_linux_*/agent /bin/cirrus-ci-agent
