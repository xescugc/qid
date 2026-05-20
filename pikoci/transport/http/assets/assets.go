package assets

import (
	"embed"
)

// Assets defines the embedded files
//
//go:embed css/* js/* images/* fonts/*
var Assets embed.FS
