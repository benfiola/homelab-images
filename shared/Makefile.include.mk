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

.PHONY: version
version:
	@COMPONENT_NAME="$(COMPONENT_NAME)" python3 $(dir $(lastword $(MAKEFILE_LIST)))/../scripts/get-version.py

.PHONY: pre-publish
pre-publish:

.PHONY: snapshot
snapshot:
	IMAGE_BASE="$(IMAGE_BASE)" goreleaser release --clean --snapshot

.PHONY: publish
publish:
	IMAGE_BASE="$(IMAGE_BASE)" goreleaser release --clean
