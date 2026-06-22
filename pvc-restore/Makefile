include ../shared/Makefile.include.mk

.PHONY: pre-publish
pre-publish:
	rm -rf chart/generated && mkdir -p chart/generated
	controller-gen object paths="./internal/api/..."
	controller-gen crd paths="./internal/..." output:stdout > chart/generated/crds.yaml
	controller-gen rbac:roleName=__roleName__ paths="./internal/..." output:stdout > chart/generated/rbac.yaml
