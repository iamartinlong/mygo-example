package main

import (
	"embed"

	"github.com/egoist/mygo/ui"
)

// Lucide icons (ISC), the same set the web UI draws.
//
//go:embed icons/plus.svg icons/folder-plus.svg icons/chevron-down.svg
var iconFS embed.FS

var (
	iconPlus        = mustIcon("icons/plus.svg")
	iconFolderPlus  = mustIcon("icons/folder-plus.svg")
	iconChevronDown = mustIcon("icons/chevron-down.svg")
)

func mustIcon(path string) *ui.SVG {
	b, err := iconFS.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return ui.MustParseSVG(b)
}
