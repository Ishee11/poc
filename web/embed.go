package web

import "embed"

//go:embed index.html manifest.webmanifest sw.js css/* js/**
var FS embed.FS
