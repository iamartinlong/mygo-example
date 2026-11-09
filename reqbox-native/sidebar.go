package main

import (
	"fmt"

	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"reqbox-desktop/pkg/store"
)

// sideRow flattens the sidebar tree into list rows: collection headers,
// their request rows (indented on a rail), a note when a collection is
// empty, and the UNSORTED section, as the web sidebar shows them.
type sideRow struct {
	header bool
	note   bool
	indent bool
	title  string
	colID  string // header's collection; "" for UNSORTED
	req    store.Request
}

// pending holds actions the row builders noted, applied after the list
// builds (menus and edit commits fire during the build pass).
type pending struct {
	newRequest    bool
	newCollection bool
	deleteReq     string
	deleteCol     string
	deleteHist    string
	clearHist     bool
	duplicate     string
	rename        string
	renameTo      string
	rebuild       bool
}

func (a *app) rebuildRows() {
	needle := strings.ToLower(a.filter)
	match := func(r store.Request) bool {
		return needle == "" ||
			strings.Contains(strings.ToLower(r.Name), needle) ||
			strings.Contains(strings.ToLower(r.URL), needle)
	}
	byCol := map[string][]store.Request{}
	var unsorted []store.Request
	for _, r := range a.reqs {
		if !match(r) {
			continue
		}
		if r.CollectionID == nil {
			unsorted = append(unsorted, r)
		} else {
			byCol[*r.CollectionID] = append(byCol[*r.CollectionID], r)
		}
	}
	rows := make([]sideRow, 0, len(a.reqs)+len(a.cols)+1)
	for _, c := range a.cols {
		reqs := byCol[c.ID]
		if needle != "" && len(reqs) == 0 {
			continue
		}
		rows = append(rows, sideRow{header: true, title: c.Name, colID: c.ID})
		if a.collapsed[c.ID] {
			continue
		}
		if len(reqs) == 0 {
			rows = append(rows, sideRow{note: true, indent: true, title: tr("sidebar.empty")})
			continue
		}
		for _, r := range reqs {
			rows = append(rows, sideRow{req: r, indent: true})
		}
	}
	if needle == "" || len(unsorted) > 0 {
		rows = append(rows, sideRow{header: true, title: tr("sidebar.unsorted")})
		for _, r := range unsorted {
			rows = append(rows, sideRow{req: r})
		}
	}
	a.rows = rows
	a.selRow = -1
	for i, row := range rows {
		if !row.header && !row.note && row.req.ID == a.selectedID {
			a.selRow = i
		}
	}
}

func (a *app) refresh() {
	if a.st == nil {
		return
	}
	a.reqs, _ = a.st.ListRequests()
	a.cols, _ = a.st.ListCollections()
	// Like the web: open the first request so the app never starts blank.
	if a.selectedID == "" && len(a.reqs) > 0 {
		a.selectedID = a.reqs[0].ID
	}
	a.rebuildRows()
	a.hist, _ = a.st.ListHistory(a.histFilter, 100)
	a.histSel = -1
	a.loadEnvs()
}

func (a *app) applyPending() {
	p := &a.pending
	switch {
	case p.rebuild:
		// A fold toggled mid-render: redo the flattened rows now that
		// the build loop is done.
		a.pending = pending{}
		a.rebuildRows()
		return
	case p.newRequest:
		a.st.CreateRequest(store.Request{Name: tr("sidebar.newRequest"), Method: "GET"})
		a.toastOK(tr("toast.requestCreated"))
	case p.newCollection:
		// The web asks for the name in its prompt dialog rather than
		// creating with a default.
		a.promptCollection()
	case p.deleteReq != "":
		a.st.DeleteRequest(p.deleteReq)
		if a.selectedID == p.deleteReq {
			a.selectedID = ""
		}
		a.toastOK(tr("toast.requestDeleted"))
	case p.deleteCol != "":
		a.st.DeleteCollection(p.deleteCol)
		a.toastOK(tr("toast.collectionDeleted"))
	case p.deleteHist != "":
		a.st.DeleteHistory(p.deleteHist)
	case p.clearHist:
		a.st.ClearHistory()
		a.toastOK(tr("toast.historyCleared"))
	case p.duplicate != "":
		if dup, err := a.st.DuplicateRequest(p.duplicate); err == nil {
			a.toastOK(tr("toast.duplicated", "name", dup.Name))
		}
	case p.rename != "":
		if r, err := a.st.GetRequest(p.rename); err == nil {
			r.Name = p.renameTo
			a.st.UpdateRequest(r, false)
		}
	default:
		return
	}
	a.pending = pending{}
	a.refresh()
}

// sidebarPane mirrors the web sidebar: environment row, Requests/History
// pills, filter row, then either the request sections or the run history.
// The environment picker's menu arrives with B8.
func (a *app) sidebarPane(c *ui.Context, p palette) {
	ui.Column(c).Fill().Background(p.rail.color()).Children(func() {
		ui.Row(c).Height(48).Padding(12, 12, 8, 12).Gap(8).AlignItems(ui.Center).
			Children(func() { a.envRow(c, p) })
		ui.Row(c).Padding(0, 12, 0, 12).Children(func() {
			ui.Row(c).Grow(1).Gap(2).Padding(2, 2, 2, 2).Radius(8).
				Background(p.muted.color().Alpha(0.6)).
				AlignItems(ui.Center).Children(func() {
				a.pill(c, p, tr("sidebar.requests"), len(a.reqs), "requests")
				a.pill(c, p, tr("sidebar.history"), len(a.hist), "history")
			})
		})
		if a.st == nil {
			return
		}
		// The history tab replaces the requests filter row with its own
		// search, as the web's Eh fragment does.
		if a.sideTab == "history" {
			a.historyPane(c, p)
			a.footerRow(c, p)
			return
		}
		a.filterRow(c, p)
		// Plain rows in a scroll, not a ui.List: the list's selectable rows
		// consume the clicks of the controls inside them (batch 23), which
		// the tree's kebab menus and fold chevrons need. The scroll sits in
		// a growing column, the batch-20 pattern.
		ui.Column(c).Grow(1).FillWidth().Children(func() {
			ui.Scroll(c).FillHeight().FillWidth().Clip().
				TrackScroll(&a.treeScroll).Children(func() {
				ui.Column(c).FillWidth().Padding(4, 8, 12, 8).Children(func() {
					for i := range a.rows {
						a.sideRowView(c, p, i)
					}
					a.applyPending()
				})
			})
		})
		a.footerRow(c, p)
	})
}

// historyPane lists recorded runs, newest first, with its own search row
// and a trash to clear everything; clicking a row whose originating
// request is still on file re-opens that run in the workspace.
func (a *app) historyPane(c *ui.Context, p palette) {
	// The web's search row: a muted field with its own placeholder and the
	// clear-all trash beside it, replacing the requests filter row.
	ui.Row(c).Height(38).Padding(10, 12, 4, 12).Gap(6).
		AlignItems(ui.Center).Children(func() {
		ui.Row(c).Grow(1).Height(28).Radius(8).
			Background(p.muted.color().Alpha(0.5)).Padding(0, 8, 0, 8).Gap(6).
			AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconSearch).Size(12, 12).TextColor(p.mutedFg.color())
			if ui.TextInputBase(c, &a.histFilter).Grow(1).Height(28).
				FontSize(12).Placeholder(tr("sidebar.searchHistory")).Changed() {
				if a.st != nil {
					a.hist, _ = a.st.ListHistory(a.histFilter, 100)
					a.histSel = -1
				}
			}
		})
		del := ui.Row(c).Width(24).Height(24).Radius(6).
			AlignItems(ui.Center).Center()
		ink := p.mutedFg.color()
		if del.Hovered() {
			del.Background(p.muted.color())
			ink = p.foreground.color()
		}
		del.Children(func() {
			ui.Icon(c, iconTrash).Size(12, 12).TextColor(ink)
		})
		if del.Clicked() {
			a.askConfirm(tr("confirm.clearHistory.title"),
				tr("confirm.clearHistory.desc"),
				tr("confirm.clearHistory.confirm"), func() { a.pending.clearHist = true })
			a.applyPending()
		}
	})
	if len(a.hist) == 0 {
		ui.Column(c).Grow(1).Padding(32, 12, 0, 12).AlignItems(ui.Center).
			Children(func() {
				ui.Icon(c, iconHistory).Size(20, 20).
					TextColor(p.mutedFg.color().Alpha(0.5))
				ui.Row(c).Height(8)
				ui.Text(c, tr("sidebar.noHistory")).FontSize(12).
					TextColor(p.mutedFg.color())
			})
		return
	}
	// The rows go back on a ui.List inside a growing column — the
	// batch-20 nesting, which keeps the footer pinned without the list
	// bleeding past it. The web's click-the-trash delete becomes a row
	// context menu here — ui.List's selectable rows consume the clicks of
	// the controls inside them (batch 23), so only the row itself and its
	// menu receive them.
	a.histList.Selected = &a.histSel
	ui.Column(c).Grow(1).FillWidth().Children(func() {
		e := ui.List(c, &a.histList, len(a.hist),
			func(i int) { a.histRowView(c, p, i) }).Fill()
		e.ItemKey(func(i int) any { return a.hist[i].ID })
		if e.Changed() && a.histSel >= 0 {
			a.viewHistoryEntry(a.histSel)
		}
		a.applyPending()
	})
}

// histRowView is the web's history row: status, method and age on the
// first line, the name, then the duration and the request path, with the
// trash icon the web's hover reveals drawn in. Deleting goes through the
// row's context menu — ui.List's selectable rows consume the clicks of
// controls inside them (batch 23), so the web's click-the-trash has no
// native counterpart on the list.
func (a *app) histRowView(c *ui.Context, p palette, i int) {
	h := a.hist[i]
	row := ui.Row(c).Fill().Padding(6, 4, 6, 8).Gap(8).AlignItems(ui.Start).
		Radius(6)
	if row.Hovered() {
		row.Background(p.accent.color().Alpha(0.5))
	}
	row.Children(func() {
		ui.Column(c).Grow(1).Basis(0).MinWidth(0).Gap(1).Children(func() {
			ui.Row(c).FillWidth().Height(16).Gap(6).AlignItems(ui.Center).
				Children(func() {
					status := strconv.Itoa(h.Status)
					ink := p.bad.color()
					switch {
					case h.Status == 0:
						status = tr("sidebar.err")
						ink = p.mutedFg.color()
					case h.Status >= 200 && h.Status < 300:
						ink = p.ok.color()
					}
					ui.Text(c, status).FontSize(10).Bold().
						Font("JetBrains Mono").FontFeatures("tnum").
						SingleLine().TextColor(ink)
					ui.Text(c, h.Method).FontSize(10).Font("JetBrains Mono").
						SingleLine().TextColor(p.mutedFg.color())
					ui.Row(c).Grow(1)
					ui.Text(c, timeAgo(h.CreatedAt.Time)).FontSize(10).
						Font("JetBrains Mono").SingleLine().
						TextColor(p.mutedFg.color().Alpha(0.7))
				})
			name := h.Name
			if name == "" {
				name = h.URL
			}
			ui.Text(c, name).FontSize(12).
				TextColor(p.foreground.color().Alpha(0.85))
			ui.Text(c, fmt.Sprintf("%d ms · %s", h.DurationMs, displayURL(h.URL))).
				FontSize(10).Font("JetBrains Mono").
				TextColor(p.mutedFg.color().Alpha(0.7))
		})
		del := ui.Row(c).Width(24).Height(24).AlignItems(ui.Center).Center()
		del.Children(func() {
			// The web's group-hover reveal.
			if row.Hovered() {
				ui.Icon(c, iconTrash).Size(12, 12).TextColor(p.mutedFg.color())
			}
		})
	}).ContextMenu(func(m *ui.Menu) {
		if m.Item(tr("sidebar.deleteHistoryEntry")).Chosen() {
			a.pending.deleteHist = h.ID
		}
	})
}

// envRow lives in env.go with the rest of the environment UI.

// pill is one of the Requests/History tabs; the web draws the active one
// as a filled pill and the other as bare text with a count.
func (a *app) pill(c *ui.Context, p palette, label string, count int, tab string) {
	active := a.sideTab == tab
	row := ui.Row(c).Grow(1).Height(25).Radius(6).
		AlignItems(ui.Center).Justify(ui.Center)
	if active {
		row = row.Background(p.background.color()).
			Shadow(0, 1, 2, 0, p.foreground.color().Alpha(0.08))
	}
	e := row.Children(func() {
		t := ui.Text(c, label).FontSize(11.5)
		if !active {
			t.TextColor(p.mutedFg.color())
		}
		ui.Text(c, strconv.Itoa(count)).FontSize(10).Margin(0, 0, 0, 6).
			Font("JetBrains Mono").FontFeatures("tnum").
			TextColor(p.mutedFg.color())
	})
	if e.Clicked() {
		a.sideTab = tab
	}
}

func (a *app) filterRow(c *ui.Context, p palette) {
	ui.Row(c).Height(42).Padding(10, 12, 4, 12).Gap(6).
		AlignItems(ui.Center).Children(func() {
		ui.SearchField(c, &a.filter).Fill().Height(28).FontSize(12).
			Placeholder(tr("sidebar.filterRequests"))
		if a.filterChanged() {
			a.rebuildRows()
		}
		a.iconBtn(c, p, iconFolderPlus, &a.pending.newCollection)
		a.iconBtn(c, p, iconPlus, &a.pending.newRequest)
	})
}

// filterChanged reports whether the SearchField took new input this pass.
func (a *app) filterChanged() bool {
	if a.filter == a.lastFilter {
		return false
	}
	a.lastFilter = a.filter
	return true
}

func (a *app) iconBtn(c *ui.Context, p palette, icon *ui.SVG, clicked *bool) {
	e := ui.Row(c).Width(24).Height(24).AlignItems(ui.Center).Center().
		Children(func() {
			ui.Icon(c, icon).TextColor(p.mutedFg.color())
		})
	if e.Clicked() {
		*clicked = true
	}
}

func (a *app) sideRowView(c *ui.Context, p palette, i int) {
	row := a.rows[i]
	switch {
	case row.header && row.colID != "":
		a.colHeaderView(c, p, row)
	case row.header:
		// UNSORTED is the web's plain eyebrow line, not a foldable header.
		ui.Row(c).Fill().Padding(8, 8, 4, 8).Children(func() {
			ui.Text(c, strings.ToUpper(row.title)).FontSize(11).
				LetterSpacing(1.54).Font("JetBrains Mono").SingleLine().
				TextColor(p.mutedFg.color())
		})
	case row.note:
		n := ui.Row(c).Fill().Margin(0, 0, 0, 8).Padding(4, 4, 4, 14).
			BorderWidth(0, 0, 0, 1).BorderColor(p.border.color().Alpha(0.6))
		n.Children(func() {
			ui.Text(c, row.title).FontSize(11).SingleLine().
				TextColor(p.mutedFg.color().Alpha(0.7))
		})
	default:
		a.reqRowView(c, p, row)
	}
}

// colHeaderView is a collection line: chevron, folder, name and count,
// with a kebab menu on the right; a click folds or unfolds it.
func (a *app) colHeaderView(c *ui.Context, p palette, row sideRow) {
	folded := a.collapsed[row.colID]
	h := ui.Row(c).Fill().Padding(4, 4, 4, 4).Gap(4).AlignItems(ui.Center).Radius(6)
	if h.Hovered() {
		h.Background(p.accent.color().Alpha(0.4))
	}
	h.Children(func() {
		chev := iconChevronDown
		if folded {
			chev = iconChevronRt
		}
		ui.Icon(c, chev).Size(12, 12).TextColor(p.mutedFg.color())
		ui.Icon(c, iconFolder).Size(14, 14).TextColor(p.mutedFg.color())
		ui.Text(c, row.title).FontSize(12).FontWeight(500).SingleLine().
			TextColor(p.foreground.color().Alpha(0.85)).Grow(1).
			Basis(0).MinWidth(0)
		if n := len(a.reqsIn(row.colID)); n > 0 {
			ui.Text(c, fmt.Sprint(n)).FontSize(10).Font("JetBrains Mono").
				TextColor(p.mutedFg.color().Alpha(0.7))
		}
		a.kebab(c, p, h, "c:"+row.colID, func(panel ui.Element) {
			a.sideMenuItem(c, p, panel, iconPlus, tr("sidebar.newRequestHere"), false, func() {
				id := row.colID
				if _, err := a.st.CreateRequest(store.Request{
					Name: tr("sidebar.untitledRequest"), Method: "GET", CollectionID: &id,
				}); err == nil {
					a.toastOK(tr("toast.requestCreated"))
					a.refresh()
				}
			})
			a.sideMenuItem(c, p, panel, nil, tr("common.rename"), false, func() {
				id, current := row.colID, row.title
				a.askPrompt(tr("common.rename"), "", tr("prompt.newCollection.label"),
					current, "", tr("common.save"), func(v string) {
						name := strings.TrimSpace(v)
						if name == "" || a.st == nil {
							return
						}
						if _, err := a.st.UpdateCollection(id, &name, nil); err == nil {
							a.refresh()
						}
					})
			})
			a.sideMenuItem(c, p, panel, iconTrash, tr("sidebar.deleteCollection"), true, func() {
				id, title := row.colID, row.title
				a.askConfirm(tr("confirm.deleteCollection.title", "name", title),
					tr("confirm.deleteCollection.desc"),
					tr("confirm.deleteCollection.confirm"),
					func() { a.pending.deleteCol = id })
			})
		})
	})
	if h.Clicked() {
		if a.collapsed == nil {
			a.collapsed = map[string]bool{}
		}
		a.collapsed[row.colID] = !folded
		// The rows were flattened before the fold; redo them after the
		// loop, so its index stays valid.
		a.pending.rebuild = true
	}
}

// reqsIn counts the requests a collection holds, for its header count.
func (a *app) reqsIn(colID string) []store.Request {
	var out []store.Request
	for _, r := range a.reqs {
		if r.CollectionID != nil && *r.CollectionID == colID {
			out = append(out, r)
		}
	}
	return out
}

// reqRowView is a request line: the method tag, name and path, the
// published dot, and a kebab menu revealed on hover.
func (a *app) reqRowView(c *ui.Context, p palette, row sideRow) {
	r := row.req
	e := ui.Row(c).Fill().Padding(6, 4, 6, 8).Gap(8).AlignItems(ui.Center).
		Radius(6)
	if row.indent {
		e = e.Margin(0, 0, 0, 8).BorderWidth(0, 0, 0, 1).
			BorderColor(p.border.color().Alpha(0.6)).Padding(6, 4, 6, 14)
	}
	if r.ID == a.selectedID {
		e.Background(p.accent.color())
	} else if e.Hovered() {
		e.Background(p.accent.color().Alpha(0.6))
	}
	e.Children(func() {
		m := p.method(r.Method)
		tag := strings.ToUpper(strings.TrimSpace(r.Method))
		if len(tag) > 4 {
			tag = tag[:4]
		}
		ui.Row(c).Padding(1, 4, 1, 4).Radius(4).
			Background(m.color().Alpha(0.1)).Children(func() {
			ui.Text(c, tag).FontSize(9.5).Bold().LetterSpacing(0.24).
				Font("JetBrains Mono").SingleLine().TextColor(m.color())
		})
		ui.Column(c).Grow(1).Basis(0).MinWidth(0).Children(func() {
			ui.Text(c, r.Name).FontSize(12.5).SingleLine().
				TextColor(p.foreground.color().Alpha(0.9))
			ui.Text(c, displayURL(r.URL)).FontSize(10.5).Font("JetBrains Mono").
				SingleLine().TextColor(p.mutedFg.color().Alpha(0.8))
		})
		if r.Exposed {
			ui.Row(c).Width(6).Height(6).Radius(3).
				Background(p.ok.color())
		}
		a.kebab(c, p, e, "r:"+r.ID, func(panel ui.Element) {
			a.sideMenuItem(c, p, panel, iconCopy, tr("sidebar.duplicate"), false, func() {
				a.pending.duplicate = r.ID
			})
			on := !r.Exposed
			label := tr("sidebar.publishMock")
			if r.Exposed {
				label = tr("sidebar.unpublishMock")
			}
			a.sideMenuItem(c, p, panel, iconGlobe, label, false, func() {
				rec := r
				rec.Exposed = on
				if on && rec.ExposeSlug == "" {
					rec.ExposeSlug = a.suggestSlug(rec.Name)
				}
				if _, err := a.st.UpdateRequest(rec, false); err == nil {
					if on {
						a.toastOK(tr("toast.mockPublished"))
					} else {
						a.toastOK(tr("toast.mockUnpublished"))
					}
					a.refresh()
				}
			})
			a.sideMenuItem(c, p, panel, iconTrash, tr("common.delete"), true, func() {
				a.askConfirm(tr("confirm.deleteRequest.title", "name", r.Name),
					tr("confirm.deleteRequest.desc"),
					tr("confirm.deleteRequest.confirm"),
					func() { a.pending.deleteReq = r.ID })
			})
		})
	})
	if e.Clicked() {
		a.selectedID = r.ID
	}
}

// kebab is the row's three-dot button, shown while the row hovers or its
// menu is open; build fills the menu.
func (a *app) kebab(c *ui.Context, p palette, row ui.Element, id string, build func(panel ui.Element)) {
	openNow := a.sideMenuOn && a.sideMenuID == id
	kb := ui.Row(c).Width(24).Height(24).Radius(6).
		AlignItems(ui.Center).Center()
	if kb.Hovered() {
		kb.Background(p.muted.color())
	}
	kb.Children(func() {
		if row.Hovered() || openNow {
			ui.Icon(c, iconEllipsis).Size(14, 14).TextColor(p.mutedFg.color())
		}
	})
	if kb.Clicked() {
		a.sideMenuOn = true
		a.sideMenuID = id
	}
	// Recompute after the click: the press may have just opened it, and
	// the dismiss sync below must not undo that in the same frame.
	open := a.sideMenuOn && a.sideMenuID == id
	ui.PopoverBase(c, kb, &open, func(panel ui.Element) {
		panel.MinWidth(176).Margin(4, 0, 0, 0).Padding(4).
			Background(p.card.color()).Radius(6).Border(1, p.border.color())
		build(panel)
	})
	if !open && a.sideMenuOn && a.sideMenuID == id {
		a.sideMenuOn = false
	}
}

// sideMenuItem is one line of a kebab menu; choosing it closes the menu
// and runs fn.
func (a *app) sideMenuItem(c *ui.Context, p palette, panel ui.Element, icon *ui.SVG, label string, destructive bool, fn func()) {
	it := ui.Row(c).FillWidth().Height(28).Padding(0, 8, 0, 8).Gap(6).
		AlignItems(ui.Center).Radius(4)
	ink := p.foreground.color()
	if destructive {
		ink = p.bad.color()
	}
	if it.Hovered() {
		it.Background(p.accent.color())
	}
	it.Children(func() {
		if icon != nil {
			ui.Icon(c, icon).Size(12, 12).TextColor(ink)
		}
		ui.Text(c, label).FontSize(12).SingleLine().TextColor(ink)
	})
	if it.Clicked() {
		a.sideMenuOn = false
		if fn != nil {
			fn()
		}
	}
}

// displayURL is the web's Sh: the URL's path, its host when the path is
// the root, or a note when there is nothing to show.
func displayURL(raw string) string {
	t := strings.TrimSpace(raw)
	if t == "" {
		return "no URL"
	}
	if !strings.Contains(t, "://") {
		t = "http://" + t
	}
	rest := t
	if i := strings.Index(t, "://"); i >= 0 {
		rest = t[i+3:]
	}
	if j := strings.Index(rest, "/"); j >= 0 {
		if path := rest[j:]; path != "/" {
			return path
		}
		return rest[:j]
	}
	return rest
}

// timeAgo is the web's relative age, worded for the shown language and
// falling back to the date after a month.
func timeAgo(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	zh := appLang == "zh"
	sec := int(d.Seconds())
	if sec < 0 {
		sec = 0
	}
	switch {
	case sec < 60:
		if zh {
			return "刚刚"
		}
		return "just now"
	case sec < 3600:
		n := strconv.Itoa(sec / 60)
		if zh {
			return n + " 分钟前"
		}
		return n + "m ago"
	case sec < 86400:
		n := strconv.Itoa(sec / 3600)
		if zh {
			return n + " 小时前"
		}
		return n + "h ago"
	case sec < 30*86400:
		n := strconv.Itoa(sec / 86400)
		if zh {
			return n + " 天前"
		}
		return n + "d ago"
	default:
		return t.Format("2006/01/02")
	}
}
