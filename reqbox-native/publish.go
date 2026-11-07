package main

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// publishPane is the Publish tab on the web's sF grid: intro, the enable
// card, the slug row with its suggest button, the mock URL bar once
// published, and the caller-overrides card, on a 512px stack with 12px
// gaps. The loopback gateway lives in this app (mock.go) and prefers
// 127.0.0.1:5178, falling back to an ephemeral port; the bar shows the
// address that is actually reachable, which is not always the web's fixed
// origin.
func (a *app) publishPane(c *ui.Context, p palette) {
	ui.Column(c).Padding(0, 16, 0, 16).Children(func() {
		ui.Column(c).MaxWidth(512).Gap(12).Children(func() {
			ui.Text(c, tr("publish.intro")).FontSize(12).LineHeight(1.625).
				TextColor(p.mutedFg.color())
			a.publishSwitch(c, p)
			ui.Column(c).Gap(4).Children(func() {
				eyebrow(c, p, "publish.slug")
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					a.slugInput(c, p)
					a.suggestBtn(c, p)
				})
				ui.Text(c, tr("publish.slugHint")).FontSize(10.5).
					TextColor(p.mutedFg.color().Alpha(0.8))
			})
			if a.edit.exposed && a.edit.exposeSlug != "" {
				a.mockURLBar(c, p)
			}
			// The web's mt-1 over the grid's 12px gap.
			ui.Row(c).Padding(4, 0, 0, 0).Children(func() {
				ui.Column(c).FillWidth().Padding(10, 10, 10, 10).Gap(6).Radius(6).
					Border(1, p.border.color().Alpha(0.6)).
					Background(p.muted.color().Alpha(0.3)).Children(func() {
					eyebrow(c, p, "publish.overrides")
					ui.Text(c, tr("publish.overridesHint")).FontSize(10.5).
						LineHeight(1.625).TextColor(p.mutedFg.color())
				})
			})
		})
	})
}

// publishSwitch is the enable card: the whole card is the check box, the
// 14px accent box beside the title and its hint.
func (a *app) publishSwitch(c *ui.Context, p palette) {
	cb := ui.CheckboxBase(c, &a.edit.exposed).Gap(10).
		AlignItems(ui.Start).Padding(10, 12, 10, 12).Radius(6).
		Border(1, p.border.color().Alpha(0.7))
	cb.Children(func() {
		ui.Row(c).Padding(2, 0, 0, 0).Children(func() {
			webBox(c, p, a.edit.exposed)
		})
		ui.Column(c).Grow(1).Children(func() {
			ui.Text(c, tr("publish.enable")).FontSize(12.5).FontWeight(500).
				LineHeight(1.4286)
			ui.Text(c, tr("publish.enableHint")).FontSize(10.5).
				LineHeight(1.4286).TextColor(p.mutedFg.color())
		})
	})
	if cb.Changed() {
		a.edit.dirty = true
		if a.edit.exposed {
			a.toastOK(tr("toast.mockPublished"))
		} else {
			a.toastOK(tr("toast.mockUnpublished"))
		}
	}
}

// slugInput is the web's Ym input for the slug: 32px mono on the input
// token border.
func (a *app) slugInput(c *ui.Context, p palette) {
	in := ui.TextInputBase(c, &a.edit.exposeSlug).Grow(1).Basis(0).MinWidth(0).
		Height(32).Radius(8).FontSize(14).Font("JetBrains Mono").
		Padding(0, 10, 0, 10).Placeholder("my-endpoint").
		Border(1, p.input.color())
	if in.Changed() {
		a.edit.dirty = true
	}
}

// suggestBtn is the web's outline sm button: paper background, muted on
// hover, the input token for border and ground in the dark variant.
func (a *app) suggestBtn(c *ui.Context, p palette) {
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
		ui.Text(c, tr("publish.suggest")).FontSize(12.8).SingleLine()
	})
	if e.Clicked() {
		a.edit.exposeSlug = a.suggestSlug(a.edit.name)
		a.edit.dirty = true
	}
}

// mockURLBar is the published address: the ok-tinted globe, the URL in
// mono, and a ghost copy button on a muted band.
func (a *app) mockURLBar(c *ui.Context, p palette) {
	addr := a.mockAddr
	if addr == "" {
		addr = "127.0.0.1:<gateway not running>"
	}
	url := "http://" + addr + "/mock/" + a.edit.exposeSlug
	ui.Row(c).Height(40).Radius(6).Background(p.muted.color().Alpha(0.5)).
		Padding(8, 10, 8, 10).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Icon(c, iconGlobe).Size(14, 14).TextColor(p.ok.color())
		ui.Text(c, url).FontSize(11).Font("JetBrains Mono").
			LineHeight(1.4286).SingleLine().Grow(1).MinWidth(0)
		cp := ui.Row(c).Height(24).Padding(0, 8, 0, 8).Radius(6).
			AlignItems(ui.Center)
		if cp.Hovered() {
			// The web's ghost hover.
			cp.Background(p.muted.color())
		}
		cp.Children(func() {
			ui.Text(c, tr("common.copy")).FontSize(12).SingleLine()
		})
		if cp.Clicked() {
			mygo.Clipboard.WriteText(url)
		}
	})
}

var slugRe = regexp.MustCompile(`[^a-z0-9-]+`)

// suggestSlug slugifies the request name and, when taken, appends -2, -3…
func (a *app) suggestSlug(name string) string {
	base := slugRe.ReplaceAllString(strings.ToLower(strings.Join(
		strings.Fields(name), "-")), "")
	base = strings.Trim(base, "-")
	if base == "" {
		base = "mock"
	}
	slug := base
	for i := 2; ; i++ {
		taken, err := a.st.SlugTaken(slug, a.edit.id)
		if err != nil || !taken {
			return slug
		}
		slug = base + "-" + strconv.Itoa(i)
	}
}
