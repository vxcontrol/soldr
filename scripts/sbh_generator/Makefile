SERVER_NAME ?= "example"
VXCA_FILE ?= "../../security/certs/$(SERVER_NAME)/vxca.key"
SBH_FILE ?= "../../security/vconf/lic/sbh.json"

.PHONY: build
build:
	go build

.PHONY: generate
generate: build
	./sbh_generator --key=$(VXCA_FILE) --expires=2025-01-01T12:00:00+00:00 --file=$(SBH_FILE) --version=$(SERVER_NAME) --force=true
