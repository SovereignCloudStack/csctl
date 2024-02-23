# Copyright 2023 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

.DEFAULT_GOAL:=help

# variables
TAG ?= dev
ARCH ?= amd64
IMAGE_PREFIX ?= ghcr.io/sovereigncloudstack
BUILDER_IMAGE = $(IMAGE_PREFIX)/csctl-builder
BUILDER_IMAGE_VERSION = $(shell cat .builder-image-version.txt)
Version := $(shell git describe --tags --always --dirty)
Commit := $(shell git rev-parse HEAD)
LDFLAGS := -X github.com/SovereignCloudStack/csmctl/pkg/cmd.Version=$(Version) -X github.com/SovereignCloudStack/csmctl/pkg/cmd.Commit=$(Commit)

# Certain aspects of the build are done in containers for consistency (e.g. protobuf generation)
# If you have the correct tools installed and you want to speed up development you can run
# make BUILD_IN_CONTAINER=false target
# or you can override this with an environment variable
BUILD_IN_CONTAINER ?= true

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

# Directories
ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
BIN_DIR := bin
TOOLS_DIR := hack/tools
TOOLS_BIN_DIR := $(TOOLS_DIR)/$(BIN_DIR)
export PATH := $(abspath $(TOOLS_BIN_DIR)):$(PATH)
export GOBIN := $(abspath $(TOOLS_BIN_DIR))

##@ Clean   
#########
# Clean #
#########

.PHONY: clean
clean: ## cleans the csmctl binary
	@if [ -f csmctl ]; then rm csmctl; fi


##@ Common
##########
# Common #
##########
.PHONY: build
build: # build the csmctl binary
	go build -ldflags "$(LDFLAGS)" -o csmctl main.go

.PHONY: lint-golang
lint-golang: ## Lint Golang codebase
ifeq ($(BUILD_IN_CONTAINER),true)
	docker run  --rm -t -i \
		-v $(shell go env GOPATH)/pkg:/go/pkg$(MOUNT_FLAGS) \
		-v $(shell pwd):/src$(MOUNT_FLAGS) \
		$(BUILDER_IMAGE):$(BUILDER_IMAGE_VERSION) $@;
else
	go version
	golangci-lint version
	golangci-lint run -v
endif

.PHONY: lint-golang-ci
lint-golang-ci:
ifeq ($(BUILD_IN_CONTAINER),true)
	docker run  --rm -t -i \
		-v $(shell go env GOPATH)/pkg:/go/pkg$(MOUNT_FLAGS) \
		-v $(shell pwd):/src$(MOUNT_FLAGS) \
		$(BUILDER_IMAGE):$(BUILDER_IMAGE_VERSION) $@;
else
	go version
	golangci-lint version
	golangci-lint run -v --out-format=github-actions
endif

.PHONY: lint-yaml
lint-yaml: ## Lint YAML files
ifeq ($(BUILD_IN_CONTAINER),true)
	docker run  --rm -t -i \
		-v $(shell go env GOPATH)/pkg:/go/pkg$(MOUNT_FLAGS) \
		-v $(shell pwd):/src$(MOUNT_FLAGS) \
		$(BUILDER_IMAGE):$(BUILDER_IMAGE_VERSION) $@;
else
	yamllint --version
	yamllint -c .yamllint.yaml --strict .
endif

.PHONY: lint-yaml-ci
lint-yaml-ci:
ifeq ($(BUILD_IN_CONTAINER),true)
	docker run  --rm -t -i \
		-v $(shell go env GOPATH)/pkg:/go/pkg$(MOUNT_FLAGS) \
		-v $(shell pwd):/src$(MOUNT_FLAGS) \
		$(BUILDER_IMAGE):$(BUILDER_IMAGE_VERSION) $@;
else
	yamllint --version
	yamllint -c .yamllint.yaml . --format github
endif

DOCKERFILES=$(shell find . -not \( -path ./hack -prune \) -not \( -path ./vendor -prune \) -type f -regex ".*Dockerfile.*"  | tr '\n' ' ')
.PHONY: lint-dockerfile
lint-dockerfile: ## Lint Dockerfiles
ifeq ($(BUILD_IN_CONTAINER),true)
	docker run  --rm -t -i \
		-v $(shell go env GOPATH)/pkg:/go/pkg$(MOUNT_FLAGS) \
		-v $(shell pwd):/src$(MOUNT_FLAGS) \
		$(BUILDER_IMAGE):$(BUILDER_IMAGE_VERSION) $@;
else
	hadolint --version
	hadolint -t error $(DOCKERFILES)
endif

lint-links: ## Link Checker
ifeq ($(BUILD_IN_CONTAINER),true)
	docker run --rm -t -i \
		-v $(shell pwd):/src$(MOUNT_FLAGS) \
		$(BUILDER_IMAGE):$(BUILDER_IMAGE_VERSION) $@;
else
	lychee --version
	lychee --config .lychee.toml ./*.md  ./docs/**/*.md
endif

.PHONY: format-golang
format-golang: ## Format the Go codebase and run auto-fixers if supported by the linter.
ifeq ($(BUILD_IN_CONTAINER),true)
	docker run  --rm -t -i \
		-v $(shell go env GOPATH)/pkg:/go/pkg$(MOUNT_FLAGS) \
		-v $(shell pwd):/src$(MOUNT_FLAGS) \
		$(BUILDER_IMAGE):$(BUILDER_IMAGE_VERSION) $@;
else
	go version
	golangci-lint version
	golangci-lint run -v --fix
endif

.PHONY: format-yaml
format-yaml: ## Lint YAML files
ifeq ($(BUILD_IN_CONTAINER),true)
	docker run  --rm -t -i \
		-v $(shell go env GOPATH)/pkg:/go/pkg$(MOUNT_FLAGS) \
		-v $(shell pwd):/src$(MOUNT_FLAGS) \
		$(BUILDER_IMAGE):$(BUILDER_IMAGE_VERSION) $@;
else
	yamlfixer --version
	yamlfixer -c .yamllint.yaml .
endif

.PHONY: generate-boilerplate
generate-boilerplate: ## Generates missing boilerplates
	./hack/ensure-boilerplate.sh

# support go modules
generate-modules: ## Generates missing go modules
ifeq ($(BUILD_IN_CONTAINER),true)
	docker run  --rm -t -i \
		-v $(shell go env GOPATH)/pkg:/go/pkg$(MOUNT_FLAGS) \
		-v $(shell pwd):/src$(MOUNT_FLAGS) \
		$(BUILDER_IMAGE):$(BUILDER_IMAGE_VERSION) $@;
else
	./hack/golang-modules-update.sh
endif

generate-modules-ci: generate-modules
	@if ! (git diff --exit-code ); then \
		echo "\nChanges found in generated files"; \
		exit 1; \
	fi

.PHONY: verify-boilerplate
verify-boilerplate:
	./hack/verify-boilerplate.sh

.PHONY: verify-shellcheck
verify-shellcheck: ## Verify shell files
	./hack/verify-shellcheck.sh

.PHONY: generate
generate: generate-boilerplate generate-modules

ALL_VERIFY_CHECKS = boilerplate shellcheck
.PHONY: verify
verify: generate lint $(addprefix verify-,$(ALL_VERIFY_CHECKS)) ## Verify all

.PHONY: modules
modules: generate-modules ## Update go.mod & go.sum

.PHONY: boilerplate
boilerplate: generate-boilerplate ## Ensure that your files have a boilerplate header

# .PHONY: test
# test: test-unit ## Runs all unit and integration tests.

.PHONY: lint
lint: lint-golang lint-yaml lint-dockerfile lint-links ## Lint Codebase

.PHONY: format
format: format-golang format-yaml ## Format Codebase
