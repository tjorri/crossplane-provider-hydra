# Project variables
PROJECT_NAME := crossplane-provider-hydra
PROJECT_REPO := github.com/tjorri/$(PROJECT_NAME)
CONTROLLER_IMAGE := ghcr.io/tjorri/$(PROJECT_NAME)-controller
PACKAGE_IMAGE := ghcr.io/tjorri/$(PROJECT_NAME)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

# Go variables
GO := go
GOLANGCI_LINT := golangci-lint

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
	rm -rf bin/ package/crds/ *.xpkg
