package static

import "embed"

// Assets embeds all static files (CSS, JS, images) into the Go binary.
//
//go:embed css/*
var Assets embed.FS
