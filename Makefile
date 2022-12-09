LOCAL_BIN := $(CURDIR)/build/bin


ifneq (,$(wildcard ./.env))
    include .env
    export
endif


RUN_ARGS :=


.PHONY: build-all
build-all: build-backend build-web

include bin-deps.mk

.PHONY: build-backend
build-backend: build-api build-server build-agent

.PHONY: build-backend-vxbuild
build-backend-vxbuild: build-api-vxbuild build-server-vxbuild build-agent-vxbuild

.PHONY: build-api
build-api:
	$(CURDIR)/build/package/api/build-local.sh

.PHONY: build-api-vxbuild
build-api-vxbuild:
	$(CURDIR)/build/package/api/build-vxbuild.sh

.PHONY: build-server
build-server:
	$(CURDIR)/build/package/server/build-local.sh

.PHONY: build-server-vxbuild
build-server-vxbuild:
	$(CURDIR)/build/package/server/build-vxbuild.sh

.PHONY: build-agent
build-agent:
	$(CURDIR)/build/package/agent/build-local.sh

.PHONY: build-agent-vxbuild
build-agent-vxbuild:
	$(CURDIR)/build/package/agent/build-vxbuild.sh

.PHONY: build-web
build-web:
	cd $(CURDIR)/web && npm install --legacy-peer-deps && npm run build

.PHONY: run-api
run-api: build-api
	cd $(CURDIR)/build && \
		LOG_DIR=$(CURDIR)/build/logs \
		CERTS_PATH=$(CURDIR)/security/certs/api \
		MIGRATION_DIR=$(CURDIR)/db/api/migrations \
		TEMPLATES_DIR=$(CURDIR)/build/package/api/templates \
		$(CURDIR)/build/bin/vxapi $(RUN_ARGS)

.PHONY: run-server
run-server: build-server
	cd $(CURDIR)/build && \
		LOG_DIR=$(CURDIR)/build/logs \
		MIGRATION_DIR=$(CURDIR)/db/server/migrations \
		CERTS_PATH=$(CURDIR)/security/certs/server \
		VALID_PATH=$(CURDIR)/security/vconf \
		$(CURDIR)/build/bin/vxserver $(RUN_ARGS)

.PHONY: run-agent
run-agent: build-agent
	cd $(CURDIR)/build && \
		LOG_DIR=$(CURDIR)/build/logs \
		$(CURDIR)/build/bin/vxagent -connect $(CONNECT) $(RUN_ARGS)

.PHONY: run-web
run-web:
	cd $(CURDIR)/web && npm run start

.PHONY: fmt
fmt: $(GOLANGCI_BIN)
	$(GOLANGCI_BIN) run --fix ./...

.PHONY: lint
lint: $(GOLANGCI_BIN)
	$(GOLANGCI_BIN) run ./...

.PHONY: test
test:
	go test -v ./...

.PHONY: generate-all
generate-all: generate-certs generate-keys generate-ssl generate

.PHONY: generate
generate: $(GOSWAGGER_BIN)
	go generate -v ./...
	$(GOSWAGGER_BIN) init -d cmd/api/ -g ../../internal/app/api/server/router.go -o internal/app/api/docs/ --parseDependency --parseInternal --parseDepth 2
	make -C $(CURDIR)/scripts/errgen generate

.PHONY: generate-ssl
generate-ssl:
	cd $(CURDIR)/build && \
		API_USE_SSL=true \
		$(CURDIR)/build/package/api/entrypoint.sh echo "ssl keys generated done"

.PHONY: generate-certs
generate-certs:
	$(CURDIR)/scripts/gen-certs.sh

.PHONY: generate-keys
generate-keys:
	go run scripts/encryption/keygen.go

.PHONY: setup-web-proxy
setup-web-proxy:
	echo '{"api": {"target": "http://localhost", "secure": false}}' > $(CURDIR)/web/proxy.conf.json

.PHONY: db-init
db-init: db-create db-seed

.PHONY: db-create
db-create:
	mysql --host=$(DB_HOST) --user=$(DB_ROOT_USER) --password=$(DB_ROOT_PASS) --port=$(DB_PORT) --execute="CREATE DATABASE IF NOT EXISTS $(DB_NAME);"
	mysql --host=$(DB_HOST) --user=$(DB_ROOT_USER) --password=$(DB_ROOT_PASS) --port=$(DB_PORT) --execute="ALTER DATABASE $(DB_NAME) DEFAULT CHARACTER SET utf8 DEFAULT COLLATE utf8_unicode_ci;"
	mysql --host=$(DB_HOST) --user=$(DB_ROOT_USER) --password=$(DB_ROOT_PASS) --port=$(DB_PORT) --execute="CREATE DATABASE IF NOT EXISTS $(AGENT_SERVER_DB_NAME);"
	mysql --host=$(DB_HOST) --user=$(DB_ROOT_USER) --password=$(DB_ROOT_PASS) --port=$(DB_PORT) --execute="ALTER DATABASE $(AGENT_SERVER_DB_NAME) DEFAULT CHARACTER SET utf8 DEFAULT COLLATE utf8_unicode_ci;"
	mysql --host=$(DB_HOST) --user=$(DB_ROOT_USER) --password=$(DB_ROOT_PASS) --port=$(DB_PORT) --execute="CREATE USER IF NOT EXISTS '$(AGENT_SERVER_DB_USER)' IDENTIFIED BY '$(AGENT_SERVER_DB_PASS)';"
	mysql --host=$(DB_HOST) --user=$(DB_ROOT_USER) --password=$(DB_ROOT_PASS) --port=$(DB_PORT) --execute="GRANT ALL PRIVILEGES ON $(AGENT_SERVER_DB_NAME).* TO $(AGENT_SERVER_DB_USER)@'%';"

.PHONY: db-seed
db-seed:
	grep -ve "^DROP TABLE" -ve "^--" $(CURDIR)/db/api/migrations/0001_initial.sql | mysql --host=$(DB_HOST) --user=$(DB_USER) --password=$(DB_PASS) --port=$(DB_PORT) --batch $(DB_NAME)
	mysql --host=$(DB_HOST) --user=$(DB_USER) --password=$(DB_PASS) --port=$(DB_PORT) $(DB_NAME) < $(CURDIR)/db/api/seed.sql
	grep -ve "^DROP TABLE" -ve "^--" $(CURDIR)/db/server/migrations/0001_initial.sql | mysql --host=$(DB_HOST) --user=$(AGENT_SERVER_DB_USER) --password=$(AGENT_SERVER_DB_PASS) --port=$(DB_PORT) $(AGENT_SERVER_DB_NAME)

.PHONY: s3-init
s3-init: $(GOMINIO_BIN)
	$(GOMINIO_BIN) config host add vxm "$(MINIO_ENDPOINT)" $(MINIO_ACCESS_KEY) $(MINIO_SECRET_KEY) 2>/dev/null
	$(GOMINIO_BIN) mb --ignore-existing vxm/$(MINIO_BUCKET_NAME)
	$(GOMINIO_BIN) cp --recursive $(CURDIR)/build/package/api/utils vxm/$(MINIO_BUCKET_NAME)/
	$(GOMINIO_BIN) config host add vxinst "$(MINIO_ENDPOINT)" $(MINIO_ACCESS_KEY) $(MINIO_SECRET_KEY) 2>/dev/null
	$(GOMINIO_BIN) mb --ignore-existing vxinst/$(AGENT_SERVER_MINIO_BUCKET_NAME)
	$(GOMINIO_BIN) cp --recursive $(CURDIR)/build/package/api/utils vxinst/$(AGENT_SERVER_MINIO_BUCKET_NAME)/

.PHONY: s3-upload-vxagent
s3-upload-vxagent: $(GOMINIO_BIN)
	$(eval VERSION := $(shell cat $(CURDIR)/build/artifacts/agent/version))
	$(eval VERSION_MAJ := $(shell echo "$(VERSION)" | cut -d '.' -f 1 | cut -c2-))
	$(eval VERSION_MIN := $(shell echo "$(VERSION)" | cut -d '.' -f 2))
	$(eval VERSION_PATCH := $(shell echo "$(VERSION)" | cut -d '.' -f 3 | cut -d '-' -f 1))
	$(eval VERSION_REV := $(shell echo "$(VERSION)" | cut -d '-' -f 2))
	$(eval BINARY_MD5 := $(shell md5sum $(CURDIR)/build/bin/vxagent | cut -d ' ' -f 1))
	$(eval BINARY_SHA256 := $(shell sha256sum $(CURDIR)/build/bin/vxagent | cut -d ' ' -f 1))
	$(eval GOOS := $(shell go env GOOS))
	$(eval GOARCH := $(shell go env GOARCH))
	@echo "Upload Agent binary: $(VERSION)-$(GOOS)-$(GOARCH)"
	@$(GOMINIO_BIN) cp $(CURDIR)/build/bin/vxagent vxm/$(MINIO_BUCKET_NAME)/vxagent/$(VERSION)/$(GOOS)/$(GOARCH)/
	@mysql --host=$(DB_HOST) --user=$(DB_ROOT_USER) --password=$(DB_ROOT_PASS) --port=$(DB_PORT) --database=vx_global --execute="SET @hash = MD5('$(VERSION)'), @info = '{\"files\": [\"vxagent/$(VERSION)/$(GOOS)/$(GOARCH)/vxagent\"], \"chksums\": {\"vxagent/$(VERSION)/$(GOOS)/$(GOARCH)/vxagent\": {\"md5\": \"$(BINARY_MD5)\", \"sha256\": \"$(BINARY_SHA256)\"}}, \"version\": {\"rev\": \"$(VERSION_REV)\", \"build\": 0, \"major\": $(VERSION_MAJ), \"minor\": $(VERSION_MIN), \"patch\": $(VERSION_PATCH)}}'; INSERT INTO \`binaries\` (\`hash\`, \`tenant_id\`, \`type\`, \`info\`) VALUES (@hash, 0, 'vxagent', @info) ON DUPLICATE KEY UPDATE hash = @hash;"

.PHONY: clean-all
clean-all: clean-build clean-web clean-sec

.PHONY: clean-build
clean-build:
	rm -rf \
		build/artifacts/* \
		build/bin/* \
		build/data/* \
		build/logs/*

.PHONY: clean-web
clean-web:
	rm -rf \
		web/.angular \
		web/dist \
		web/node_modules

.PHONY: clean-sec
clean-sec:
	rm -rf \
		build/ssl/* \
		scripts/certs_writer/certs_writer \
		scripts/sbh_generator/sbh_generator \
		security/certs \
		security/vconf/hardening/abh.json \
		security/vconf/lic/sbh.json \
		internal/app/api/utils/dbencryptor/sec-store-key.txt
