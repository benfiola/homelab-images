COMPONENT_DIR := $(notdir $(CURDIR))
COMPONENT_NAME := $(basename $(COMPONENT_DIR))
IMAGE_BASE := ghcr.io/benfiola/homelab-images

.DEFAULT_GOAL := list-targets

.PHONY: list-targets
list-targets:
	@echo "available targets:"
	@LC_ALL=C $(MAKE) -pRrq -f $(firstword $(MAKEFILE_LIST)) : 2>/dev/null \
		| awk -v RS= -F: '/(^|\n)# Files(\n|$$$$)/,/(^|\n)# Finished Make data base/ {if ($$$$1 !~ "^[#.]") {print $$$$1}}' \
		| sort \
		| grep -E -v -e '^[^[:alnum:]]' -e '^$$@$$$$' \
		| sed -e 's/^/\t/' -e 's/:$$$$//'

.PHONY: next-version
next-version:
	@set -e; \
	FLAGS=""; \
	if [ -n "$(RC)" ]; then \
		FLAGS="$${FLAGS} --prerelease rc.$(RC)"; \
	fi; \
	if [ -n "$(ALPHA)" ]; then \
		FLAGS="$${FLAGS} --prerelease alpha.$(ALPHA)"; \
	fi; \
	if [ -n "$(METADATA)" ]; then \
		FLAGS="$${FLAGS} --metadata $(METADATA)"; \
	fi; \
	svu next --tag.prefix="$(COMPONENT_NAME)/v" --tag.pattern="$(COMPONENT_NAME)/v*" --tag.output="v" --always=true $${FLAGS} 2>/dev/null || echo "v1.0.0"

.PHONY: pre-publish
pre-publish:

.PHONY: snapshot
snapshot:
	IMAGE_BASE="$(IMAGE_BASE)" goreleaser release --clean --snapshot

.PHONY: publish
publish:
	IMAGE_BASE="$(IMAGE_BASE)" goreleaser release --clean
