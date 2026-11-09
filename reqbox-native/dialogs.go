package main

import (
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"reqbox-desktop/pkg/runner"
)

// Dialog scaffolding on the web's shadcn dialogs: a popover-grounded card
// with a hairline ring of foreground/10, 12px corners, 16px padding and
// 16px part gaps.

// dialogShell centres a w-wide dialog card; body fills the padded content
// area, footer draws below it inside the ring (band footers clip against
// the card's rounded bottom).
func dialogShell(c *ui.Context, p palette, w float32, body, footer func()) {
	ui.Column(c).Width(w).Clip().Background(p.popover.color()).Radius(12).
		Border(1, p.foreground.color().Alpha(0.1)).Children(func() {
		ui.Column(c).FillWidth().Padding(16, 16, 16, 16).Gap(16).Children(body)
		if footer != nil {
			footer()
		}
	})
}

// dialogTitle is the web's DialogTitle: 14px medium, with a bad X in front
// of destructive confirms.
func dialogTitle(c *ui.Context, p palette, title string, destructive bool) {
	ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Start).Children(func() {
		if destructive {
			ui.Icon(c, iconX).Size(16, 16).TextColor(p.bad.color())
		}
		ui.Text(c, title).FontSize(14).FontWeight(500).Grow(1)
	})
}

// btnOutlineSm is the web's outline sm button, as suggestBtn draws it.
func btnOutlineSm(c *ui.Context, p palette, label string, fn func()) {
	border, bg := p.border.color(), p.background.color()
	if p.isDark {
		border, bg = p.input.color(), p.input.color().Alpha(0.3)
	}
	e := ui.Row(c).Height(28).Padding(0, 10, 0, 10).Radius(6).
		AlignItems(ui.Center).Border(1, border).Background(bg)
	if e.Hovered() {
		if p.isDark {
			e.Background(p.input.color().Alpha(0.5))
		} else {
			e.Background(p.muted.color())
		}
	}
	e.Children(func() {
		ui.Text(c, label).FontSize(12.8).FontWeight(500).SingleLine()
	})
	if e.Clicked() && fn != nil {
		fn()
	}
}

// btnPrimarySm is the web's default sm button.
func btnPrimarySm(c *ui.Context, p palette, label string, fn func()) {
	e := ui.Row(c).Height(28).Padding(0, 10, 0, 10).Radius(6).
		AlignItems(ui.Center).Background(p.primary.color())
	if e.Hovered() {
		e.Background(p.primary.color().Alpha(0.8))
	}
	e.Children(func() {
		ui.Text(c, label).FontSize(12.8).FontWeight(500).SingleLine().
			TextColor(p.primaryFg.color())
	})
	if e.Clicked() && fn != nil {
		fn()
	}
}

// btnDestructiveSm is the web's destructive variant: a bad tint rather
// than a filled red block.
func btnDestructiveSm(c *ui.Context, p palette, label string, fn func()) {
	e := ui.Row(c).Height(28).Padding(0, 10, 0, 10).Radius(6).
		AlignItems(ui.Center).Background(p.bad.color().Alpha(0.1))
	if e.Hovered() {
		e.Background(p.bad.color().Alpha(0.2))
	}
	e.Children(func() {
		ui.Text(c, label).FontSize(12.8).FontWeight(500).SingleLine().
			TextColor(p.bad.color())
	})
	if e.Clicked() && fn != nil {
		fn()
	}
}

// promptState backs the web's dh dialog: one text field asked over a
// cancel/confirm pair, e.g. the new-collection name.
type promptState struct {
	open         bool
	title        string
	desc         string
	label        string
	value        string
	placeholder  string
	confirmLabel string
	fn           func(string)
}

func (a *app) askPrompt(title, desc, label, value, placeholder, confirmLabel string, fn func(string)) {
	a.prompt = promptState{open: true, title: title, desc: desc, label: label,
		value: value, placeholder: placeholder, confirmLabel: confirmLabel, fn: fn}
}

// promptDialog is the web's dh: title, optional description, an eyebrow
// field with an optional hint, then cancel and confirm.
func (a *app) promptDialog(c *ui.Context, p palette) {
	ui.Modal(c, &a.prompt.open, func() {
		dialogShell(c, p, 384, func() {
			ui.Column(c).FillWidth().Gap(8).Children(func() {
				// The prompt keeps the DialogTitle default: 16px medium.
				ui.Text(c, a.prompt.title).FontSize(16).FontWeight(500)
				if a.prompt.desc != "" {
					ui.Text(c, a.prompt.desc).FontSize(14).
						TextColor(p.mutedFg.color())
				}
			})
			ui.Column(c).FillWidth().Gap(4).Children(func() {
				// The field label is raw wording, not an i18n key, so the
				// eyebrow look goes inline.
				ui.Text(c, strings.ToUpper(a.prompt.label)).FontSize(11).
					LetterSpacing(1.54).LineHeight(1.4286).Font("JetBrains Mono").
					SingleLine().TextColor(p.mutedFg.color())
				in := ui.TextInputBase(c, &a.prompt.value).FillWidth().Height(32).
					Radius(8).FontSize(12).Padding(0, 10, 0, 10).
					Border(1, p.input.color()).Placeholder(a.prompt.placeholder).
					AutoFocus()
				focusedRing(p, in)
			})
			ui.Row(c).FillWidth().Padding(4, 0, 0, 0).Gap(8).Justify(ui.End).
				Children(func() {
					btnOutlineSm(c, p, tr("common.cancel"), func() { a.prompt.open = false })
					btnPrimarySm(c, p, a.prompt.confirmLabel, func() {
						a.prompt.open = false
						if a.prompt.fn != nil {
							a.prompt.fn(a.prompt.value)
						}
					})
				})
		}, nil)
	})
}

// curlState backs the web's vF dialog: the request as a curl command in a
// read-only editor, with a copy button.
type curlState struct {
	open bool
	text string
	st   *editor.State
}

// openCurlDialog builds the curl command for the workspace's request, as
// the web's Gm.curl does.
func (a *app) openCurlDialog() {
	req := a.editRequest()
	a.curl.text = runner.EncodeCurl(req, a.resolveEnvVars())
	buf := editor.NewBuffer(a.curl.text)
	buf.SetLanguage(editor.LanguageFor("plain.txt"))
	a.curl.st = &editor.State{Buffer: buf, ReadOnly: true,
		Font: ui.Font{Family: "JetBrains Mono", Size: 12.5}, LineHeight: 1.65}
	a.curl.open = true
}

// curlDialog is the web's vF: terminal-titled, the command in a framed
// read-only editor, a copy button at the right.
func (a *app) curlDialog(c *ui.Context, p palette) {
	ui.Modal(c, &a.curl.open, func() {
		dialogShell(c, p, 672, func() {
			ui.Column(c).FillWidth().Gap(8).Children(func() {
				ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, iconTerminal).Size(16, 16).
						TextColor(p.foreground.color())
					ui.Text(c, tr("curl.title")).FontSize(14).FontWeight(500).
						SingleLine()
				})
				ui.Text(c, tr("curl.subtitle")).FontSize(11).
					TextColor(p.mutedFg.color())
			})
			// The web's $P editor frame: rounded, hairline border, a tinted
			// ground and 10px of lead above the text.
			ui.Column(c).FillWidth().Height(160).Radius(6).
				Border(1, p.border.color().Alpha(0.7)).
				Background(p.background.color().Alpha(0.4)).Clip().
				Children(func() {
					ui.Row(c).Height(10)
					if a.curl.st != nil {
						editor.View(c, a.curl.st).Fill()
					}
				})
			ui.Row(c).FillWidth().Gap(8).Justify(ui.End).Children(func() {
				btnOutlineSm(c, p, tr("common.copy"), func() {
					mygo.Clipboard.WriteText(a.curl.text)
					a.toastOK(tr("common.copied"))
				})
			})
		}, nil)
	})
}

// promptCollection opens the new-collection prompt with the web's wording.
func (a *app) promptCollection() {
	a.askPrompt(tr("prompt.newCollection.title"), tr("prompt.newCollection.desc"),
		tr("prompt.newCollection.label"), "", tr("prompt.newCollection.placeholder"),
		tr("prompt.newCollection.confirm"), func(v string) {
			name := strings.TrimSpace(v)
			if name == "" || a.st == nil {
				return
			}
			a.st.CreateCollection(name)
			a.toastOK(tr("toast.collectionCreated"))
			a.refresh()
		})
}
