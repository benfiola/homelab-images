CONTROLLER_GEN_VERSION ?= 0.20.0
GORELEASER_VERSION ?= 2.12.7
SVU_VERSION ?= 3.4.1

BIN ?= ./.bin
bin := $(abspath $(BIN))

OS := $(shell uname -s)
ARCH := $(shell uname -m)
ARCH_NORMALIZED := $(shell [ "$(ARCH)" = "aarch64" ] && echo "arm64" || echo "$(ARCH)")

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:' $(MAKEFILE_LIST) | grep -v '^\.' | awk -F: '{print $$1}' | sort -u

.PHONY: install-tools
install-tools:
	@echo "All tools installed to $(bin)"

.PHONY: install-goreleaser
install-tools: install-goreleaser
install-goreleaser:
	@mkdir -p $(bin)
	@echo "Installing goreleaser $(GORELEASER_VERSION)..."
	@curl -sL \
		"https://github.com/goreleaser/goreleaser/releases/download/v$(GORELEASER_VERSION)/goreleaser_$(OS)_$(ARCH_NORMALIZED).tar.gz" \
		| tar xz -C $(bin)
	@echo "  ✓ goreleaser installed"

.PHONY: install-svu
install-tools: install-svu
install-svu:
	@mkdir -p $(bin)
	@echo "Installing svu from fork (issue/297 branch)..."
	@cd /tmp && \
	rm -rf svu && \
	git clone --depth 1 --branch issue/297 https://github.com/benfiola/svu.git && \
	cd svu && \
	go build -o $(bin)/svu . && \
	cd /tmp && \
	rm -rf svu
	@echo "  ✓ svu installed from fork"

.PHONY: install-controller-gen
install-tools: install-controller-gen
install-controller-gen:
	@mkdir -p $(bin)
	@echo "Installing controller-gen $(CONTROLLER_GEN_VERSION)..."
	@GOBIN="$(bin)" go install sigs.k8s.io/controller-tools/cmd/controller-gen@v$(CONTROLLER_GEN_VERSION)
	@echo "  ✓ controller-gen installed"

.PHONY: install-python
install-tools: install-python
install-python:
	@echo "Installing python..."
	@apt -y update
	@apt -y install python3
	@echo "  ✓ python installed"

.PHONY: export-path
export-path:
	@echo "Add to PATH: export PATH=\$$PATH:$(bin)"
