package assets

import (
	"embed"
)

// Assets defines the embedded files
//
//go:embed css/* js/*
var Assets embed.FS
