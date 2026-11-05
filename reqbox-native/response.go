package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"reqbox-desktop/pkg/runner"
	"reqbox-desktop/pkg/store"
)

// runState holds the SEND pipeline: settings for the next run and the
// outcome of the last one.
type runState struct {
	timeout string // seconds, as typed
	follow  bool
	verify  bool

	running bool
	resp    *runner.Response
	errText string
	bodyBuf *editor.Buffer
	bodySt  *editor.State
}

func (a *app) send() {
	if a.run.running || a.exec == nil || a.edit.id == "" {
		return
	}
	req := a.editRequest()
	name := a.edit.name
	opts := a.runOpts()
	a.run.running = true
	a.run.errText = ""
	go func() {
		resp, err := a.exec.Run(context.Background(), req, nil, opts)
		errText := ""
		if err != nil {
			errText = err.Error()
		}
		a.recordHistory(req, name, resp, errText)
		a.win.Update(func() {
			a.run.running = false
			if err != nil {
				a.run.resp = nil
				a.run.errText = errText
			} else {
				a.run.errText = ""
				a.run.resp = &resp
				a.resetRespBuf()
			}
			a.refresh()
		})
	}()
}

// recordHistory persists the run so the History tab can replay it.
func (a *app) recordHistory(req store.Request, name string, resp runner.Response, errText string) {
	if a.st == nil {
		return
	}
	h := store.HistoryEntry{
		RequestID:       req.ID,
		Name:            name,
		Method:          req.Method,
		URL:             req.URL,
		Error:           errText,
		OK:              errText == "" && resp.Status >= 200 && resp.Status < 400,
		RequestSnapshot: store.Execution{Request: req}.MarshalSnapshot(),
	}
	if errText == "" {
		h.Status = resp.Status
		h.DurationMs = resp.DurationMs
		h.SizeBytes = resp.SizeBytes
		h.ResponseBody = resp.BodyText
		h.BodyEncoding = resp.Encoding
		if b, err := json.Marshal(resp.Headers); err == nil {
			h.ResponseHeaders = string(b)
		}
		if resp.Timing != nil {
			if b, err := json.Marshal(resp.Timing); err == nil {
				h.Timing = string(b)
			}
		}
	}
	a.st.InsertHistory(h)
}

// viewHistoryEntry re-opens a recorded run: selecting the originating
// request when it still exists and showing its recorded response. Entries
// whose request was deleted stay viewable in the web version only.
func (a *app) viewHistoryEntry(i int) {
	e := a.hist[i]
	a.histOpenID = e.ID
	exec, ok := store.ParseSnapshot(e.RequestSnapshot)
	if !ok || a.st == nil {
		return
	}
	if _, err := a.st.GetRequest(exec.Request.ID); err != nil {
		return
	}
	a.selectedID = exec.Request.ID
	a.loadEdit()
	a.run.errText = e.Error
	a.run.resp = historyResponse(e)
	a.resetRespBuf()
}

// historyResponse rebuilds a runner.Response from a history row.
func historyResponse(e store.HistoryEntry) *runner.Response {
	r := runner.Response{
		Status:     e.Status,
		BodyText:   e.ResponseBody,
		Encoding:   e.BodyEncoding,
		DurationMs: e.DurationMs,
		SizeBytes:  e.SizeBytes,
		RequestURL: e.URL,
	}
	if e.ResponseHeaders != "" {
		json.Unmarshal([]byte(e.ResponseHeaders), &r.Headers)
	}
	if e.Timing != "" {
		t := runner.Timing{}
		if json.Unmarshal([]byte(e.Timing), &t) == nil {
			r.Timing = &t
		}
	}
	if vs := r.Headers["Content-Type"]; len(vs) > 0 {
		r.ContentType = vs[0]
	}
	return &r
}

// editRequest snapshots the workspace into a request for the runner.
func (a *app) editRequest() store.Request {
	return store.Request{
		ID:       a.edit.id,
		Method:   a.edit.method,
		URL:      a.edit.url,
		Params:   a.edit.params,
		Headers:  a.edit.headers,
		BodyType: a.edit.bodyType,
		Body:     a.bodyBuf.String(),
		Form:     a.edit.form,
		AuthType: a.edit.authType,
		Auth:     a.edit.auth,
	}
}

func (a *app) runOpts() runner.Opts {
	o := runner.DefaultOpts()
	if s, err := strconv.ParseFloat(strings.TrimSpace(a.run.timeout), 64); err == nil && s > 0 {
		o.Timeout = time.Duration(s * float64(time.Second))
	}
	o.FollowRedirects = a.run.follow
	o.VerifyTLS = a.run.verify
	return o
}

// resetRespBuf rebinds the response body preview to the latest response.
func (a *app) resetRespBuf() {
	if a.run.resp == nil {
		return
	}
	a.run.bodyBuf = editor.NewBuffer(a.run.resp.BodyText)
	ct := a.run.resp.ContentType
	switch {
	case strings.Contains(ct, "json"):
		a.run.bodyBuf.SetLanguage(editor.LanguageFor("x.json"))
	case strings.Contains(ct, "html") || strings.Contains(ct, "xml"):
		a.run.bodyBuf.SetLanguage(editor.LanguageFor("x.html"))
	default:
		a.run.bodyBuf.SetLanguage(editor.LanguageFor("plain.txt"))
	}
	a.run.bodySt = &editor.State{Buffer: a.run.bodyBuf, SoftWrap: true}
}

// runBar mirrors the web strip between the tabs and the response: timeout,
// redirects, TLS, and the "GET · none" summary on the right.
func (a *app) runBar(c *ui.Context, p palette) {
	ui.Row(c).Height(44).Padding(0, 16, 0, 16).Gap(10).
		AlignItems(ui.Center).Children(func() {
		ui.Text(c, "Timeout").FontSize(12).SingleLine().
			TextColor(p.mutedFg.color())
		ui.TextInput(c, &a.run.timeout).Width(56).Height(28).FontSize(12)
		ui.Text(c, "s").FontSize(12).SingleLine().TextColor(p.mutedFg.color())
		ui.Checkbox(c, &a.run.follow, "Follow redirects")
		ui.Checkbox(c, &a.run.verify, "Verify TLS certificates")
		ui.Row(c).Grow(1)
		ui.Text(c, a.edit.method+" · "+a.edit.authType).FontSize(10.5).
			Font("JetBrains Mono").SingleLine().TextColor(p.mutedFg.color())
	})
}

// responsePane is everything below the run bar: empty state, wait, failure,
// or the response itself.
func (a *app) responsePane(c *ui.Context, p palette) {
	ui.Column(c).FillHeight().Background(p.background.color()).
		Children(func() {
		switch {
		case a.run.running:
			a.respCenter(c, p, "WAITING", "waiting for response…")
		case a.run.errText != "":
			a.respFailed(c, p)
		case a.run.resp == nil:
			a.respCenter(c, p, "NO RESPONSE",
				"Send the request to see status, headers and body here.")
		default:
			a.respBody(c, p)
		}
	})
}

func (a *app) respCenter(c *ui.Context, p palette, eyebrow, msg string) {
	ui.Column(c).Grow(1).FillHeight().AlignItems(ui.Center).Children(func() {
		ui.Row(c).Grow(1)
		ui.Text(c, eyebrow).FontSize(11).LetterSpacing(1).
			SingleLine().TextColor(p.mutedFg.color())
		ui.Text(c, msg).FontSize(12.5).SingleLine().TextColor(p.mutedFg.color())
		ui.Row(c).Grow(1)
	})
}

func (a *app) respFailed(c *ui.Context, p palette) {
	ui.Column(c).Grow(1).FillHeight().AlignItems(ui.Center).Children(func() {
		ui.Row(c).Grow(1)
		ui.Text(c, "REQUEST FAILED").FontSize(11).LetterSpacing(1).
			SingleLine().TextColor(p.bad.color())
		ui.Text(c, a.run.errText).FontSize(12.5).SingleLine().
			TextColor(p.mutedFg.color())
		ui.Text(c, "Check the URL, DNS, or the timeout setting for this request.").
			FontSize(12).SingleLine().TextColor(p.mutedFg.color())
		ui.Row(c).Grow(1)
	})
}

func statusColor(p palette, code int) lch {
	switch {
	case code < 300:
		return p.ok
	case code < 400:
		return p.accent
	case code < 500:
		return p.warn
	default:
		return p.bad
	}
}

// respBody shows the response: status line, General facts, timing bars and
// the body preview.
func (a *app) respBody(c *ui.Context, p palette) {
	r := a.run.resp
	ui.Row(c).Height(36).Padding(0, 16, 0, 16).Gap(12).
		AlignItems(ui.Center).Children(func() {
		ui.Row(c).Height(22).Padding(0, 8, 0, 8).AlignItems(ui.Center).
			Background(statusColor(p, r.Status).color().Alpha(0.12)).
			Children(func() {
				ui.Text(c, strconv.Itoa(r.Status)).FontSize(11.5).Bold().
					Font("JetBrains Mono").SingleLine().
					TextColor(statusColor(p, r.Status).color())
			})
		if r.StatusText != "" {
			ui.Text(c, r.StatusText).FontSize(12).SingleLine()
		}
		ui.Row(c).Grow(1)
		ui.Text(c, fmt.Sprintf("%d ms · %s", r.DurationMs, humanBytes(r.SizeBytes))).
			FontSize(11).Font("JetBrains Mono").SingleLine().
			TextColor(p.mutedFg.color())
	})
	a.respGeneral(c, p)
	a.respTiming(c, p)
	ui.Row(c).Height(28).Padding(0, 16, 0, 16).Gap(8).
		AlignItems(ui.Center).Children(func() {
		ui.Text(c, "BODY").FontSize(10).LetterSpacing(0.8).SingleLine().
			TextColor(p.mutedFg.color())
		ui.Row(c).Grow(1)
		if r.Truncated {
			ui.Text(c, "truncated").FontSize(10.5).SingleLine().
				TextColor(p.warn.color())
		}
	})
	if a.run.bodySt != nil {
		editor.View(c, a.run.bodySt).Grow(1).FillHeight().
			Padding(4, 16, 8, 16)
	}
}

// respGeneral lists the request-side facts the web General section shows.
func (a *app) respGeneral(c *ui.Context, p palette) {
	r := a.run.resp
	rows := [][2]string{
		{"Request URL", r.RequestURL},
		{"Remote address", r.RemoteAddr},
		{"HTTP version", r.HTTPVersion},
	}
	if r.FinalURL != "" && r.FinalURL != r.RequestURL {
		rows = append(rows, [2]string{"Final URL", r.FinalURL})
	}
	for _, kv := range rows {
		if kv[1] == "" {
			continue
		}
		ui.Row(c).Height(22).Padding(0, 16, 0, 16).Gap(8).
			AlignItems(ui.Center).Children(func() {
			ui.Text(c, kv[0]).FontSize(11).Width(110).SingleLine().
				TextColor(p.mutedFg.color())
			ui.Text(c, kv[1]).FontSize(11).Font("JetBrains Mono").
				SingleLine().Grow(1)
		})
	}
}

// respTiming draws the per-phase bars, the web timing panel's grid.
func (a *app) respTiming(c *ui.Context, p palette) {
	r := a.run.resp
	if r.Timing == nil {
		return
	}
	phases := []struct {
		label string
		ms    int64
	}{
		{"DNS", r.Timing.DNSMs},
		{"Connect", r.Timing.ConnectMs},
		{"TLS", r.Timing.TLSMs},
		{"TTFB", r.Timing.TTFBMs},
		{"Download", r.Timing.DownloadMs},
	}
	total := r.DurationMs
	if total < 1 {
		total = 1
	}
	any := false
	for _, ph := range phases {
		if ph.ms != 0 {
			any = true
		}
	}
	if !any {
		return
	}
	for _, ph := range phases {
		ui.Row(c).Height(20).Padding(0, 16, 0, 16).Gap(8).
			AlignItems(ui.Center).Children(func() {
			ui.Text(c, ph.label).FontSize(11).Width(70).SingleLine().
				TextColor(p.mutedFg.color())
			ui.Row(c).Width(240).Height(6).Background(p.muted.color()).
				Radius(3).Children(func() {
				if ph.ms > 0 {
					frac := float32(ph.ms) / float32(total)
					if frac < 0.02 {
						frac = 0.02
					}
					ui.Row(c).Width(240 * frac).Height(6).
						Background(p.foreground.color().Alpha(0.4)).
						Radius(3)
				}
				ui.Row(c).Grow(1)
			})
			ms := "—"
			if ph.ms > 0 {
				ms = fmt.Sprintf("%d ms", ph.ms)
			}
			ui.Text(c, ms).FontSize(10.5).Font("JetBrains Mono").
				SingleLine().TextColor(p.mutedFg.color())
		})
	}
}

func humanBytes(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}
