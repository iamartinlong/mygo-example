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

// palette transcribes the oklch tokens of the web UI's :root (light) and
// .dark variants value-for-value, so the native UI matches it exactly.
// Method-band colors are the light-theme --m-* set.
type palette struct {
	background, card, secondary, accent, border lch
	foreground, mutedFg                         lch
	primary, primaryFg                          lch
	ring, ok, warn, bad                         lch
	panel, rail, chrome, hairline, input, muted lch
	secondaryFg, accentFg                       lch
	methods                                     methodColors
	isDark                                      bool
}

type methodColors struct {
	get, post, put, patch, delete, other lch
}

func (p palette) method(m string) lch {
	switch m {
	case "GET":
		return p.methods.get
	case "POST":
		return p.methods.post
	case "PUT":
		return p.methods.put
	case "PATCH":
		return p.methods.patch
	case "DELETE":
		return p.methods.delete
	default:
		return p.methods.other
	}
}

// web :root — light (blueprint paper)
var light = palette{
	background: lch{98.5, .003, 240}, card: lch{100, 0, 0},
	secondary: lch{95.5, .005, 240}, accent: lch{94.5, .008, 240},
	border:     lch{90, .006, 240},
	foreground: lch{22, .012, 250}, mutedFg: lch{53, .012, 250},
	primary: lch{28, .014, 250}, primaryFg: lch{98.5, .003, 240},
	ring: lch{55, .11, 250}, ok: lch{55, .11, 165}, warn: lch{62, .13, 70}, bad: lch{56, .17, 25},
	panel: lch{97, .004, 240}, rail: lch{95.5, .005, 240}, chrome: lch{98.5, .003, 240},
	hairline: lch{90, .006, 240}, input: lch{88, .006, 240}, muted: lch{95.5, .005, 240},
	secondaryFg: lch{30, .012, 250}, accentFg: lch{24, .012, 250},
	isDark: false,
	methods: methodColors{
		get:    lch{55, .11, 165},
		post:   lch{60, .12, 70},
		put:    lch{55, .12, 250},
		patch:  lch{56, .13, 305},
		delete: lch{56, .16, 25},
		other:  lch{55, .012, 250},
	},
}

// web .dark
var dark = palette{
	background: lch{16.5, .008, 250}, card: lch{20, .009, 250},
	secondary: lch{25, .009, 250}, accent: lch{27, .01, 250},
	border:     lch{29, .009, 250},
	foreground: lch{93, .004, 240}, mutedFg: lch{66, .012, 245},
	primary: lch{92, .005, 240}, primaryFg: lch{19, .009, 250},
	ring: lch{72, .03, 250}, ok: lch{78, .13, 165}, warn: lch{82, .13, 75}, bad: lch{70, .16, 25},
	panel: lch{18.5, .008, 250}, rail: lch{15.5, .008, 250}, chrome: lch{17.5, .008, 250},
	hairline: lch{27, .009, 250}, input: lch{33, .009, 250}, muted: lch{24.5, .009, 250},
	secondaryFg: lch{90, .004, 240}, accentFg: lch{95, .004, 240},
	isDark: true,
	methods: methodColors{
		get:    lch{78, .13, 165},
		post:   lch{80, .12, 75},
		put:    lch{74, .12, 250},
		patch:  lch{75, .13, 305},
		delete: lch{72, .15, 25},
		other:  lch{70, .012, 250},
	},
}

// theme lays the web tokens over the default widget theme; called at the
// top of the view so every widget follows them.
func theme(p palette, system *ui.Theme) *ui.Theme {
	t := *system
	t.Dark = p.isDark
	t.Background = p.background.color()
	t.Border = p.border.color()
	t.Text = p.foreground.color()
	t.TextMuted = p.mutedFg.color()
	t.Surface = p.card.color()
	t.SurfaceHover = p.secondary.color()
	t.SurfacePressed = p.accent.color()
	t.Accent = p.primary.color()
	t.AccentHover = p.primary.color()
	t.AccentPressed = p.primary.color()
	t.AccentText = p.primaryFg.color()
	t.Danger = p.bad.color()
	t.Warning = p.warn.color()
	t.Success = p.ok.color()
	t.Selection = p.accent.color()
	t.Focus = p.ring.color()
	t.FontSize = 12.5
	t.Font = "Archivo"
	// The web's --radius: rounded-lg inputs take 8px, its buttons keep 6.
	t.Radius = 8
	return &t
}
