package main

import (
	"embed"
	"log"

	"github.com/egoist/mygo/ui"
)

// OFL-licensed variable fonts, the same families the web UI uses
// (Archivo + JetBrains Mono; CJK and emoji fall back to system fonts,
// as the web version relies on the OS stack for those too).
//
//go:embed fonts/Archivo.ttf fonts/JetBrainsMono.ttf
var fonts embed.FS

func init() {
	for _, f := range []struct {
		path, family string
	}{
		{"fonts/Archivo.ttf", "Archivo"},
		{"fonts/JetBrainsMono.ttf", "JetBrains Mono"},
	} {
		b, err := fonts.ReadFile(f.path)
		if err != nil {
			log.Fatal(err)
		}
		if err := ui.RegisterFont(b, f.family); err != nil {
			log.Fatal(err)
		}
	}
}
