package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"reqbox-desktop/pkg/runner"
	"reqbox-desktop/pkg/store"
)

var methods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}

// reqEdit is the request loaded into the workspace pane; Save writes it back.
type reqEdit struct {
	id         string
	colID      string
	name       string
	method     string
	url        string
	params     []store.KV
	headers    []store.KV
	bodyType   string
	body       string
	form       []store.KV
	authType   string
	auth       store.Auth
	exposed    bool
	exposeSlug string
	tab        string // Params / Headers / Body / Auth / Publish
	dirty      bool
}

func (a *app) loadEdit() {
	a.edit = reqEdit{tab: "Params"}
	if a.st == nil || a.selectedID == "" {
		return
	}
	r, err := a.st.GetRequest(a.selectedID)
	if err != nil {
		return
	}
	a.edit = reqEdit{
		id: r.ID, colID: ptrStr(r.CollectionID),
		name: r.Name, method: r.Method, url: r.URL,
		params: r.Params, headers: r.Headers,
		bodyType: r.BodyType, body: r.Body, form: r.Form,
		authType: r.AuthType, auth: r.Auth,
		exposed: r.Exposed, exposeSlug: r.ExposeSlug,
		tab: "Params",
	}
	a.resetBodyBuf()
}

func ptrStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func kvID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// mainPane is the workspace: title row, request bar, tab strip, the
// Params/Headers/Body/Auth editors, then the run bar and response pane.
func (a *app) mainPane(c *ui.Context, p palette) {
	if a.edit.id != a.selectedID {
		a.loadEdit()
	}
	ui.Column(c).Fill().Background(p.background.color()).Children(func() {
		if a.edit.id == "" {
			a.emptyState(c, p)
			return
		}
		a.titleRow(c, p)
		a.requestBar(c, p)
		a.fullURLRow(c, p)
		a.curlRow(c, p)
		a.tabStrip(c, p)
		// The web's tab group and panel: an 8px gap to a panel padded 12px
		// top and bottom, so 20px of lead before the content and 20px after
		// it. The panel takes its content's height up to 46vh — 46% of the
		// pane here, the closest mygo's view gets to the viewport — and
		// scrolls beyond it, as the web's overflow-y-auto does. A tab
		// switch remounts the panel, so its scroll starts over at the top.
		if a.panelTab != a.edit.tab {
			a.panelTab = a.edit.tab
			a.panelScroll = ui.ScrollState{}
		}
		ui.Column(c).FillWidth().MaxHeightPercent(46).Children(func() {
			ui.Scroll(c).FillHeight().FillWidth().TrackScroll(&a.panelScroll).Children(func() {
				ui.Column(c).FillWidth().Padding(20, 0, 20, 0).Children(func() {
					switch a.edit.tab {
					case "Params":
						a.kvEditor(c, p, "parameter", &a.edit.params)
					case "Headers":
						a.kvEditor(c, p, "header", &a.edit.headers)
					case "Body":
						a.bodyPane(c, p)
					case "Auth":
						a.authPane(c, p)
					case "Publish":
						a.publishPane(c, p)
					}
				})
			})
		})
		a.runBar(c, p)
		a.responsePane(c, p)
	})
}

// emptyState is the web's workspace-empty splash: eyebrow, headline,
// description, a New-request button, then the three feature columns
// above a hairline.
func (a *app) emptyState(c *ui.Context, p palette) {
	ui.Column(c).Fill().AlignItems(ui.Center).Center().Padding(0, 24, 0, 24).
		Children(func() {
			ui.Column(c).Width(448).Gap(12).Children(func() {
				ui.Text(c, strings.ToUpper(tr("empty.eyebrow"))).
					FontSize(11).LetterSpacing(1.5).Font("JetBrains Mono").
					TextColor(p.mutedFg.color())
				ui.Text(c, tr("empty.title")).FontSize(21).Bold().LineHeight(1.25)
				ui.Text(c, tr("empty.desc")).FontSize(13).
					TextColor(p.mutedFg.color())
				ui.Row(c).Height(34).Gap(10).AlignItems(ui.Center).Children(func() {
					e := ui.Row(c).Height(28).Padding(0, 14, 0, 14).Gap(7).Radius(6).
						AlignItems(ui.Center).Background(p.foreground.color()).
						Children(func() {
							ui.Icon(c, iconFileText).Size(13, 13).
								TextColor(p.primaryFg.color())
							ui.Text(c, tr("empty.newRequest")).FontSize(12).
								SingleLine().TextColor(p.primaryFg.color())
						})
					if e.Clicked() {
						a.pending.newRequest = true
					}
					ui.Text(c, tr("empty.orPress")).FontSize(10.5).
						Font("JetBrains Mono").TextColor(p.mutedFg.color())
				})
				ui.Row(c).Fill().Height(1).Background(p.hairline.color())
				ui.Row(c).Gap(16).Children(func() {
					a.featureCol(c, p, tr("empty.featRequests"), tr("empty.featRequestsSub"))
					a.featureCol(c, p, tr("empty.featHistory"), tr("empty.featHistorySub"))
					a.featureCol(c, p, tr("empty.featMocks"), tr("empty.featMocksSub"))
				})
			})
		})
}

func (a *app) featureCol(c *ui.Context, p palette, title, sub string) {
	ui.Column(c).Grow(1).Gap(2).Children(func() {
		ui.Text(c, title).FontSize(12).Bold().SingleLine()
		ui.Text(c, sub).FontSize(10.5).SingleLine().
			TextColor(p.mutedFg.color())
	})
}

func (a *app) titleRow(c *ui.Context, p palette) {
	status := tr("bar.saved")
	if a.edit.dirty {
		status = tr("bar.unsavedChanges")
	}
	ui.Row(c).Height(48).Padding(0, 16, 0, 16).Gap(10).
		AlignItems(ui.Center).Children(func() {
		if ui.EditableText(c, &a.edit.name).FontSize(13.5).Bold().
			SingleLine().Changed() {
			a.edit.dirty = true
		}
		ui.Text(c, status).FontSize(11).Font("JetBrains Mono").SingleLine().
			TextColor(p.mutedFg.color()).Grow(1)
		a.saveBtn(c, p)
	})
}

func (a *app) saveBtn(c *ui.Context, p palette) {
	e := ui.Row(c).Height(28).Padding(0, 10, 0, 10).Gap(4).Radius(6).
		AlignItems(ui.Center).Background(p.foreground.color()).Children(func() {
		ui.Icon(c, iconSave).Size(12, 12).TextColor(p.primaryFg.color())
		ui.Text(c, tr("bar.save")).FontSize(12.8).FontWeight(500).SingleLine().
			TextColor(p.primaryFg.color())
	})
	if e.Clicked() {
		a.saveEdit()
	}
}

func (a *app) saveEdit() {
	if a.st == nil || a.edit.id == "" {
		return
	}
	r, err := a.st.GetRequest(a.edit.id)
	if err != nil {
		return
	}
	if strings.TrimSpace(a.edit.url) == "" {
		a.toastErr(tr("toast.addUrlFirst"))
		return
	}
	r.Name, r.Method, r.URL = a.edit.name, a.edit.method, a.edit.url
	r.Params, r.Headers = a.edit.params, a.edit.headers
	r.BodyType, r.Body, r.Form = a.edit.bodyType, a.bodyBuf.String(), a.edit.form
	r.AuthType, r.Auth = a.edit.authType, a.edit.auth
	r.Exposed, r.ExposeSlug = a.edit.exposed, a.edit.exposeSlug
	if _, err := a.st.UpdateRequest(r, false); err == nil {
		a.bodyBuf.MarkSaved()
		a.edit.dirty = false
		a.refresh()
		a.toastOK(tr("toast.requestSaved"))
	} else {
		a.toastErr(tr("toast.cannotSave"))
	}
}

func (a *app) requestBar(c *ui.Context, p palette) {
	ui.Row(c).Height(42).Padding(0, 16, 0, 16).AlignItems(ui.Center).
		Children(func() {
			// The web's segmented box: one rounded border around method,
			// URL, copy and SEND, drawn flush inside it. The method cell and
			// the URL field are the lookless variants, which draw no frame.
			box := ui.Row(c).Grow(1).Height(34).Radius(8).
				Border(1, p.border.color()).Background(p.background.color()).
				AlignItems(ui.Center)
			box.Children(func() {
				a.methodCell(c, p)
				if ui.TextInputBase(c, &a.edit.url).Placeholder(tr("bar.urlPlaceholder")).
					Grow(1).Height(32).FontSize(12.5).Font("JetBrains Mono").
					TextColor(p.foreground.color()).
					Padding(0, 12, 0, 12).Changed() {
					a.edit.dirty = true
				}
				e := ui.Row(c).Width(32).Height(32).AlignItems(ui.Center).Center().
					BorderWidth(0, 0, 0, 1).BorderColor(p.border.color()).
					Children(func() {
						ui.Icon(c, iconCopy).Size(14, 14).TextColor(p.mutedFg.color())
					})
				if e.Clicked() {
					mygo.Clipboard.WriteText(curlFor(a.edit.method, a.edit.url, a.edit.params))
				}
				a.sendBtn(c, p)
			})
			// The web's focus-within:border-ring/60 on the box.
			if box.FocusWithin() {
				box.BorderColor(p.ring.color().Alpha(0.6))
			}
		})
}

// methodCell is the segmented box's left cell: the method and a chevron,
// over a divider, opening the method list in a borderless select.
func (a *app) methodCell(c *ui.Context, p palette) {
	m := p.method(a.edit.method)
	a.methodPick = a.edit.method
	sel := ui.SelectBase(c, &a.methodPick)
	sel.Trigger.Width(92).Height(32).Padding(0, 10, 0, 10).Gap(4).
		AlignItems(ui.Center).BorderWidth(0, 1, 0, 0).BorderColor(p.border.color()).
		Children(func() {
			ui.Text(c, a.edit.method).FontSize(12).FontWeight(700).
				LetterSpacing(0.3).Font("JetBrains Mono").SingleLine().
				TextColor(m.color()).Grow(1)
			ui.Icon(c, iconChevronDown).Size(12, 12).
				TextColor(m.color().Alpha(0.5))
		})
	sel.Popup(func(panel ui.Element) {
		panel.MinWidth(144).Padding(4).Background(p.card.color()).
			Radius(6).Border(1, p.border.color())
		for _, method := range methods {
			it := sel.Item(method).Padding(6, 10)
			it.Children(func(name string) func() {
				return func() {
					ui.Text(c, name).FontSize(12).FontWeight(700).
						LetterSpacing(0.3).Font("JetBrains Mono").
						SingleLine().TextColor(p.method(name).color())
				}
			}(method))
		}
	})
	if a.methodPick != "" && a.methodPick != a.edit.method {
		a.edit.method = a.methodPick
		a.edit.dirty = true
	}
}

func (a *app) sendBtn(c *ui.Context, p palette) {
	e := ui.Row(c).Width(104).Height(32).Padding(0, 10, 0, 10).Gap(6).
		AlignItems(ui.Center).Center().Radius(0).
		BorderWidth(0, 0, 0, 1).BorderColor(p.border.color()).
		Background(p.primary.color()).Children(func() {
		if a.run.running {
			ui.Text(c, "…").FontSize(12).Font("JetBrains Mono").FontWeight(600).
				LetterSpacing(0.3).SingleLine().TextColor(p.primaryFg.color())
		} else {
			ui.Icon(c, iconSend).Size(14, 14).TextColor(p.primaryFg.color())
			ui.Text(c, tr("bar.send")).FontSize(12).FontWeight(600).LetterSpacing(0.3).
				Font("JetBrains Mono").SingleLine().TextColor(p.primaryFg.color())
		}
	})
	if e.Clicked() {
		a.send()
	}
}

func (a *app) fullURLRow(c *ui.Context, p palette) {
	ui.Row(c).Height(25).Padding(0, 16, 0, 16).Gap(6).
		AlignItems(ui.Center).Children(func() {
		e := ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconChevronRt).Size(11, 11).TextColor(p.mutedFg.color())
			ui.Text(c, tr("bar.fullUrl")).FontSize(10).SingleLine().
				TextColor(p.mutedFg.color().Alpha(0.7))
		})
		if e.Clicked() {
			a.showFullURL = !a.showFullURL
		}
	})
	if a.showFullURL {
		raw := fullURL(a.edit.url, a.edit.params)
		ui.Row(c).Padding(2, 16, 8, 16).Gap(8).Children(func() {
			ui.Text(c, raw).FontSize(11).
				Font("JetBrains Mono").SingleLine().TextColor(p.mutedFg.color())
			if strings.Contains(raw, "{{") {
				if res := runner.Resolve(raw, a.resolveEnvVars()); res != raw {
					ui.Text(c, res).FontSize(11).Font("JetBrains Mono").
						SingleLine().TextColor(p.ok.color())
				}
			}
		})
	}
}

// curlRow holds the cURL copy and Delete actions plus, flush right, the
// collection the request lives in.
func (a *app) curlRow(c *ui.Context, p palette) {
	label := tr("sidebar.unsorted")
	for _, col := range a.cols {
		if col.ID == a.edit.colID {
			label = col.Name
		}
	}
	// The web appends " · edited" to the same span while the draft differs
	// from what is on file.
	if a.edit.dirty && a.edit.id != "" {
		label += " · " + tr("bar.edited")
	}
	ui.Row(c).Height(31).Padding(0, 16, 6, 16).Gap(4).
		AlignItems(ui.Center).Children(func() {
		e := ui.Row(c).Height(24).Padding(0, 8, 0, 8).Gap(4).
			AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconTerminal).Size(14, 14).TextColor(p.mutedFg.color())
			ui.Text(c, tr("bar.curl")).FontSize(12).FontWeight(500).
				SingleLine().TextColor(p.mutedFg.color())
		})
		if e.Clicked() {
			mygo.Clipboard.WriteText(curlFor(a.edit.method, a.edit.url, a.edit.params))
		}
		del := ui.Row(c).Height(24).Padding(0, 8, 0, 8).AlignItems(ui.Center)
		del.Children(func() {
			// The web's hover:text-bad; the label is muted until then.
			ink := p.mutedFg.color()
			if del.Hovered() {
				ink = p.bad.color()
			}
			ui.Text(c, tr("common.delete")).FontSize(12).FontWeight(500).
				SingleLine().TextColor(ink)
		})
		if del.Clicked() {
			a.askConfirm(tr("confirm.deleteRequest.title", "name", a.edit.name),
				tr("confirm.deleteRequest.desc"),
				tr("confirm.deleteRequest.confirm"), func() { a.pending.deleteReq = a.edit.id })
		}
		ui.Row(c).Grow(1).Justify(ui.End).Children(func() {
			ui.Text(c, label).FontSize(10).Font("JetBrains Mono").
				SingleLine().TextColor(p.mutedFg.color().Alpha(0.6))
		})
	})
}

var tabKeys = map[string]string{
	"Params": "tabs.params", "Headers": "tabs.headers", "Body": "tabs.body",
	"Auth": "tabs.auth", "Publish": "tabs.publish",
}

func (a *app) tabStrip(c *ui.Context, p palette) {
	// The web's line tabs: no pill background, a hairline under the whole
	// strip, and a 2px foreground underline on the active tab.
	counts := map[string]int{
		"Params":  enabledKV(a.edit.params),
		"Headers": enabledKV(a.edit.headers),
	}
	// The web dots a tab when its panel differs from the default: a body
	// type, any credentials, an exposed request.
	dots := map[string]lch{
		"Body":    p.method("PUT"),
		"Auth":    p.method("POST"),
		"Publish": p.ok,
	}
	ui.Row(c).Height(32).Padding(0, 16, 0, 16).Gap(16).
		AlignItems(ui.End).BorderWidth(0, 0, 1, 0).
		BorderColor(p.border.color().Alpha(0.6)).Children(func() {
		for _, tab := range []string{"Params", "Headers", "Body", "Auth", "Publish"} {
			t := tab
			active := a.edit.tab == t
			ink := p.foreground.color().Alpha(0.6)
			if p.isDark {
				ink = p.mutedFg.color()
			}
			col := ui.Column(c).Height(32).
				AlignItems(ui.Start).Children(func() {
				ui.Row(c).Grow(1).Gap(4).AlignItems(ui.Center).Children(func() {
					tt := ui.Text(c, tr(tabKeys[t])).FontSize(12).FontWeight(500).
						SingleLine()
					if active {
						tt.TextColor(p.foreground.color())
					} else {
						tt.TextColor(ink)
					}
					if n := counts[t]; n > 0 {
						tabBadge(c, p, n)
					}
					if dot, ok := dots[t]; ok && a.tabDotted(t) {
						ui.Row(c).Width(6).Height(6).Radius(3).
							Background(dot.color())
					}
				})
			})
			if active {
				col.BorderWidth(0, 0, 2, 0).BorderColor(p.foreground.color())
			}
			if col.Clicked() {
				a.edit.tab = t
			}
		}
	})
}

// tabDotted says whether the web marks a tab's status dot.
func (a *app) tabDotted(tab string) bool {
	switch tab {
	case "Body":
		return a.edit.bodyType != "none"
	case "Auth":
		return a.edit.authType != "none"
	case "Publish":
		return a.edit.exposed
	}
	return false
}

// enabledKV counts the rows the web badges a tab for: enabled, with a key.
func enabledKV(kvs []store.KV) int {
	n := 0
	for _, kv := range kvs {
		if kv.Enabled && kv.Key != "" {
			n++
		}
	}
	return n
}

// tabBadge is the web's count chip beside a tab: mono 10px on --muted.
func tabBadge(c *ui.Context, p palette, n int) {
	ui.Row(c).Height(15).Padding(0, 4, 0, 4).Radius(4).AlignItems(ui.Center).
		Background(p.muted.color()).Children(func() {
		ui.Text(c, fmt.Sprint(n)).FontSize(10).Font("JetBrains Mono").
			SingleLine().TextColor(p.mutedFg.color())
	})
}

// kvAddKey names each editor's full-width add offer label.
var kvAddKey = map[string]string{
	"parameter": "kv.addQueryParam", "header": "kv.addHeader", "field": "kv.addFormField",
}

// kvPlaceholders are the web's eF pairs: what the key and value boxes
// suggest per editor kind.
var kvPlaceholders = map[string][2]string{
	"parameter": {"limit", "10"},
	"header":    {"Authorization", "Bearer …"},
	"field":     {"field", "value"},
}

// kvEditor edits one KV list on the web's tF grid: an eyebrow header over
// a border/60 hairline, rows of check box + key + value + delete, then
// either the dashed empty offer or the rows with a small add button, and
// the per-kind footer note. what is the kv.* noun: parameter, header or
// field.
func (a *app) kvEditor(c *ui.Context, p palette, what string, kvs *[]store.KV) {
	ph := kvPlaceholders[what]
	ui.Column(c).Padding(0, 16, 0, 16).Children(func() {
		// The web's grid: 24px check box, 1fr key, 1.4fr value, 32px
		// delete, on 8px gaps; px-1 inside the panel's px-4.
		ui.Row(c).Padding(0, 4, 6, 4).Gap(8).AlignItems(ui.Center).
			BorderWidth(0, 0, 1, 0).BorderColor(p.border.color().Alpha(0.6)).
			Children(func() {
				ui.Row(c).Width(24)
				kvEyebrow(c, p, tr("kv."+what), 1)
				kvEyebrow(c, p, tr("kv.value"), 1.4)
				ui.Row(c).Width(32)
			})
		if len(*kvs) == 0 {
			// The web's mt-2 h-16 dashed offer; no small button then.
			ui.Row(c).Height(8)
			a.kvAddButton(c, p, kvs, tr(kvAddKey[what]), true)
		} else {
			ui.Row(c).Height(4)
			for i := range *kvs {
				a.kvRow(c, p, ph, kvs, i)
			}
			ui.Row(c).Height(6)
			ui.Row(c).Children(func() {
				a.kvAddButton(c, p, kvs, tr("common.addRow"), false)
			})
		}
		switch what {
		case "parameter":
			// The web's mono footer only once rows would be appended.
			if enabledKV(*kvs) > 0 {
				ui.Row(c).Height(12)
				ui.Text(c, tr("kv.appendedOnSend")).FontSize(10.5).
					Font("JetBrains Mono").
					TextColor(p.mutedFg.color().Alpha(0.7))
			}
		case "header":
			ui.Row(c).Height(12)
			ui.Text(c, tr("kv.headerHint")).FontSize(10.5).
				TextColor(p.mutedFg.color().Alpha(0.7))
		}
	})
}

// kvEyebrow is the header's column label: the web's .eyebrow, grown onto
// its grid track.
func kvEyebrow(c *ui.Context, p palette, label string, grow float32) {
	ui.Text(c, strings.ToUpper(label)).FontSize(11).LetterSpacing(1.54).
		LineHeight(1.4286).Font("JetBrains Mono").SingleLine().
		TextColor(p.mutedFg.color()).Grow(grow).Basis(0).MinWidth(0)
}

// kvRow is one grid row: a 14px accent check box, the key and value
// inputs, and a delete button the row's hover reveals, the whole row at
// 45% while it is off, as the web's.
func (a *app) kvRow(c *ui.Context, p palette, ph [2]string, kvs *[]store.KV, i int) {
	kv := &(*kvs)[i]
	row := ui.Row(c).Height(32).Padding(0, 4, 0, 4).Gap(8).
		AlignItems(ui.Center).Radius(4)
	if !kv.Enabled {
		row.Opacity(0.45)
	}
	row.Children(func() {
		ui.Row(c).Width(24).AlignItems(ui.Center).Children(func() {
			webCheckbox(c, p, &kv.Enabled)
		})
		a.kvInput(c, p, &kv.Key, ph[0], 1)
		a.kvInput(c, p, &kv.Value, ph[1], 1.4)
		ui.Row(c).Width(32).AlignItems(ui.Center).Children(func() {
			del := ui.Row(c).Width(24).Height(24).Radius(6).
				AlignItems(ui.Center).Center()
			if del.Hovered() {
				// The web's ghost hover under the trash.
				del.Background(p.muted.color())
			}
			del.Children(func() {
				// The web's group-hover reveal.
				if row.Hovered() {
					ui.Icon(c, iconTrash).Size(14, 14).
						TextColor(p.foreground.color())
				}
			})
			if del.Clicked() {
				a.edit.dirty = true
				*kvs = append((*kvs)[:i], (*kvs)[i+1:]...)
			}
		})
	})
}

// kvInput is the web's KV text box: a 28px mono input without a frame
// until the row hovers or the box focuses. grow puts it on the 1fr or
// 1.4fr track.
func (a *app) kvInput(c *ui.Context, p palette, value *string, placeholder string, grow float32) {
	in := ui.TextInputBase(c, value).Grow(grow).Basis(0).MinWidth(0).
		Height(28).Radius(8).FontSize(14).Font("JetBrains Mono").
		Padding(0, 6, 0, 6).Placeholder(placeholder)
	ink := p.border.color().Alpha(0)
	if in.Hovered() {
		ink = p.border.color().Alpha(0.6)
	}
	if in.Focused() {
		ink = p.border.color()
	}
	in.Border(1, ink)
	if in.Changed() {
		a.edit.dirty = true
	}
}

// kvAddButton adds a row: the full-width dashed offer while the list is
// empty, the small ghost button under the rows otherwise, with the web's
// hover states.
func (a *app) kvAddButton(c *ui.Context, p palette, kvs *[]store.KV, label string, full bool) {
	var e ui.Element
	if full {
		e = ui.Row(c).Fill().Height(64).Radius(6).Gap(6).
			Border(1, p.border.color().Alpha(0.7)).BorderStyle(ui.BorderDashed).
			AlignItems(ui.Center).Center()
	} else {
		e = ui.Row(c).Height(24).Padding(0, 8, 0, 8).Gap(4).
			AlignItems(ui.Center)
	}
	ink := p.mutedFg.color()
	if full {
		if e.Hovered() {
			// The web's hover:border-border hover:bg-accent/40.
			e.Border(1, p.border.color()).Background(p.accent.color().Alpha(0.4))
			ink = p.foreground.color()
		}
	} else if e.Hovered() {
		// The web's ghost hover.
		e.Background(p.muted.color())
		ink = p.foreground.color()
	}
	e.Children(func() {
		ui.Icon(c, iconPlus).Size(14, 14).TextColor(ink)
		ui.Text(c, label).FontSize(12).SingleLine().TextColor(ink)
	})
	if e.Clicked() {
		a.edit.dirty = true
		*kvs = append(*kvs, store.KV{ID: kvID(), Enabled: true})
	}
}

// webBox paints the web's 14px accent check box as Chromium does: the
// ring colour under a check that contrasts with it when on; off, a white
// box on a grey frame in the light theme and a dark one in the dark
// theme, as the browser adapts the control to the page's ground.
func webBox(c *ui.Context, p palette, checked bool) {
	box := ui.Box(c).Size(14, 14).Radius(2)
	if checked {
		box.Background(p.ring.color())
		// The check picks the ink that contrasts with the accent.
		ink := ui.RGB(255, 255, 255)
		if p.isDark {
			ink = p.primaryFg.color()
		}
		box.DrawOver(func(pt *ui.Painter, r ui.Rect) {
			pt.StrokePath(webCheckPath(r), 2, ink)
		})
	} else if p.isDark {
		box.Background(ui.RGB(34, 35, 37)).Border(1, ui.RGB(67, 68, 70))
	} else {
		box.Background(ui.RGB(255, 255, 255)).Border(1, ui.RGB(118, 118, 118))
	}
}

// webCheckbox is that box as a stand-alone 14px control.
func webCheckbox(c *ui.Context, p palette, checked *bool) {
	cb := ui.CheckboxBase(c, checked).Width(14).Height(14)
	cb.Children(func() { webBox(c, p, *checked) })
}

// webCheckPath is mygo's checkPath, unexported there: the check stroke
// the theme's own check box draws.
func webCheckPath(r ui.Rect) *ui.Path {
	var path ui.Path
	return path.MoveTo(r.X+r.W*0.22, r.Y+r.H*0.52).
		LineTo(r.X+r.W*0.42, r.Y+r.H*0.71).
		LineTo(r.X+r.W*0.78, r.Y+r.H*0.31)
}

// fullURL joins the URL with enabled query params, as the web disclosure shows.
func fullURL(raw string, params []store.KV) string {
	if raw == "" {
		return ""
	}
	q := url.Values{}
	for _, kv := range params {
		if kv.Enabled && kv.Key != "" {
			q.Add(kv.Key, kv.Value)
		}
	}
	if len(q) == 0 {
		return raw
	}
	sep := "?"
	if strings.Contains(raw, "?") {
		sep = "&"
	}
	return raw + sep + q.Encode()
}

// curlFor renders the request as a curl command, as the web cURL button copies.
func curlFor(method, raw string, params []store.KV) string {
	full := fullURL(raw, params)
	if full == "" {
		full = "http://"
	}
	return fmt.Sprintf("curl -X %s '%s'", method, full)
}
