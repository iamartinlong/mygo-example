package main

import (
	"github.com/egoist/mygo/ui"
)

// confirmState backs the shared destructive-action dialog, matching the
// web's confirm.* keys; fn runs only when the action button is chosen.
type confirmState struct {
	open   bool
	title  string
	desc   string
	action string
	fn     func()
}

func (a *app) askConfirm(title, desc, action string, fn func()) {
	a.confirm = confirmState{open: true, title: title, desc: desc, action: action, fn: fn}
}

func (a *app) confirmDialog(c *ui.Context, p palette) {
	ui.Modal(c, &a.confirm.open, func() {
		ui.Column(c).Width(420).Padding(16, 20, 16, 20).Gap(10).
			Background(p.card.color()).Radius(8).Children(func() {
			ui.Text(c, a.confirm.title).Bold()
			ui.Text(c, a.confirm.desc).FontSize(12).
				TextColor(p.mutedFg.color())
			ui.Row(c).Height(30).Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Row(c).Grow(1)
				e := btnSecondary(c, p, "Cancel")
				if e.Clicked() {
					a.confirm.open = false
				}
				e = ui.Row(c).Height(28).Padding(0, 14, 0, 14).
					AlignItems(ui.Center).Background(p.bad.color()).Children(func() {
					ui.Text(c, a.confirm.action).FontSize(12).SingleLine().
						TextColor(ui.Oklch(1, 0, 0))
				})
				if e.Clicked() {
					a.confirm.open = false
					if a.confirm.fn != nil {
						a.confirm.fn()
					}
				}
			})
		})
	})
}
