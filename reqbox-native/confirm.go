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

// confirmDialog is the web's fh: a 384px card, the title with a bad X
// (the confirm hook defaults to destructive), the description at 11.5 on
// relaxed lines, then cancel and the tinted destructive action.
func (a *app) confirmDialog(c *ui.Context, p palette) {
	ui.Modal(c, &a.confirm.open, func() {
		dialogShell(c, p, 384, func() {
			ui.Column(c).FillWidth().Gap(8).Children(func() {
				dialogTitle(c, p, a.confirm.title, true)
				ui.Text(c, a.confirm.desc).FontSize(11.5).LineHeight(1.625).
					TextColor(p.mutedFg.color())
			})
			ui.Row(c).FillWidth().Padding(4, 0, 0, 0).Gap(8).Justify(ui.End).
				Children(func() {
					btnOutlineSm(c, p, tr("common.cancel"), func() { a.confirm.open = false })
					btnDestructiveSm(c, p, a.confirm.action, func() {
						a.confirm.open = false
						if a.confirm.fn != nil {
							a.confirm.fn()
						}
					})
				})
		}, nil)
	})
}
