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
xpkg-build: generate
	@sed 's|image:.*|image: $(CONTROLLER_IMAGE):$(VERSION)|' package/crossplane.yaml > package/crossplane.yaml.tmp \
		&& mv package/crossplane.yaml.tmp package/crossplane.yaml
	crossplane xpkg build --package-root=./package --package-file=./$(PROJECT_NAME)-$(VERSION).xpkg

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
