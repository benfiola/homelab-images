PLATFORMS ?= linux/amd64,linux/arm64
BUILD_DIR ?= .build
REPO_ROOT := $(shell cd $(dir $(lastword $(MAKEFILE_LIST)))/.. && pwd)

.DEFAULT_GOAL := list-targets

.PHONY: list-targets
list-targets:
	@echo "available targets:"
	@LC_ALL=C $(MAKE) -pRrq -f $(firstword $(MAKEFILE_LIST)) : 2>/dev/null \
		| awk -v RS= -F: '/(^|\n)# Files(\n|$$$$)/,/(^|\n)# Finished Make data base/ {if ($$$$1 !~ "^[#.]") {print $$$$1}}' \
		| sort \
		| grep -E -v -e '^[^[:alnum:]]' -e '^$$@$$$$' \
		| sed -e 's/^/\t/'

.PHONY: current-version
current-version:
	@go run $(REPO_ROOT)/scripts/main.go get-current-version

.PHONY: next-version
next-version:
	@RC="$(RC)" ALPHA="$(ALPHA)" METADATA="$(METADATA)" \
		go run $(REPO_ROOT)/scripts/main.go get-next-version

VERSION ?= $(shell RC="$(RC)" ALPHA="$(ALPHA)" METADATA="$(METADATA)" go run $(REPO_ROOT)/scripts/main.go get-next-version)

.PHONY: build-go
build-go:
	@VERSION="$(VERSION)" BUILD_DIR="$(BUILD_DIR)" PLATFORMS="$(PLATFORMS)" \
		go run $(REPO_ROOT)/scripts/main.go build-go

.PHONY: build-helm
build-helm:
	@VERSION="$(VERSION)" BUILD_DIR="$(BUILD_DIR)" \
		go run $(REPO_ROOT)/scripts/main.go build-helm

.PHONY: generate
generate:
	@go run $(REPO_ROOT)/scripts/main.go generate

.PHONY: pre-build
pre-build: generate

.PHONY: build
build: pre-build build-go build-helm

.PHONY: package-docker
package-docker:
	@VERSION="$(VERSION)" PLATFORMS="$(PLATFORMS)" \
		go run $(REPO_ROOT)/scripts/main.go package-docker

.PHONY: package-helm
package-helm:
	@VERSION="$(VERSION)" BUILD_DIR="$(BUILD_DIR)" \
		go run $(REPO_ROOT)/scripts/main.go package-helm

.PHONY: publish-docker
publish-docker:
	@VERSION="$(VERSION)" PLATFORMS="$(PLATFORMS)" \
		go run $(REPO_ROOT)/scripts/main.go publish-docker

.PHONY: publish-helm
publish-helm:
	@VERSION="$(VERSION)" \
		go run $(REPO_ROOT)/scripts/main.go publish-helm

.PHONY: publish
publish: publish-docker publish-helm

.PHONY: github-release
github-release:
	@VERSION="$(VERSION)" \
		go run $(REPO_ROOT)/scripts/main.go create-github-release

.PHONY: release
release: github-release
