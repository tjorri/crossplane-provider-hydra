# The build stage is pinned to BUILDPLATFORM (the runner's native arch) and
# cross-compiles the Go binary for TARGETARCH. Multi-platform builds under
# `docker buildx` would otherwise run the build stage under QEMU user-mode
# emulation once per target arch, which slows a `go build` by ~9×. Go's
# native cross-compile support via GOOS/GOARCH runs at full native speed.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine3.22 AS build
ARG TARGETOS
ARG TARGETARCH

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /provider ./cmd/provider/

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /provider /usr/local/bin/provider
ENTRYPOINT ["provider"]
