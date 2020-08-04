FROM goreleaser/goreleaser:latest as builder

ENV GORELEASER_CURRENT_TAG=latest

WORKDIR /tmp/cirrus-ci-agent
ADD . /tmp/cirrus-ci-agent/

RUN goreleaser build --snapshot

FROM alpine:latest

RUN apk add --no-cache rsync
COPY --from=builder /tmp/cirrus-ci-agent/dist/agent_linux_amd64/agent /bin/cirrus-ci-agent
