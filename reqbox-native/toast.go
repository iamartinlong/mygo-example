package main

import "github.com/egoist/mygo/ui"

// toastMsg is a queued toast: some sources (the run goroutine) have no
// build context, so toasts drain at the top of the view.
type toastMsg struct {
	title, desc string
	bad         bool
}

func (a *app) toastOK(title string) {
	a.toasts = append(a.toasts, toastMsg{title: title})
}

func (a *app) toastErr(title string) {
	a.toasts = append(a.toasts, toastMsg{title: title, bad: true})
}

// drainToasts fires queued toasts and builds the toast viewport, bottom
// right as on the web.
func (a *app) drainToasts(c *ui.Context, p palette) {
	for _, t := range a.toasts {
		ty := "success"
		if t.bad {
			ty = "error"
		}
		c.AddToast(ui.Toast{Title: t.title, Description: t.desc, Type: ty})
	}
	a.toasts = nil
	ui.ToastViewportBase(c, func(viewport ui.Element, toasts []ui.Toast) {
		viewport.Padding(16).AlignItems(ui.End).Gap(8)
		for _, t := range toasts {
			toast := ui.ToastBase(c, t)
			toast.Root.Row().Gap(10).Padding(10, 14).Radius(8).MaxWidth(380).
				AlignItems(ui.Center).Background(p.foreground.color()).
				Shadow(0, 6, 20, 0, ui.RGBA(0, 0, 0, 0.25))
			toast.Root.Children(func() {
				dot := p.ok
				if t.Type == "error" {
					dot = p.bad
				}
				ui.Column(c).Width(7).Height(7).Radius(4).Background(dot.color())
				if t.Description == "" {
					ui.Text(c, t.Title).FontSize(12).SingleLine().
						TextColor(p.primaryFg.color())
				} else {
					ui.Column(c).Gap(2).Children(func() {
						ui.Text(c, t.Title).FontSize(12).Bold().SingleLine().
							TextColor(p.primaryFg.color())
						ui.Text(c, t.Description).FontSize(11.5).
							TextColor(p.primaryFg.color().Alpha(0.75))
					})
				}
				cb := toast.CloseButton()
				cb.Label("Close").Children(func() {
					ui.Text(c, "×").FontSize(14).
						TextColor(p.primaryFg.color().Alpha(0.6))
				})
			})
		}
	})
}
