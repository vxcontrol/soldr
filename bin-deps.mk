.PHONY: go-get
go-get:
	go get ./...

GOLANGCI_BIN=$(LOCAL_BIN)/golangci-lint
.PHONY: $(GOLANGCI_BIN)
$(GOLANGCI_BIN):
	GOBIN=$(LOCAL_BIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

GOSWAGGER_BIN=$(LOCAL_BIN)/swag
.PHONY: $(GOSWAGGER_BIN)
$(GOSWAGGER_BIN):
	GOBIN=$(LOCAL_BIN) go install github.com/swaggo/swag/cmd/swag@v1.8.7

GOMINIO_BIN=$(LOCAL_BIN)/mc
.PHONY: $(GOMINIO_BIN)
$(GOMINIO_BIN):
	GOBIN=$(LOCAL_BIN) go install github.com/minio/mc@latest
