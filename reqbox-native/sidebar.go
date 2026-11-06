package main

import (
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"reqbox-desktop/pkg/store"
)

// sideRow flattens the sidebar into list rows: section headers and request
// rows, sections following the collections' order with unfiled requests
// under UNSORTED, as the web sidebar does.
type sideRow struct {
	header bool
	title  string
	colID  string // header's collection; "" for UNSORTED
	req    store.Request
	edit   string // rename buffer for EditableText
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
	add := func(colID, title string, reqs []store.Request) {
		rows = append(rows, sideRow{header: true, title: title, colID: colID})
		for _, r := range reqs {
			rows = append(rows, sideRow{req: r, edit: r.Name})
		}
	}
	for _, c := range a.cols {
		if reqs := byCol[c.ID]; needle == "" || len(reqs) > 0 {
			add(c.ID, c.Name, reqs)
		}
	}
	if needle == "" || len(unsorted) > 0 {
		add("", tr("sidebar.unsorted"), unsorted)
	}
	a.rows = rows
	a.selRow = -1
	for i, row := range rows {
		if !row.header && row.req.ID == a.selectedID {
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
	a.hist, _ = a.st.ListHistory(a.filter, 100)
	a.histSel = -1
	a.loadEnvs()
}

func (a *app) applyPending() {
	p := &a.pending
	switch {
	case p.newRequest:
		a.st.CreateRequest(store.Request{Name: tr("sidebar.newRequest"), Method: "GET"})
		a.toastOK(tr("toast.requestCreated"))
	case p.newCollection:
		a.st.CreateCollection(tr("sidebar.newCollection"))
		a.toastOK(tr("toast.collectionCreated"))
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
		ui.Row(c).Height(48).Padding(12, 12, 8, 12).AlignItems(ui.Center).
			Children(func() { a.envRow(c, p) })
		ui.Row(c).Padding(0, 12, 0, 12).Children(func() {
			ui.Row(c).Grow(1).Gap(2).Padding(2, 2, 2, 2).Radius(8).
				Background(p.muted.color().Alpha(0.6)).
				AlignItems(ui.Center).Children(func() {
				a.pill(c, p, tr("sidebar.requests"), len(a.reqs), "requests")
				a.pill(c, p, tr("sidebar.history"), len(a.hist), "history")
			})
		})
		a.filterRow(c, p)
		if a.st == nil {
			return
		}
		if a.sideTab == "history" {
			a.historyPane(c, p)
			a.footerRow(c, p)
			return
		}
		a.list.Selected = &a.selRow
		e := ui.List(c, &a.list, len(a.rows),
			func(i int) { a.sideRowView(c, p, i) })
		e.ItemKey(func(i int) any {
			if a.rows[i].header {
				return "h:" + a.rows[i].title
			}
			return a.rows[i].req.ID
		})
		if e.Changed() && a.selRow >= 0 && !a.rows[a.selRow].header {
			a.selectedID = a.rows[a.selRow].req.ID
		}
		a.applyPending()
		a.footerRow(c, p)
	})
}

// historyPane lists recorded runs, newest first; clicking one with its
// originating request still on file re-opens that run in the workspace.
func (a *app) historyPane(c *ui.Context, p palette) {
	if len(a.hist) == 0 {
		ui.Column(c).Grow(1).Padding(12, 12, 0, 12).Children(func() {
			ui.Text(c, tr("sidebar.noHistory")).FontSize(11.5).TextColor(p.mutedFg.color())
		})
		return
	}
	a.histList.Selected = &a.histSel
	e := ui.List(c, &a.histList, len(a.hist),
		func(i int) { a.histRowView(c, p, i) })
	e.ItemKey(func(i int) any { return a.hist[i].ID })
	if e.Changed() && a.histSel >= 0 {
		a.viewHistoryEntry(a.histSel)
	}
	a.applyPending()
	ui.Row(c).Height(34).Padding(0, 12, 0, 12).AlignItems(ui.Center).
		Children(func() {
			e := ui.Text(c, tr("sidebar.clearAll")).FontSize(11.5).SingleLine().
				TextColor(p.bad.color())
			if e.Clicked() {
				a.askConfirm(tr("confirm.clearHistory.title"),
					tr("confirm.clearHistory.desc"),
					tr("confirm.clearHistory.confirm"), func() { a.pending.clearHist = true })
				a.applyPending()
			}
		})
}

func (a *app) histRowView(c *ui.Context, p palette, i int) {
	h := a.hist[i]
	row := ui.Row(c).Fill().Padding(6, 10, 6, 10).Gap(8).
		AlignItems(ui.Center)
	if h.ID == a.histOpenID {
		row = row.Background(p.accent.color())
	}
	row.Children(func() {
		m := p.method(h.Method)
		ui.Row(c).Height(18).Padding(0, 6, 0, 6).AlignItems(ui.Center).
			Background(m.color().Alpha(0.1)).Children(func() {
			ui.Text(c, h.Method).FontSize(9).Font("JetBrains Mono").
				TextColor(m.color())
		})
		ui.Column(c).Grow(1).Gap(1).Children(func() {
			ui.Text(c, h.Name).FontSize(12.5).SingleLine()
			ui.Text(c, displayURL(h.URL)).FontSize(11).Font("JetBrains Mono").
				SingleLine().TextColor(p.mutedFg.color())
		})
		ui.Column(c).Gap(1).AlignItems(ui.End).Children(func() {
			if h.OK {
				ui.Text(c, strconv.Itoa(h.Status)).FontSize(10.5).
					Font("JetBrains Mono").FontFeatures("tnum").
					SingleLine().TextColor(p.ok.color())
			} else {
				ui.Text(c, tr("common.failed")).FontSize(10.5).SingleLine().
					TextColor(p.bad.color())
			}
			ui.Text(c, timeAgo(h.CreatedAt.Time)).FontSize(10.5).SingleLine().
				TextColor(p.mutedFg.color())
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
			if a.sideTab == "history" && a.st != nil {
				a.hist, _ = a.st.ListHistory(a.filter, 100)
			}
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
	if row.header {
		ui.Row(c).Fill().Padding(12, 12, 6, 12).Children(func() {
			ui.Text(c, row.title).FontSize(10).LetterSpacing(0.8).
				SingleLine().TextColor(p.mutedFg.color())
		}).ContextMenu(func(m *ui.Menu) {
			if row.colID != "" && m.Item(tr("common.delete")).Chosen() {
				a.askConfirm(tr("confirm.deleteCollection.title", "name", row.title),
					tr("confirm.deleteCollection.desc"),
					tr("confirm.deleteCollection.confirm"),
					func() { a.pending.deleteCol = row.colID })
			}
		})
		return
	}
	r := row.req
	reqRow := ui.Row(c).Fill().Padding(6, 10, 6, 10).Gap(8).
		AlignItems(ui.Center)
	if r.ID == a.selectedID {
		reqRow = reqRow.Background(p.accent.color())
	}
	reqRow.Children(func() {
		m := p.method(r.Method)
		ui.Row(c).Height(18).Padding(0, 6, 0, 6).AlignItems(ui.Center).
			Background(m.color().Alpha(0.1)).Children(func() {
			ui.Text(c, r.Method).FontSize(9).Font("JetBrains Mono").
				TextColor(m.color())
		})
		ui.Column(c).Grow(1).Gap(1).Children(func() {
			// The web keeps the name text-foreground on the accent row.
			t := ui.EditableText(c, &a.rows[i].edit).FontSize(12.5).
				TextColor(p.foreground.color()).SingleLine()
			if t.Changed() {
				a.pending.rename = r.ID
				a.pending.renameTo = a.rows[i].edit
			}
			ui.Text(c, displayURL(r.URL)).FontSize(11).Font("JetBrains Mono").
				SingleLine().TextColor(p.mutedFg.color())
		})
	}).ContextMenu(func(m *ui.Menu) {
		if m.Item(tr("sidebar.duplicate")).Chosen() {
			a.pending.duplicate = r.ID
		}
		m.Separator()
		if m.Item(tr("common.delete")).Chosen() {
			a.askConfirm(tr("confirm.deleteRequest.title", "name", r.Name),
				tr("confirm.deleteRequest.desc"),
				tr("confirm.deleteRequest.confirm"), func() { a.pending.deleteReq = r.ID })
		}
	})
}

// displayURL strips the scheme and host, as the web subtitle shows the path.
func displayURL(raw string) string {
	if i := strings.Index(raw, "://"); i >= 0 {
		rest := raw[i+3:]
		if j := strings.Index(rest, "/"); j >= 0 {
			return rest[j:]
		}
		return "/"
	}
	return raw
}

func timeAgo(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return strconv.Itoa(int(d.Minutes())) + "m ago"
	case d < 24*time.Hour:
		return strconv.Itoa(int(d.Hours())) + "h ago"
	case d < 7*24*time.Hour:
		return strconv.Itoa(int(d.Hours()/24)) + "d ago"
	default:
		return t.Format("2006-01-02")
	}
}
