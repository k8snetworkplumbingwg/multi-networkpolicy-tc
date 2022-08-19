# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

# General Project parameters
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))

# Image related parameters, used when building image
IMAGE_REPOSITORY ?= nvidia.com
IMAGE_NAME ?= multi-networkpolicy-tc
IMAGE_TAG ?= latest
IMG ?= $(IMAGE_REPOSITORY)/$(IMAGE_NAME):$(IMAGE_TAG)

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

TARGET_OS ?= $(shell go env GOOS)
TARGET_ARCH ?= $(shell go env GOARCH)

# Options for go build command
GO_BUILD_OPTS ?= CGO_ENABLED=0 GOOS=$(TARGET_OS) GOARCH=$(TARGET_ARCH)
# Suffix for binary files
GO_BIN_SUFFIX ?= $(TARGET_OS)-$(TARGET_ARCH)

all: build

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

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)


##@ Development
lint: golangci-lint ## Lint code.
	$(GOLANGCILINT) run --timeout 10m

unit-test: ## Run unit tests.
	go test ./... -coverprofile cover.out

test: lint unit-test ## Run all tests (lint, unit-test).

##@ Build
.PHONY: build
build: ## Build multi-networkpolicy-tc binary.
	$(GO_BUILD_OPTS) go build -o build/multi-networkpolicy-tc-$(GO_BIN_SUFFIX) cmd/multi-networkpolicy-tc/main.go

build-in-docker: ## Build in docker container
	docker run -ti -v $(shell pwd):/code --user $(shell id -u $$(logname)):$(shell id -g $$(logname)) golang:1.18 bash -c \
	"export GOCACHE=/tmp && cd /code && make build"

run: ## Run a multi-networkpolicy-tc from your host.
	go run ./cmd/multi-networkpolicy-tc/main.go

docker-build: ## Build docker image.
	dockerfile=Dockerfile; \
	[ -f "$$dockerfile".$(TARGET_ARCH) ] && dockerfile="$$dockerfile".$(TARGET_ARCH); \
	docker build -t $(IMG) -f $$dockerfile .

docker-push: ## Push docker image.
	docker push ${IMG}


##@ Deployment
deploy: ## Deploy multi-networkpolicy-tc to the K8s cluster specified in ~/.kube/config.
	kubectl apply -f deploy/deploy.yaml

undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	kubectl delete -f deploy/deploy.yaml


##@ Dependency download
KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v4@v4.5.5)

GOLANGCILINT = $(shell pwd)/bin/golangci-lint
golangci-lint: ## Download golangci-lint locally if necessary.
	$(call go-install-tool,$(GOLANGCILINT),github.com/golangci/golangci-lint/cmd/golangci-lint@v1.46.2)

MOCKERY = $(shell pwd)/bin/mockery
mockery: ## Download mockery if necessary.
	$(call go-install-tool,$(MOCKERY),github.com/vektra/mockery/v2@v2.14.0)

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
