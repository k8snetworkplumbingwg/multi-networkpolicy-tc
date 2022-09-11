# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL := /usr/bin/env bash -o pipefail
.SHELLFLAGS := -ec

# General Project parameters
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
BUILD_DIR := $(PROJECT_DIR)/build
BIN_DIR := $(PROJECT_DIR)/bin
COVERAGE_DIR := $(BUILD_DIR)/coverage
PKGS := $(shell cd $(PROJECT_DIR) && go list ./... | grep -v mocks)
TESTPKGS := $(shell go list -f '{{ if or .TestGoFiles .XTestGoFiles }}{{ .ImportPath }}{{ end }}' $(PKGS))

# Image related parameters, used when building image
IMAGE_REPOSITORY ?= nvidia.com
IMAGE_NAME ?= multi-networkpolicy-tc
IMAGE_TAG ?= latest
IMG ?= $(IMAGE_REPOSITORY)/$(IMAGE_NAME):$(IMAGE_TAG)

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN := $(shell go env GOPATH)/bin
else
GOBIN := $(shell go env GOBIN)
endif

TARGET_OS ?= $(shell go env GOOS)
TARGET_ARCH ?= $(shell go env GOARCH)

# Options for go build command
GO_BUILD_OPTS ?= CGO_ENABLED=0 GOOS=$(TARGET_OS) GOARCH=$(TARGET_ARCH)
# Suffix for binary files
GO_BIN_SUFFIX ?= $(TARGET_OS)-$(TARGET_ARCH)

# Binaries
KUSTOMIZE := $(BIN_DIR)/kustomize
GOLANGCILINT := $(BIN_DIR)/golangci-lint
MOCKERY := $(BIN_DIR)/mockery
ENVTEST := $(BIN_DIR)/setup-envtest
GOCOVMERGE := $(BIN_DIR)/gocovmerge
GCOV2LCOV := $(BIN_DIR)/gcov2lcov

ENVTEST_K8S_VERSION := 1.24

.PHONY: all
all: build test

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)


##@ Development
.PHONY: lint
lint: golangci-lint ## Lint code.
	$(GOLANGCILINT) run

.PHONY: unit-test
unit-test: envtest ## Run unit tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ./...

.PHONY: test
test: lint unit-test ## Run all tests (lint, unit-test).

.PHONY: test-coverage
test-coverage: | envtest gocovmerge gcov2lcov ## Run coverage tests
	mkdir -p $(PROJECT_DIR)/build/coverage/pkgs
	for pkg in $(TESTPKGS); do \
		KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test \
		-covermode=atomic \
		-coverprofile="$(COVERAGE_DIR)/pkgs/`echo $$pkg | tr "/" "-"`.cover" $$pkg ;\
	done
	$(GOCOVMERGE) $(COVERAGE_DIR)/pkgs/*.cover > $(COVERAGE_DIR)/profile.out
	$(GCOV2LCOV) -infile $(COVERAGE_DIR)/profile.out -outfile $(COVERAGE_DIR)/lcov.info

##@ Build
.PHONY: build
build: ## Build multi-networkpolicy-tc binary.
	$(GO_BUILD_OPTS) go build -o build/multi-networkpolicy-tc-$(GO_BIN_SUFFIX) cmd/multi-networkpolicy-tc/main.go

.PHONY: build-in-docker
build-in-docker: ## Build in docker container
	docker run -ti -v $(shell pwd):/code --user $(shell id -u $$(logname)):$(shell id -g $$(logname)) golang:1.18 bash -c \
	"export GOCACHE=/tmp && cd /code && make build"

.PHONY: run
run: ## Run a multi-networkpolicy-tc from your host.
	go run ./cmd/multi-networkpolicy-tc/main.go

.PHONY: docker-build
docker-build: ## Build docker image.
	dockerfile=Dockerfile; \
	[ -f "$$dockerfile".$(TARGET_ARCH) ] && dockerfile="$$dockerfile".$(TARGET_ARCH); \
	docker build -t $(IMG) -f $$dockerfile .

.PHONY: docker-push
docker-push: ## Push docker image.
	docker push ${IMG}


##@ Deployment
.PHONY: deploy
deploy: ## Deploy multi-networkpolicy-tc to the K8s cluster specified in ~/.kube/config.
	kubectl apply -f deploy/deploy.yaml

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	kubectl delete -f deploy/deploy.yaml


##@ Dependency download
.PHONY: kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v4@v4.5.5)

.PHONY: golangci-lint
golangci-lint: ## Download golangci-lint locally if necessary.
	$(call go-install-tool,$(GOLANGCILINT),github.com/golangci/golangci-lint/cmd/golangci-lint@v1.49.0)

.PHONY: mockery
mockery: ## Download mockery if necessary.
	$(call go-install-tool,$(MOCKERY),github.com/vektra/mockery/v2@v2.14.0)

.PHONY: envtest
envtest: ## Download envtest if necessary
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest@latest)

.PHONY: gocovmerge
gocovmerge: ## Download gocovmerge if necessary
	$(call go-install-tool,$(GOCOVMERGE),github.com/wadey/gocovmerge@latest)

.PHONY: gcov2lcov
gcov2lcov: ## Download gcov2lcov if necessary
	$(call go-install-tool,$(GCOV2LCOV),github.com/jandelgado/gcov2lcov@v1.0.5)

.PHONY: clean
clean:
	@rm -rf build
	@rm -rf bin


# go-get-tool will 'go get' any package $2 and install it to $1.
define go-install-tool
@[ -f $(1) ] || { \
set -e ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go install $(2) ;\
}
endef
