package web

import "embed"

//go:embed index.html manifest.webmanifest sw.js css/* js/** assets/* svg/*
var FS embed.FS
