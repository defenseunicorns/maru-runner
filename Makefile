# SPDX-License-Identifier: Apache-2.0
# SPDX-FileCopyrightText: 2023-Present the Maru Authors

ARCH ?= amd64
CLI_VERSION ?= $(if $(shell git describe --tags),$(shell git describe --tags),"UnknownVersion")
BUILD_ARGS := -s -w -X 'github.com/defenseunicorns/maru-runner/src/config.CLIVersion=$(CLI_VERSION)'
SRC_FILES ?= $(shell find . -type f -name "*.go")

BUILD_CLI_FOR_SYSTEM := build-cli
UNAME_S := $(shell uname -s)
UNAME_P := $(shell uname -p)
ifeq ($(UNAME_S),Darwin)
	ifeq ($(UNAME_P),i386)
		BUILD_CLI_FOR_SYSTEM := $(addsuffix -mac-intel,$(BUILD_CLI_FOR_SYSTEM))
	endif
	ifeq ($(UNAME_P),arm)
		BUILD_CLI_FOR_SYSTEM := $(addsuffix -mac-apple,$(BUILD_CLI_FOR_SYSTEM))
	endif
else ifeq ($(UNAME_S),Linux)
	ifeq ($(UNAME_P),x86_64)
		BUILD_CLI_FOR_SYSTEM := $(addsuffix -linux-amd,$(BUILD_CLI_FOR_SYSTEM))
	endif
	ifeq ($(UNAME_P),aarch64)
		BUILD_CLI_FOR_SYSTEM := $(addsuffix -linux-arm,$(BUILD_CLI_FOR_SYSTEM))
	endif
endif

.PHONY: help
help: ## Display this help information
	@grep -E '^[0-9a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
	  | sort | awk 'BEGIN {FS = ":.*?## "}; \
	  {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Build the CLI for the current machine's OS and architecture
	$(MAKE) $(BUILD_CLI_FOR_SYSTEM)

build-cli-linux-amd: ## Build the CLI for Linux AMD64
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$(BUILD_ARGS)" -o build/maru main.go

build-cli-linux-arm: ## Build the CLI for Linux ARM64
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="$(BUILD_ARGS)" -o build/maru-arm main.go

build-cli-mac-intel: ## Build the CLI for Mac Intel
	GOOS=darwin GOARCH=amd64 go build -ldflags="$(BUILD_ARGS)" -o build/maru-mac-intel main.go

build-cli-mac-apple: ## Build the CLI for Mac Apple
	GOOS=darwin GOARCH=arm64 go build -ldflags="$(BUILD_ARGS)" -o build/maru-mac-apple main.go

.PHONY: test-unit
test-unit: ## Run unit tests
	go test -failfast -v -timeout 30m $$(go list ./... | grep -v '^github.com/defenseunicorns/maru-runner/src/test/e2e')


.PHONY: test-e2e
test-e2e: ## Run End to End (e2e) tests
	cd src/test/e2e && go test -failfast -v -timeout 30m

schema: ## Update JSON schema for maru tasks
	./hack/generate-schema.sh

test-schema: schema ## Test if the schema has been modified
	./hack/test-generate-schema.sh

clean: ## Clean up build artifacts
	rm -rf build
