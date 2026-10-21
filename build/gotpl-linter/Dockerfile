FROM golang:1.22.2-alpine3.19 AS builder

ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH
ARG DBUILD_DATE

ENV CGO_ENABLED=0
ENV GOOS=$TARGETOS
ENV GOARCH=$TARGETARCH

RUN apk add --update bash git make

ADD . /project/
WORKDIR /project/

RUN mkdir -p ./bin \
    && env GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o ./bin/gotpl_linter_${TARGETOS}-${TARGETARCH} -ldflags "-s -w" ./cmd/gotpl-linter/main.go

FROM alpine:3.19

RUN apk add --update bash ca-certificates curl

ENV GOMEMLIMIT=256MiB

COPY --from=builder /project/bin/gotpl_linter_${TARGETOS}-${TARGETARCH}  /usr/bin/gotpl_linter

RUN chmod +x /usr/bin/gotpl_lint \
    && adduser -D service

USER service
WORKDIR /home/service

CMD ["gotpl_linter", "lint"]
ENTRYPOINT []