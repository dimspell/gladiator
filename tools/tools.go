//go:build tools

// This file exists only to retain testcontainers-go (and its subpackages) in
// go.mod even though the integration suite imports them behind the `integration`
// build tag. `go mod tidy` evaluates all build tags, so this keeps the
// dependency from being pruned. It is never compiled into the normal build.
package tools

import (
	_ "github.com/testcontainers/testcontainers-go"
	_ "github.com/testcontainers/testcontainers-go/wait"
)
