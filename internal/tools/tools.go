//go:build tools

package tools

// list packages here to prevent them from removal out of go.mod
import (
	_ "github.com/go-swagger/go-swagger/cmd/swagger"
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
)
