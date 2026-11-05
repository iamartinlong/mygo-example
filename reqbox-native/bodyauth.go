package main

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

var bodyTypes = []string{"none", "json", "text", "form"}
var authTypes = []string{"none", "bearer", "basic"}

// bodyKeys and authKeys give each stored type value its label key.
var bodyKeys = map[string]string{
	"none": "body.typeNone", "json": "body.typeJson",
	"text": "body.typeText", "form": "body.typeForm",
}
var authKeys = map[string]string{
	"none": "common.none", "bearer": "auth.bearer", "basic": "auth.basic",
}

// resetBodyBuf rebinds the code editor to the loaded request's body.
func (a *app) resetBodyBuf() {
	a.bodyBuf = editor.NewBuffer(a.edit.body)
	a.bodyBuf.SetLanguage(editor.LanguageFor("x.json"))
	a.bodyState = &editor.State{Buffer: a.bodyBuf, SoftWrap: true}
}

// typeSelect is a select of stored values with translated labels, as the
// Body and Auth type pickers.
func typeSelect(c *ui.Context, value *string, values []string, keys map[string]string, w float32) {
	sel := ui.SelectBase(c, value)
	sel.Trigger.Width(w).Height(26).Children(func() {
		ui.Text(c, tr(keys[*value])).FontSize(12)
	})
	sel.Popup(func(panel ui.Element) {
		panel.MinWidth(120).Padding(4)
		for _, v := range values {
			v := v
			it := sel.Item(v).Padding(6, 10)
			it.Children(func() { ui.Text(c, tr(keys[v])).FontSize(12) })
		}
	})
}

// bodyPane is the Body tab: a type select, the JSON/code editor for
// json/text bodies, and form KV rows for form bodies.
func (a *app) bodyPane(c *ui.Context, p palette) {
	ui.Row(c).Height(36).Padding(0, 16, 8, 16).Gap(8).AlignItems(ui.Center).
		Children(func() {
			ui.Text(c, "TYPE").FontSize(10).LetterSpacing(0.8).
				SingleLine().TextColor(p.mutedFg.color())
			typeSelect(c, &a.edit.bodyType, bodyTypes, bodyKeys, 100)
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
			typeSelect(c, &a.edit.authType, authTypes, authKeys, 100)
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
		field(tr("auth.token"), &a.edit.auth.Token, true)
	case "basic":
		field(tr("auth.username"), &a.edit.auth.Username, false)
		field(tr("auth.password"), &a.edit.auth.Password, true)
	}
}
