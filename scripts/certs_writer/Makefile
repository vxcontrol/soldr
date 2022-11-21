DST_DIR ?= ./certs

.PHONY: build
build:
	go build

.PHONY: generate
generate: build
	./certs_writer -cert_gen=true --dst=$(DST_DIR) --server_name=example
