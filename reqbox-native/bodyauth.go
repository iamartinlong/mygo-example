package main

import (
	"encoding/json"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

var bodyTypes = []string{"none", "json", "text", "xml", "form"}
var authTypes = []string{"none", "bearer", "basic", "apikey"}

// bodyKeys and authKeys give each stored type value its label key.
var bodyKeys = map[string]string{
	"none": "body.typeNone", "json": "body.typeJson",
	"text": "body.typeText", "xml": "body.typeXml",
	"form": "body.typeForm",
}
var authKeys = map[string]string{
	"none": "common.none", "bearer": "auth.bearer",
	"basic": "auth.basic", "apikey": "auth.apikey",
}

var addToTypes = []string{"header", "query"}
var addToKeys = map[string]string{
	"header": "auth.inHeader", "query": "auth.inQuery",
}

// resetBodyBuf rebinds the code editor to the loaded request's body.
func (a *app) resetBodyBuf() {
	a.bodyBuf = editor.NewBuffer(a.edit.body)
	a.bindBodyBuf()
}

// bindBodyBuf points the editor at the language and placeholder of the
// current body type; the text itself stays in the buffer, as the web keeps
// one draft body string across the types.
func (a *app) bindBodyBuf() {
	var lang *editor.Language
	ph := tr("body.placeholder.raw")
	if a.edit.bodyType == "json" {
		lang = editor.LanguageFor("x.json")
		ph = tr("body.placeholder.json")
	}
	a.bodyBuf.SetLanguage(lang)
	a.bodyState = &editor.State{
		Buffer: a.bodyBuf, SoftWrap: true, Placeholder: ph,
		// The web's editor is 12.5px mono on 1.65 lines, without a fold
		// gutter; its active line is the muted/60 band MyGo draws.
		Font:       ui.Font{Family: "JetBrains Mono", Size: 12.5},
		LineHeight: 1.65,
		NoFolding:  true,
	}
}

// eyebrow is the web's .eyebrow label: mono 11px, tracked and uppercased.
func eyebrow(c *ui.Context, p palette, key string) {
	ui.Text(c, strings.ToUpper(tr(key))).FontSize(11).LetterSpacing(1.54).
		LineHeight(1.4286).Font("JetBrains Mono").SingleLine().
		TextColor(p.mutedFg.color())
}

// dashedNote is the web's dashed empty note: 12px muted text, with a lock
// icon in front when one is given; w of 0 spans the row.
func dashedNote(c *ui.Context, p palette, w, h float32, icon *ui.SVG, text string, center bool) {
	e := ui.Row(c).Height(h).Padding(0, 12, 0, 12).Gap(8).
		AlignItems(ui.Center).Radius(6).
		Border(1, p.border.color().Alpha(0.7)).BorderStyle(ui.BorderDashed)
	if w == 0 {
		e = e.Fill()
	} else {
		e = e.Width(w)
	}
	if center {
		e = e.Center()
	}
	e.Children(func() {
		if icon != nil {
			ui.Icon(c, icon).Size(14, 14).TextColor(p.mutedFg.color())
		}
		ui.Text(c, text).FontSize(12).SingleLine().
			TextColor(p.mutedFg.color())
	})
}

// pickType draws a type select and marks the draft dirty when the choice
// changes, as the web's setDraft does; SelectBase writes *value in the
// same frame it is chosen.
func (a *app) pickType(c *ui.Context, p palette, value *string, values []string, keys map[string]string, w float32) {
	before := *value
	typeSelect(c, p, value, values, keys, w)
	if *value != before {
		a.edit.dirty = true
	}
}

// typeLabel names a stored type value; a value the keys do not cover, as an
// older request's "raw" body, shows as it is.
func typeLabel(keys map[string]string, v string) string {
	if k, ok := keys[v]; ok {
		return tr(k)
	}
	return v
}

func typeSelect(c *ui.Context, p palette, value *string, values []string, keys map[string]string, w float32) {
	sel := ui.SelectBase(c, value)
	sel.Trigger.Width(w).Height(28).Padding(0, 8, 0, 10).Gap(6).Radius(6).
		Border(1, p.border.color()).AlignItems(ui.Center).Children(func() {
		ui.Text(c, typeLabel(keys, *value)).FontSize(14).Grow(1).SingleLine()
		ui.Icon(c, iconChevronDown).Size(16, 16).TextColor(p.mutedFg.color())
	})
	sel.Popup(func(panel ui.Element) {
		panel.MinWidth(w).Padding(4).Background(p.card.color()).
			Radius(8).Border(1, p.border.color())
		for _, v := range values {
			v := v
			it := sel.Item(v).Padding(4, 32, 4, 6)
			it.Children(func() { ui.Text(c, typeLabel(keys, v)).FontSize(14) })
		}
	})
}

// bodyTypePicker is the web's segmented control: one muted track whose
// active button sits on the background colour.
func (a *app) bodyTypePicker(c *ui.Context, p palette) {
	ui.Row(c).Height(28).Padding(2).Gap(2).Radius(8).AlignItems(ui.Center).
		Background(p.muted.color().Alpha(0.6)).Children(func() {
		for _, t := range bodyTypes {
			t := t
			on := a.edit.bodyType == t
			b := ui.Row(c).Height(24).Padding(0, 8, 0, 8).Radius(6).
				AlignItems(ui.Center).Center()
			if on {
				b = b.Background(p.background.color()).
					Shadow(0, 1, 2, 0, p.foreground.color().Alpha(0.15))
			}
			ink := p.mutedFg.color()
			if on || b.Hovered() {
				ink = p.foreground.color()
			}
			b.Children(func() {
				ui.Text(c, tr(bodyKeys[t])).FontSize(11.5).FontWeight(500).
					SingleLine().TextColor(ink)
			})
			if b.Clicked() {
				a.edit.bodyType = t
				a.edit.dirty = true
				a.bindBodyBuf()
			}
		}
	})
}

// formatBody pretty-prints the buffer, as the web's Format button does: a
// body that does not parse is left as it is.
func (a *app) formatBody() {
	var v any
	if err := json.Unmarshal([]byte(a.bodyBuf.String()), &v); err != nil {
		return
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return
	}
	a.bodyBuf.Set(string(out))
	a.edit.dirty = true
}

// bodyPane is the Body tab: the segmented type control with its Format
// action, then the empty note, code editor or form rows of that type.
func (a *app) bodyPane(c *ui.Context, p palette) {
	ui.Row(c).Height(36).Padding(0, 16, 8, 16).Gap(8).AlignItems(ui.Center).
		Children(func() {
			a.bodyTypePicker(c, p)
			if a.edit.bodyType != "json" {
				return
			}
			ui.Row(c).Grow(1).Justify(ui.End).Children(func() {
				f := ui.Row(c).Height(24).Padding(0, 8, 0, 8).Radius(6).
					AlignItems(ui.Center).Center()
				ink := p.mutedFg.color()
				if f.Hovered() {
					// The web's ghost xs button: hover:bg-muted and a
					// foreground label.
					ink = p.foreground.color()
					f = f.Background(p.muted.color())
				}
				f.Children(func() {
					ui.Text(c, tr("body.format")).FontSize(12).FontWeight(500).
						SingleLine().TextColor(ink)
				})
				if f.Clicked() {
					a.formatBody()
				}
			})
		})
	// The header row's 8px bottom padding is the web's gap to what follows.
	ui.Column(c).Children(func() {
		switch a.edit.bodyType {
		case "form":
			a.kvEditor(c, p, "field", &a.edit.form)
		case "none":
			ui.Row(c).Height(128).Padding(0, 16, 0, 16).
				AlignItems(ui.Center).Children(func() {
				dashedNote(c, p, 0, 128, nil, tr("body.none"), true)
			})
		case "json", "text", "xml":
			// The web's cm-editor fills a bordered frame of 178px with 10px of
			// lead above its content; MyGo paints an editor over the whole box
			// it is given, so a spacer row carries that lead.
			ui.Row(c).Height(178).Padding(0, 16, 0, 16).Children(func() {
				ui.Column(c).Fill().Height(178).Radius(6).
					Border(1, p.border.color().Alpha(0.7)).Clip().Children(func() {
					ui.Row(c).Height(10)
					editor.View(c, a.bodyState).Fill()
				})
			})
			if a.bodyBuf.Modified() {
				a.edit.dirty = true
			}
		}
	})
}

// authField is one field of the web's auth grid: an eyebrow above the
// control and an optional hint below it.
func authField(c *ui.Context, p palette, w float32, labelKey, hintKey string, control func()) {
	ui.Column(c).Width(w).Gap(4).Children(func() {
		eyebrow(c, p, labelKey)
		control()
		if hintKey != "" {
			ui.Text(c, tr(hintKey)).FontSize(10.5).SingleLine().
				TextColor(p.mutedFg.color().Alpha(0.8))
		}
	})
}

// authPane is the Auth tab: the web's 448-wide stack of the type select and
// the credential fields of the chosen helper, then the variable hint.
func (a *app) authPane(c *ui.Context, p palette) {
	ui.Row(c).Padding(0, 16, 0, 16).Children(func() {
		ui.Column(c).Width(448).Gap(12).Children(func() {
			authField(c, p, 448, "auth.type", "", func() {
				a.pickType(c, p, &a.edit.authType, authTypes, authKeys, 448)
			})
			switch a.edit.authType {
			case "bearer":
				authField(c, p, 448, "auth.token", "auth.tokenHint", func() {
					a.authInput(c, 448, &a.edit.auth.Token, true, "eyJhbGciOi…")
				})
			case "basic":
				ui.Row(c).Gap(8).Children(func() {
					authField(c, p, 220, "auth.username", "", func() {
						a.authInput(c, 220, &a.edit.auth.Username, false, "")
					})
					authField(c, p, 220, "auth.password", "", func() {
						a.authInput(c, 220, &a.edit.auth.Password, true, "")
					})
				})
			case "apikey":
				ui.Column(c).Width(448).Gap(8).Children(func() {
					ui.Row(c).Gap(8).Children(func() {
						authField(c, p, 220, "auth.key", "", func() {
							a.authInput(c, 220, &a.edit.auth.Key, false, "X-Api-Key")
						})
						authField(c, p, 220, "kv.value", "", func() {
							a.authInput(c, 220, &a.edit.auth.Value, false, "")
						})
					})
					authField(c, p, 448, "auth.addTo", "", func() {
						if a.edit.auth.In == "" {
							a.edit.auth.In = "header"
						}
						a.pickType(c, p, &a.edit.auth.In, addToTypes, addToKeys, 448)
					})
				})
			case "none":
				dashedNote(c, p, 448, 66, iconLock, tr("auth.noCredentials"), false)
			}
			// The variable hint sits under every auth type, as in the web.
			ui.Row(c).Padding(4, 0, 0, 0).Children(func() {
				ui.Row(c).Width(448).Padding(8, 10, 8, 10).Gap(8).Radius(6).
					Background(p.muted.color().Alpha(0.4)).Children(func() {
					ui.Icon(c, iconKeyRound).Size(14, 14).
						TextColor(p.mutedFg.color())
					ui.Text(c, tr("auth.variableHint")).FontSize(10.5).
						LineHeight(1.625).TextColor(p.mutedFg.color())
				})
			})
		})
	})
}

// authInput is the web's credential input: 32px tall, mono 14px.
func (a *app) authInput(c *ui.Context, w float32, value *string, secret bool, ph string) {
	in := ui.TextInput(c, value).Width(w).Height(32).FontSize(14).
		Font("JetBrains Mono").Radius(8)
	if ph != "" {
		in = in.Placeholder(ph)
	}
	if secret {
		in = in.Password()
	}
	if in.Changed() {
		a.edit.dirty = true
	}
}
