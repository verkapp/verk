package main

import (
	"embed"

	"verk/src"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	src.Run(assets)
}
