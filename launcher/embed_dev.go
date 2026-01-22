//go:build !prod

package main

import (
	"embed"
)

// Development mode: no embedded assets
// Launcher will run `go run` and `pnpm dev` commands

var embeddedBackend []byte
var embeddedFrontend embed.FS

const isProduction = false
