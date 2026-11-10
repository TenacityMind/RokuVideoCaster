package web

import "embed"

//go:embed index.html js/*
var FS embed.FS
