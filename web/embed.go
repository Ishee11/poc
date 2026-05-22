package web

import "embed"

//go:embed index.html manifest.webmanifest sw.js css/* js/** assets/*
var FS embed.FS
