package web

import "embed"

//go:embed templates/* images/*
var templatesFS embed.FS
