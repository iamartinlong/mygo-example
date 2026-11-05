// Reqbox in native UI: the web frontend rebuilt with mygo's ui package,
// talking to the same store in-process. Route B of the migration
// (docs/01-research/reqbox-migration.md); the webview shell in
// reqbox-desktop/ stays as the 100%-fidelity baseline.
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type app struct {
	dark    bool    // follows the web .dark variant; a toggle arrives in B13
	sidebar float32 // sidebar width in DIPs, ui.Split drags it
}

func (a *app) view(c *ui.Context) {
	p := a.pal()
	c.SetTheme(theme(p, c.Theme()))

	ui.Column(c).Fill().Children(func() {
		a.topBar(c, p)
		ui.Split(c, &a.sidebar, func() { a.sidebarPane(c, p) }, func() { a.mainPane(c, p) }).Fill()
	})
}

func (a *app) pal() palette {
	if a.dark {
		return dark
	}
	return light
}

// topBar mirrors the web chrome: wordmark + LOCAL badge left, health right.
func (a *app) topBar(c *ui.Context, p palette) {
	ui.Row(c).Height(40).Padding(0, 12, 0, 12).Gap(8).
		AlignItems(ui.Center).Children(func() {
		ui.Text(c, "Reqbox").Bold().Grow(1)
		ui.Text(c, "LOCAL").FontSize(9).LetterSpacing(0.6).Font("JetBrains Mono").
			TextColor(p.mutedFg.color())
		ui.Row(c).Gap(5).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Width(7).Height(7).Background(p.ok.color())
			ui.Text(c, "server online").FontSize(11).TextColor(p.mutedFg.color())
		})
	})
}

func (a *app) sidebarPane(c *ui.Context, p palette) {
	ui.Column(c).Fill().Background(p.background.color()).Children(func() {
		// eyebrow, as the web sidebar's section headers; the list is B2
		ui.Row(c).Padding(12, 12, 6, 12).Children(func() {
			ui.Text(c, "REQUESTS").FontSize(10).LetterSpacing(0.8).
				TextColor(p.mutedFg.color())
		})
	})
}

func (a *app) mainPane(c *ui.Context, p palette) {
	// workspace empty-state frame; its copy lands with the send flow (B5)
	ui.Column(c).Fill().Background(p.card.color()).AlignItems(ui.Center)
}

func main() {
	a := &app{sidebar: 260}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:     "Reqbox",
			Width:     1280,
			Height:    820,
			MinWidth:  960,
			MinHeight: 600,
			Content:   ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
