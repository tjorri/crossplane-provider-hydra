# Base images are pinned by the digest of their multi-platform index, so a
# build is reproducible and Dependabot opens a PR whenever a tag is rebuilt
# (e.g. for a Go patch release or a Debian security update). Pin the index
# digest rather than a per-platform one, or every target platform would get
# that one platform's image.
#
# Prefer tags the upstream project keeps updating: `golang:<ver>-alpine3.NN`
# stops receiving Go patch releases once a newer Alpine ships, and
# distroless only maintains the images for its current Debian release.

# The build stage is pinned to BUILDPLATFORM (the runner's native arch) and
# cross-compiles the Go binary for TARGETARCH. Multi-platform builds under
# `docker buildx` would otherwise run the build stage under QEMU user-mode
# emulation once per target arch, which slows a `go build` by ~9×. Go's
# native cross-compile support via GOOS/GOARCH runs at full native speed.
FROM --platform=$BUILDPLATFORM golang:1.27-alpine@sha256:d200d2dd76b03a99b52651fd1325aff7dcd35422c35c01bac56571b0507f3deb AS build
ARG TARGETOS
ARG TARGETARCH

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /provider ./cmd/provider/

FROM gcr.io/distroless/static-debian13:nonroot@sha256:18f4cc26c9ffa7f20a359298be7deadac5c607c8e03a02b8119f1a5fc2f82c99
COPY --from=build /provider /usr/local/bin/provider
ENTRYPOINT ["provider"]
