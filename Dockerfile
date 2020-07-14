FROM goreleaser/goreleaser:latest as builder

ENV GORELEASER_CURRENT_TAG=latest

WORKDIR /tmp/cirrus-ci-agent
ADD . /tmp/cirrus-ci-agent/

RUN goreleaser --snapshot

FROM gcr.io/distroless/base-debian10
COPY --from=builder /tmp/cirrus-ci-agent/ /bin