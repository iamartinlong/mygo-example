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
	ui.Column(c).Fill().Background(p.card.color()).Children(func() {
		if a.edit.id == "" {
			a.emptyState(c, p)
			return
		}
		a.titleRow(c, p)
		a.requestBar(c, p)
		a.fullURLRow(c, p)
		a.curlRow(c, p)
		a.tabStrip(c, p)
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
		a.runBar(c, p)
		a.responsePane(c, p)
	})
}

func (a *app) emptyState(c *ui.Context, p palette) {
	ui.Column(c).Fill().AlignItems(ui.Center).Children(func() {
		ui.Row(c).Grow(1)
		ui.Text(c, "NO REQUEST").FontSize(11).LetterSpacing(1).
			SingleLine().TextColor(p.mutedFg.color())
		ui.Text(c, "Pick a request on the left, or make one with +").
			FontSize(12.5).SingleLine().TextColor(p.mutedFg.color())
		ui.Row(c).Grow(1)
	})
}

func (a *app) titleRow(c *ui.Context, p palette) {
	status := "saved"
	if a.edit.dirty {
		status = "unsaved"
	}
	ui.Row(c).Height(44).Padding(0, 16, 0, 16).Gap(10).
		AlignItems(ui.Center).Children(func() {
		if ui.EditableText(c, &a.edit.name).FontSize(13.5).Bold().
			SingleLine().Changed() {
			a.edit.dirty = true
		}
		ui.Text(c, status).FontSize(11.5).SingleLine().
			TextColor(p.mutedFg.color()).Grow(1)
		a.saveBtn(c, p)
	})
}

func (a *app) saveBtn(c *ui.Context, p palette) {
	e := ui.Row(c).Height(28).Padding(0, 14, 0, 14).Gap(6).
		AlignItems(ui.Center).Background(p.foreground.color()).Children(func() {
		ui.Text(c, "Save").FontSize(12).SingleLine().TextColor(p.primaryFg.color())
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
	r.Name, r.Method, r.URL = a.edit.name, a.edit.method, a.edit.url
	r.Params, r.Headers = a.edit.params, a.edit.headers
	r.BodyType, r.Body, r.Form = a.edit.bodyType, a.bodyBuf.String(), a.edit.form
	r.AuthType, r.Auth = a.edit.authType, a.edit.auth
	r.Exposed, r.ExposeSlug = a.edit.exposed, a.edit.exposeSlug
	if _, err := a.st.UpdateRequest(r, false); err == nil {
		a.bodyBuf.MarkSaved()
		a.edit.dirty = false
		a.refresh()
	}
}

func (a *app) requestBar(c *ui.Context, p palette) {
	ui.Row(c).Height(40).Padding(0, 16, 0, 16).Gap(8).
		AlignItems(ui.Center).Children(func() {
		if ui.Select(c, &a.edit.method, methods).Width(112).Height(30).Changed() {
			a.edit.dirty = true
		}
		if ui.TextInput(c, &a.edit.url).Placeholder("http://localhost:8080/path").
			Grow(1).Height(30).FontSize(12.5).Changed() {
			a.edit.dirty = true
		}
		e := ui.Row(c).Width(28).Height(28).AlignItems(ui.Center).Center().
			Children(func() {
				ui.Icon(c, iconCopy).Size(13, 13).TextColor(p.mutedFg.color())
			})
		if e.Clicked() {
			mygo.Clipboard.WriteText(curlFor(a.edit.method, a.edit.url, a.edit.params))
		}
		a.sendBtn(c, p)
	})
}

func (a *app) sendBtn(c *ui.Context, p palette) {
	e := ui.Row(c).Height(30).Padding(0, 16, 0, 16).Gap(6).
		AlignItems(ui.Center).Background(p.primary.color()).Children(func() {
		if a.run.running {
			ui.Text(c, "…").FontSize(11.5).Bold().
				SingleLine().TextColor(p.primaryFg.color())
		} else {
			ui.Icon(c, iconSend).Size(13, 13).TextColor(p.primaryFg.color())
			ui.Text(c, "SEND").FontSize(11.5).Bold().LetterSpacing(0.5).
				SingleLine().TextColor(p.primaryFg.color())
		}
	})
	if e.Clicked() {
		a.send()
	}
}

func (a *app) fullURLRow(c *ui.Context, p palette) {
	ui.Row(c).Height(26).Padding(0, 16, 0, 16).Gap(6).
		AlignItems(ui.Center).Children(func() {
		e := ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconChevronRt).Size(11, 11).TextColor(p.mutedFg.color())
			ui.Text(c, "full URL").FontSize(11).SingleLine().
				TextColor(p.mutedFg.color())
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

// curlRow holds the cURL copy and Delete actions plus the collection label.
func (a *app) curlRow(c *ui.Context, p palette) {
	label := "Unsorted"
	for _, col := range a.cols {
		if col.ID == a.edit.colID {
			label = col.Name
		}
	}
	ui.Row(c).Height(28).Padding(0, 16, 0, 16).Gap(12).
		AlignItems(ui.Center).Children(func() {
		e := ui.Row(c).Children(func() {
			ui.Text(c, "cURL").FontSize(11.5).Font("JetBrains Mono").
				SingleLine().TextColor(p.foreground.color())
		})
		if e.Clicked() {
			mygo.Clipboard.WriteText(curlFor(a.edit.method, a.edit.url, a.edit.params))
		}
		e = ui.Row(c).Children(func() {
			ui.Text(c, "Delete").FontSize(11.5).SingleLine().TextColor(p.bad.color())
		})
		if e.Clicked() {
			a.askConfirm(`Delete "`+a.edit.name+`"`,
				"Past runs of this request stay in History.",
				"Delete request", func() { a.pending.deleteReq = a.edit.id })
		}
		ui.Row(c).Grow(1).Children(func() {
			ui.Text(c, label).FontSize(11).SingleLine().TextColor(p.mutedFg.color())
		})
	})
}

func (a *app) tabStrip(c *ui.Context, p palette) {
	ui.Row(c).Height(32).Padding(0, 16, 0, 16).Gap(16).
		AlignItems(ui.Center).Children(func() {
		for _, tab := range []string{"Params", "Headers", "Body", "Auth", "Publish"} {
			t := tab
			active := a.edit.tab == t
			row := ui.Row(c).Height(30).AlignItems(ui.Center)
			if active {
				row = row.Background(p.card.color())
			}
			e := row.Children(func() {
				ui.Text(c, t).FontSize(12).SingleLine()
			})
			if e.Clicked() {
				a.edit.tab = t
			}
		}
	})
}

// kvEditor edits one KV list: column eyebrows, then rows of
// checkbox + key + value + delete.
func (a *app) kvEditor(c *ui.Context, p palette, what string, kvs *[]store.KV) {
	ui.Row(c).Height(30).Padding(0, 16, 0, 16).Gap(8).
		AlignItems(ui.Center).Children(func() {
		ui.Row(c).Width(24)
		ui.Text(c, strings.ToUpper(what)).FontSize(10).LetterSpacing(0.8).
			SingleLine().TextColor(p.mutedFg.color()).Grow(2)
		ui.Text(c, "VALUE").FontSize(10).LetterSpacing(0.8).
			SingleLine().TextColor(p.mutedFg.color()).Grow(3)
		ui.Row(c).Width(24)
	})
	for i := range *kvs {
		i := i
		ui.Row(c).Height(32).Padding(0, 16, 0, 16).Gap(8).
			AlignItems(ui.Center).Children(func() {
			if ui.Checkbox(c, &(*kvs)[i].Enabled, "").Changed() {
				a.edit.dirty = true
			}
			if ui.TextInput(c, &(*kvs)[i].Key).Placeholder(what).
				Grow(2).Height(28).FontSize(12).Changed() {
				a.edit.dirty = true
			}
			if ui.TextInput(c, &(*kvs)[i].Value).Grow(3).Height(28).
				FontSize(12).Changed() {
				a.edit.dirty = true
			}
			e := ui.Row(c).Width(24).Height(24).AlignItems(ui.Center).Center().
				Children(func() {
					ui.Icon(c, iconX).Size(12, 12).TextColor(p.mutedFg.color())
				})
			if e.Clicked() {
				a.edit.dirty = true
				*kvs = append((*kvs)[:i], (*kvs)[i+1:]...)
			}
		})
	}
	ui.Row(c).Height(44).Padding(0, 16, 0, 16).Children(func() {
		e := ui.Row(c).Fill().Height(36).AlignItems(ui.Center).Center().
			Background(p.background.color()).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, iconPlus).Size(12, 12).TextColor(p.mutedFg.color())
				ui.Text(c, "Add a "+what).FontSize(12).SingleLine().
					TextColor(p.mutedFg.color())
			})
		})
		if e.Clicked() {
			a.edit.dirty = true
			*kvs = append(*kvs, store.KV{ID: kvID(), Enabled: true})
		}
	})
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
