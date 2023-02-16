FROM goreleaser/goreleaser:latest as builder

WORKDIR /tmp/cirrus-ci-agent
ADD . /tmp/cirrus-ci-agent/

RUN goreleaser build --single-target --snapshot --timeout 60m

FROM alpine:latest

LABEL org.opencontainers.image.source=https://github.com/cirruslabs/cirrus-ci-agent

RUN apk add --no-cache rsync
COPY --from=builder /tmp/cirrus-ci-agent/dist/agent_linux_*/agent /bin/cirrus-ci-agent
