# syntax=docker/dockerfile:1

# Multi-arch build: pass --platform (e.g. linux/amd64,linux/arm64) via buildx/podman.
# BuildKit injects BUILDPLATFORM/TARGETOS/TARGETARCH so the Go toolchain runs
# natively on the build host and cross-compiles for the target arch.
FROM --platform=$BUILDPLATFORM golang:1.22-alpine AS build

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown

WORKDIR /src

# Dependencies resolve in their own layer so source-only changes reuse it.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath \
    -ldflags "-s -w \
    -X 'main.BuildVersion=${VERSION}' \
    -X 'main.BuildHash=${COMMIT}' \
    -X 'main.BuildDate=${BUILD_DATE}'" \
    -o /out/gotpl-linter ./cmd/gotpl-linter

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/gotpl-linter /usr/local/bin/gotpl-linter

# The linter reads charts mounted into the container; /work is the conventional
# mount point, e.g. `podman run -v $PWD:/work:ro <image> lint -p templates`.
WORKDIR /work

USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/gotpl-linter"]
CMD ["lint"]
