package main

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

var bodyTypes = []string{"none", "json", "text", "form"}
var authTypes = []string{"none", "bearer", "basic"}

// resetBodyBuf rebinds the code editor to the loaded request's body.
func (a *app) resetBodyBuf() {
	a.bodyBuf = editor.NewBuffer(a.edit.body)
	a.bodyBuf.SetLanguage(editor.LanguageFor("x.json"))
	a.bodyState = &editor.State{Buffer: a.bodyBuf, SoftWrap: true}
}

// bodyPane is the Body tab: a type select, the JSON/code editor for
// json/text bodies, and form KV rows for form bodies.
func (a *app) bodyPane(c *ui.Context, p palette) {
	ui.Row(c).Height(36).Padding(0, 16, 8, 16).Gap(8).AlignItems(ui.Center).
		Children(func() {
			ui.Text(c, "TYPE").FontSize(10).LetterSpacing(0.8).
				SingleLine().TextColor(p.mutedFg.color())
			if ui.Select(c, &a.edit.bodyType, bodyTypes).Width(100).
				Height(26).Changed() {
				a.edit.dirty = true
			}
		})
	switch a.edit.bodyType {
	case "form":
		a.kvEditor(c, p, "field", &a.edit.form)
	case "json", "text":
		editor.View(c, a.bodyState).Height(220).
			Padding(8, 16, 8, 16)
		if a.bodyBuf.Modified() {
			a.edit.dirty = true
		}
	}
}

// authPane is the Auth tab: type select and the credential fields of the
// chosen helper, mirroring the web Auth tab.
func (a *app) authPane(c *ui.Context, p palette) {
	ui.Row(c).Height(36).Padding(0, 16, 8, 16).Gap(8).AlignItems(ui.Center).
		Children(func() {
			ui.Text(c, "TYPE").FontSize(10).LetterSpacing(0.8).
				SingleLine().TextColor(p.mutedFg.color())
			if ui.Select(c, &a.edit.authType, authTypes).Width(100).
				Height(26).Changed() {
				a.edit.dirty = true
			}
		})
	field := func(label string, value *string, secret bool) {
		ui.Row(c).Height(32).Padding(0, 16, 0, 16).Gap(8).
			AlignItems(ui.Center).Children(func() {
			ui.Text(c, label).FontSize(11).SingleLine().
				TextColor(p.mutedFg.color()).Width(80)
			in := ui.TextInput(c, value).Grow(1).Height(28).FontSize(12)
			if secret {
				in = in.Password()
			}
			if in.Changed() {
				a.edit.dirty = true
			}
		})
	}
	switch a.edit.authType {
	case "bearer":
		field("Token", &a.edit.auth.Token, true)
	case "basic":
		field("Username", &a.edit.auth.Username, false)
		field("Password", &a.edit.auth.Password, true)
	}
}
