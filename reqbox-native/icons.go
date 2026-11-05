package main

import (
	"embed"

	"github.com/egoist/mygo/ui"
)

// Lucide icons (ISC), the same set the web UI draws.
//
//go:embed icons/plus.svg icons/folder-plus.svg icons/chevron-down.svg icons/x.svg icons/send.svg icons/copy.svg icons/chevron-right.svg icons/upload.svg icons/download.svg icons/settings-2.svg icons/file-text.svg
var iconFS embed.FS

var (
	iconPlus        = mustIcon("icons/plus.svg")
	iconFolderPlus  = mustIcon("icons/folder-plus.svg")
	iconChevronDown = mustIcon("icons/chevron-down.svg")
	iconX           = mustIcon("icons/x.svg")
	iconSend        = mustIcon("icons/send.svg")
	iconCopy        = mustIcon("icons/copy.svg")
	iconChevronRt   = mustIcon("icons/chevron-right.svg")
	iconUpload      = mustIcon("icons/upload.svg")
	iconDownload    = mustIcon("icons/download.svg")
	iconSettings2   = mustIcon("icons/settings-2.svg")
	iconFileText    = mustIcon("icons/file-text.svg")
)

func mustIcon(path string) *ui.SVG {
	b, err := iconFS.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return ui.MustParseSVG(b)
}
