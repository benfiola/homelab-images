CONTROLLER_GEN_VERSION ?= 0.20.0
HELM_VERSION ?= 4.2.2
GH_VERSION ?= 2.95.0

BIN ?= ./.bin
bin := $(abspath $(BIN))

OS := $(shell uname -s)
OS_LOWER := $(shell echo $(OS) | tr A-Z a-z)
ARCH := $(shell uname -m)
ARCH_NORMALIZED := $(shell [ "$(ARCH)" = "aarch64" ] && echo "arm64" || echo "$(ARCH)")

.DEFAULT_GOAL := list-targets

.PHONY: list-targets
list-targets:
	@echo "available targets:"
	@LC_ALL=C $(MAKE) -pRrq -f $(firstword $(MAKEFILE_LIST)) : 2>/dev/null \
		| awk -v RS= -F: '/(^|\n)# Files(\n|$$$$)/,/(^|\n)# Finished Make data base/ {if ($$$$1 !~ "^[#.]") {print $$$$1}}' \
		| sort \
		| grep -E -v -e '^[^[:alnum:]]' -e '^$$@$$$$' \
		| sed -e 's/^/\t/' -e 's/:$$$$//'

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
	@curl -sL "https://get.helm.sh/helm-v$(HELM_VERSION)-$(OS_LOWER)-$(ARCH_NORMALIZED).tar.gz" | \
		tar xz --strip-components=1 -C $(bin) --wildcards "*/helm"
	@echo "  ✓ helm installed"

.PHONY: install-gh
install-tools: install-gh
install-gh:
	@mkdir -p $(bin)
	@echo "Installing gh $(GH_VERSION)..."
	@curl -sL "https://github.com/cli/cli/releases/download/v$(GH_VERSION)/gh_$(GH_VERSION)_$(OS_LOWER)_$(ARCH_NORMALIZED).tar.gz" | \
		tar xz --strip-components=2 -C $(bin) --wildcards "*/bin/gh"
	@echo "  ✓ gh installed"

.PHONY: export-path
export-path:
	@echo "Add to PATH: export PATH=\$$PATH:$(bin)"
