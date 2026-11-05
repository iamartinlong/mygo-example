package main

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// publishPane is the Publish tab: publish toggle, slug with suggest, and the
// mock URL to copy. The loopback gateway lives in this app (mock.go) and
// prefers 127.0.0.1:5178, falling back to an ephemeral port.
func (a *app) publishPane(c *ui.Context, p palette) {
	ui.Column(c).Padding(8, 16, 8, 16).Gap(10).Children(func() {
		ui.Text(c, "发布后，该请求可通过一个固定 URL 被调用：系统会代为套用"+
			"已保存的 URL、请求头和认证信息。").FontSize(12).
			TextColor(p.mutedFg.color())
		if ui.Checkbox(c, &a.edit.exposed, "Publish as a mock endpoint").Changed() {
			a.edit.dirty = true
			if a.edit.exposed {
				a.toastOK("Mock published")
			} else {
				a.toastOK("Mock unpublished")
			}
		}
		ui.Text(c, "Requests are executed live, not replayed from cache.").
			FontSize(11.5).SingleLine().TextColor(p.mutedFg.color())
		if !a.edit.exposed {
			return
		}
		ui.Row(c).Height(30).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Slug").FontSize(12).Width(40).SingleLine()
			if ui.TextInput(c, &a.edit.exposeSlug).Grow(1).Height(28).
				FontSize(12).Changed() {
				a.edit.dirty = true
			}
			e := ui.Row(c).Height(28).Padding(0, 12, 0, 12).
				AlignItems(ui.Center).Background(p.secondary.color()).Children(func() {
				ui.Text(c, "Suggest").FontSize(12).SingleLine().
					TextColor(p.secondaryFg.color())
			})
			if e.Clicked() {
				a.edit.exposeSlug = a.suggestSlug(a.edit.name)
				a.edit.dirty = true
			}
		})
		ui.Text(c, "Letters, numbers, dashes. Must be unique.").FontSize(11.5).
			SingleLine().TextColor(p.mutedFg.color())
		addr := a.mockAddr
		if addr == "" {
			addr = "127.0.0.1:<gateway not running>"
		}
		url := "http://" + addr + "/mock/" + a.edit.exposeSlug
		ui.Row(c).Height(30).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, url).FontSize(11.5).Font("JetBrains Mono").Grow(1).
				SingleLine()
			e := ui.Row(c).Height(28).Padding(0, 12, 0, 12).
				AlignItems(ui.Center).Background(p.secondary.color()).Children(func() {
				ui.Text(c, "Copy").FontSize(12).SingleLine().
					TextColor(p.secondaryFg.color())
			})
			if e.Clicked() && a.edit.exposeSlug != "" {
				mygo.Clipboard.WriteText(url)
			}
		})
		ui.Text(c, "Caller overrides").FontSize(12)
		ui.Text(c, "Query parameters on the mock URL fill same-name "+
			"{{name}} placeholders in this request.").FontSize(11.5).
			TextColor(p.mutedFg.color())
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
