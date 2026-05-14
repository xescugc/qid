package assets

import (
	"embed"
)

// Assets defines the embedded files
//
//go:embed css/* js/* images/*
var Assets embed.FS
