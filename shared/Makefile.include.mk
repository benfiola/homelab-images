COMPONENT_DIR ?= $(notdir $(CURDIR))
CONTROLLER_GEN_VERSION ?= 0.20.0

.DEFAULT_GOAL := help

.PHONY: help
help:
	@grep -E '^\w+:' $(MAKEFILE_LIST) | grep -v '^\.' | awk -F: '{print $$1}' | sort -u

.PHONY: generate
generate:

.PHONY: snapshot
snapshot:
	goreleaser build --snapshot

.PHONY: release
release:
	goreleaser release

.PHONY: fmt
fmt:
	gofmt -w ./...

.PHONY: mod-tidy
mod-tidy:
	go mod tidy

.PHONY: install-tools
install-tools: install-controller-gen

.PHONY: install-controller-gen
install-controller-gen:
	go install sigs.k8s.io/controller-tools/cmd/controller-gen@v$(CONTROLLER_GEN_VERSION)
