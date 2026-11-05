package main

import "github.com/egoist/mygo/ui"

// lch keeps each token readable next to its CSS source line; l is the CSS
// percentage, Oklch wants 0..1.
type lch struct {
	l, c, h float32
}

func (v lch) color() ui.Color {
	return ui.Oklch(v.l/100, v.c, v.h)
}

// palette transcribes the oklch tokens of the web UI's index.css
// value-for-value, so the native UI matches it exactly. Light is the web
// default (blueprint paper); dark mirrors the .dark variant. Method-band
// colors join the batch that first renders them (B2).
type palette struct {
	background, card, secondary, accent, border lch
	foreground, mutedFg                         lch
	primary, primaryFg                          lch
	ring, ok, warn, bad                         lch
}

var light = palette{
	background: lch{98.5, .003, 240}, card: lch{100, 0, 0}, secondary: lch{95.5, .005, 240},
	accent: lch{94.5, .008, 240}, border: lch{90, .006, 240},
	foreground: lch{22, .012, 250}, mutedFg: lch{66, .012, 245},
	primary: lch{92, .005, 240}, primaryFg: lch{19, .009, 250},
	ring: lch{72, .03, 250}, ok: lch{78, .13, 165}, warn: lch{82, .13, 75}, bad: lch{70, .16, 25},
}

var dark = palette{
	background: lch{16.5, .008, 250}, card: lch{20, .009, 250}, secondary: lch{25, .009, 250},
	accent: lch{27, .01, 250}, border: lch{29, .009, 250},
	foreground: lch{93, .004, 240}, mutedFg: lch{53, .012, 250},
	primary: lch{28, .014, 250}, primaryFg: lch{98.5, .003, 240},
	ring: lch{55, .11, 250}, ok: lch{55, .11, 165}, warn: lch{62, .13, 70}, bad: lch{56, .17, 25},
}

// theme lays the web tokens over the default widget theme; called at the
// top of the view so every widget follows them.
func theme(p palette, system *ui.Theme) *ui.Theme {
	t := *system
	t.Background = p.background.color()
	t.Border = p.border.color()
	t.Text = p.foreground.color()
	t.TextMuted = p.mutedFg.color()
	t.Accent = p.primary.color()
	t.AccentText = p.primaryFg.color()
	t.Danger = p.bad.color()
	t.Warning = p.warn.color()
	t.Success = p.ok.color()
	t.Selection = p.accent.color()
	t.Focus = p.ring.color()
	t.FontSize = 12.5
	t.Font = "Archivo"
	return &t
}
