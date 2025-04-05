package app

import (
	"embed"
)

//go:embed templates/*
var resources embed.FS
