# Project variables
PROJECT_NAME := crossplane-provider-hydra
PROJECT_REPO := github.com/tjorri/$(PROJECT_NAME)
CONTROLLER_IMAGE := ghcr.io/tjorri/$(PROJECT_NAME)-controller
PACKAGE_IMAGE := ghcr.io/tjorri/$(PROJECT_NAME)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

# Platforms the multi-arch xpkg is built for, in `docker buildx --platform`
# format, so the release workflow can share one PLATFORMS value between the
# controller image and the xpkgs.
PLATFORMS ?= linux/amd64,linux/arm64
# Where the multi-arch xpkg is pushed by xpkg-push-multiarch.
PACKAGE_REF ?= $(PACKAGE_IMAGE):$(VERSION)
OUTPUT_DIR := _output

# Go variables
GO := go
GOLANGCI_LINT := golangci-lint

empty :=
space := $(empty) $(empty)
comma := ,
PLATFORM_LIST := $(subst $(comma),$(space),$(PLATFORMS))

.PHONY: all
all: generate lint test build

.PHONY: generate
generate:
	controller-gen object paths=./apis/...
	controller-gen crd:crdVersions=v1 paths=./apis/... output:artifacts:config=package/crds
	angryjet generate-methodsets ./apis/...

.PHONY: lint
lint:
	$(GOLANGCI_LINT) run ./...

.PHONY: test
test:
	$(GO) test -short ./...

.PHONY: test-e2e
test-e2e:
	$(GO) test -v -count=1 -timeout=10m ./e2e/...

.PHONY: build
build:
	$(GO) build -o bin/provider ./cmd/provider/

.PHONY: docker-build
docker-build:
	docker build -t $(CONTROLLER_IMAGE):$(VERSION) .

.PHONY: docker-push
docker-push: docker-build
	docker push $(CONTROLLER_IMAGE):$(VERSION)

.PHONY: xpkg-build
# Builds the Crossplane package with the controller image embedded as a
# runtime layer via `--embed-runtime-image`. The resulting xpkg is
# self-contained: Crossplane's package manager extracts the embedded
# controller and runs it directly, so users only need to apply a Provider
# CR pointing at the xpkg — no DeploymentRuntimeConfig override.
#
# Requires the controller image to exist locally (docker-build target).
xpkg-build: generate docker-build
	crossplane xpkg build \
		--package-root=./package \
		--embed-runtime-image=$(CONTROLLER_IMAGE):$(VERSION) \
		--package-file=./$(PROJECT_NAME)-$(VERSION).xpkg

.PHONY: xpkg-build-multiarch
# Builds one xpkg per platform in PLATFORMS into $(OUTPUT_DIR), each
# embedding the controller built for that platform. xpkg-push-multiarch
# then publishes them under a single tag as an OCI image index, and the
# node's container runtime pulls the matching platform.
#
# The controller image is passed as a per-platform tarball instead of
# through the Docker daemon: `--embed-runtime-image` reads from the daemon
# with no platform selector, so what it embeds for a multi-platform tag
# depends on the daemon's image store.
xpkg-build-multiarch: generate
	@mkdir -p $(OUTPUT_DIR)
	@set -e; for platform in $(PLATFORM_LIST); do \
		suffix=$$(echo $$platform | tr / -); \
		echo "==> Building xpkg for $$platform"; \
		docker buildx build --platform $$platform \
			--output type=docker,dest=$(OUTPUT_DIR)/controller-$$suffix.tar .; \
		crossplane xpkg build \
			--package-root=./package \
			--embed-runtime-image-tarball=$(OUTPUT_DIR)/controller-$$suffix.tar \
			--package-file=$(OUTPUT_DIR)/$(PROJECT_NAME)-$(VERSION)-$$suffix.xpkg; \
	done

.PHONY: xpkg-push-multiarch
# Pushes the per-platform xpkgs from xpkg-build-multiarch to PACKAGE_REF as
# a single OCI image index. Crossplane's package manager always reads the
# package contents from the linux/amd64 entry, so PLATFORMS must include it.
xpkg-push-multiarch:
	$(if $(filter linux/amd64,$(PLATFORM_LIST)),,$(error PLATFORMS must include linux/amd64, got "$(PLATFORMS)"))
	crossplane xpkg push \
		-f $(subst $(space),$(comma),$(foreach p,$(PLATFORM_LIST),$(OUTPUT_DIR)/$(PROJECT_NAME)-$(VERSION)-$(subst /,-,$(p)).xpkg)) \
		$(PACKAGE_REF)

.PHONY: xpkg-verify-multiarch
# Fails unless PACKAGE_REF is an image index with exactly one entry per
# platform in PLATFORMS.
xpkg-verify-multiarch:
	crane manifest $(PACKAGE_REF) | jq -e --arg want "$(PLATFORM_LIST)" \
		'[.manifests[].platform | "\(.os)/\(.architecture)"] | sort == ($$want | split(" ") | sort)'

.PHONY: test-e2e-kind
test-e2e-kind:
	./hack/kind-e2e.sh

.PHONY: test-e2e-kind-teardown
test-e2e-kind-teardown:
	./hack/kind-e2e.sh teardown

.PHONY: tidy
tidy:
	$(GO) mod tidy

.PHONY: clean
clean:
	rm -rf bin/ package/crds/ *.xpkg $(OUTPUT_DIR)/
