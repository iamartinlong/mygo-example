package main

import (
	"strconv"
	"strings"

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
		add("", "UNSORTED", unsorted)
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
	a.rebuildRows()
}

func (a *app) applyPending() {
	p := &a.pending
	switch {
	case p.newRequest:
		a.st.CreateRequest(store.Request{Name: "New request", Method: "GET"})
	case p.newCollection:
		a.st.CreateCollection("New collection")
	case p.deleteReq != "":
		a.st.DeleteRequest(p.deleteReq)
		if a.selectedID == p.deleteReq {
			a.selectedID = ""
		}
	case p.deleteCol != "":
		a.st.DeleteCollection(p.deleteCol)
	case p.duplicate != "":
		a.st.DuplicateRequest(p.duplicate)
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
// pills, filter row, then the sections of requests. History is B6, the
// environment picker B8.
func (a *app) sidebarPane(c *ui.Context, p palette) {
	ui.Column(c).Fill().Background(p.panel.color()).Children(func() {
		a.envRow(c, p)
		ui.Row(c).Padding(8, 12, 8, 12).Gap(8).AlignItems(ui.Center).Children(func() {
			a.pill(c, p, "Requests", len(a.reqs), true)
			a.pill(c, p, "History", 0, false)
		})
		a.filterRow(c, p)
		if a.st == nil {
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
	})
}

// envRow is the environment picker; its menu arrives with B8.
func (a *app) envRow(c *ui.Context, p palette) {
	ui.Row(c).Height(36).Padding(0, 12, 0, 12).Gap(8).
		AlignItems(ui.Center).Children(func() {
		ui.Text(c, "No environment").FontSize(12).Grow(1)
		ui.Icon(c, iconChevronDown).TextColor(p.mutedFg.color())
	})
}

// pill is one of the Requests/History tabs; the web draws the active one
// as a filled pill and the other as bare text with a count.
func (a *app) pill(c *ui.Context, p palette, label string, count int, active bool) {
	row := ui.Row(c).Height(26).Padding(0, 12, 0, 12).Gap(6).
		AlignItems(ui.Center)
	if active {
		row = row.Background(p.card.color())
	}
	row.Children(func() {
		ui.Text(c, label).FontSize(12)
		ui.Text(c, strconv.Itoa(count)).FontSize(11).TextColor(p.mutedFg.color())
	})
}

func (a *app) filterRow(c *ui.Context, p palette) {
	ui.Row(c).Height(30).Padding(0, 12, 8, 12).Gap(6).
		AlignItems(ui.Center).Children(func() {
		ui.SearchField(c, &a.filter).Fill().Height(26).FontSize(12)
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
	if row.header {
		ui.Row(c).Fill().Padding(12, 12, 6, 12).Children(func() {
			ui.Text(c, row.title).FontSize(10).LetterSpacing(0.8).
				SingleLine().TextColor(p.mutedFg.color())
		}).ContextMenu(func(m *ui.Menu) {
			if row.colID != "" && m.Item("Delete").Chosen() {
				a.pending.deleteCol = row.colID
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
			if ui.EditableText(c, &a.rows[i].edit).FontSize(12.5).
				SingleLine().Changed() {
				a.pending.rename = r.ID
				a.pending.renameTo = a.rows[i].edit
			}
			ui.Text(c, displayURL(r.URL)).FontSize(11).Font("JetBrains Mono").
				SingleLine().TextColor(p.mutedFg.color())
		})
	}).ContextMenu(func(m *ui.Menu) {
		if m.Item("Duplicate").Chosen() {
			a.pending.duplicate = r.ID
		}
		m.Separator()
		if m.Item("Delete").Chosen() {
			a.pending.deleteReq = r.ID
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
