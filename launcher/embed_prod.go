//go:build prod

package main

import (
	"embed"
)

// Production mode: embed bundled assets
// These files are created by CI/CD before building

//go:embed bundled/backend
var embeddedBackend []byte

//go:embed all:bundled/frontend
var embeddedFrontend embed.FS

const isProduction = true
