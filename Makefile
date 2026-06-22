CONTROLLER_GEN_VERSION ?= 0.20.0
HELM_VERSION ?= 4.2.2
GH_VERSION ?= 2.95.0

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

.PHONY: install-helm
install-tools: install-helm
install-helm:
	@mkdir -p $(bin)
	@echo "Installing helm $(HELM_VERSION)..."
	@cd /tmp && \
	rm -rf helm-tmp && \
	mkdir helm-tmp && \
	cd helm-tmp && \
	curl -sL "https://get.helm.sh/helm-v$(HELM_VERSION)-$(shell echo $(OS) | tr A-Z a-z)-$(ARCH_NORMALIZED).tar.gz" | tar xz && \
	mv $(shell echo $(OS) | tr A-Z a-z)-$(ARCH_NORMALIZED)/helm $(bin)/helm && \
	cd /tmp && \
	rm -rf helm-tmp
	@echo "  ✓ helm installed"

.PHONY: install-gh
install-tools: install-gh
install-gh:
	@mkdir -p $(bin)
	@echo "Installing gh $(GH_VERSION)..."
	@cd /tmp && \
	rm -rf gh-tmp && \
	mkdir gh-tmp && \
	cd gh-tmp && \
	curl -sL "https://github.com/cli/cli/releases/download/v$(GH_VERSION)/gh_$(GH_VERSION)_$(shell echo $(OS) | tr A-Z a-z)_$(ARCH_NORMALIZED).tar.gz" | tar xz && \
	mv gh_$(GH_VERSION)_$(shell echo $(OS) | tr A-Z a-z)_$(ARCH_NORMALIZED)/bin/gh $(bin)/gh && \
	cd /tmp && \
	rm -rf gh-tmp
	@echo "  ✓ gh installed"

.PHONY: export-path
export-path:
	@echo "Add to PATH: export PATH=\$$PATH:$(bin)"
