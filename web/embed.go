package web

import "embed"

//go:embed index.html css/* js/* js/ui/*
var FS embed.FS
