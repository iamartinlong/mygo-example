package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo"
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
	envName string // active environment's name when the request went out

	// The response pane's tabs and their state, as the web's cF keeps:
	// headers | request | preview | response.
	respTab      string
	respLastTab  string
	respScroll   ui.ScrollState
	respCopyOpen bool
	respClosed   map[string]bool // collapsible sections, open by default

	bodyBuf *editor.Buffer
	bodySt  *editor.State
	rawSt   *editor.State
	reqSt   *editor.State
}

func (a *app) send() {
	if a.run.running || a.exec == nil || a.edit.id == "" {
		return
	}
	req := a.editRequest()
	name := a.edit.name
	opts := a.runOpts()
	vars := a.resolveEnvVars()
	a.run.running = true
	a.run.errText = ""
	if e := a.envByID(a.activeEnv); e != nil {
		a.run.envName = e.Name
	} else {
		a.run.envName = ""
	}
	go func() {
		resp, err := a.exec.Run(context.Background(), req, vars, opts)
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
				a.toastErr(errText)
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

// historyResponse rebuilds a runner.Response from a history row. What was
// actually sent lives only in the snapshot, so the request headers and body
// here are the draft's, not the substituted wire form the web's history API
// returns.
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
	if exec, ok := store.ParseSnapshot(e.RequestSnapshot); ok {
		req := exec.Request
		r.RequestType = req.BodyType
		r.RequestBody = req.Body
		r.RequestHeaders = map[string][]string{}
		for _, h := range req.Headers {
			if h.Enabled && h.Key != "" {
				r.RequestHeaders[h.Key] = append(r.RequestHeaders[h.Key], h.Value)
			}
		}
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

// resetRespBuf rebinds the three read-only previews (body, raw, sent
// request) to the latest response.
func (a *app) resetRespBuf() {
	r := a.run.resp
	if r == nil {
		return
	}
	a.run.bodyBuf = editor.NewBuffer(r.BodyText)
	lang := editor.LanguageFor("plain.txt")
	if bodyIsJSON(r) {
		lang = editor.LanguageFor("x.json")
	}
	a.run.bodyBuf.SetLanguage(lang)
	a.run.bodySt = &editor.State{Buffer: a.run.bodyBuf, SoftWrap: true, ReadOnly: true}

	raw := editor.NewBuffer(r.BodyText)
	raw.SetLanguage(editor.LanguageFor("plain.txt"))
	a.run.rawSt = &editor.State{Buffer: raw, SoftWrap: true, ReadOnly: true}

	a.run.reqSt = nil
	if r.RequestBody != "" {
		l := editor.LanguageFor("plain.txt")
		if r.RequestType == "json" {
			l = editor.LanguageFor("x.json")
		}
		b := editor.NewBuffer(r.RequestBody)
		b.SetLanguage(l)
		a.run.reqSt = &editor.State{Buffer: b, SoftWrap: true, ReadOnly: true}
	}
}

// bodyIsJSON decides the preview's language, as the web's uF does: by the
// content type, then by whether the body reads as JSON.
func bodyIsJSON(r *runner.Response) bool {
	if r.Encoding == "base64" {
		return false
	}
	if strings.Contains(r.ContentType, "json") {
		return true
	}
	t := strings.TrimSpace(r.BodyText)
	return strings.HasPrefix(t, "{") && strings.HasSuffix(t, "}") ||
		strings.HasPrefix(t, "[") && strings.HasSuffix(t, "]")
}

// runBar mirrors the web strip at the foot of the request pane: timeout,
// redirects, TLS, and the "GET · none" summary on the right.
func (a *app) runBar(c *ui.Context, p palette) {
	ui.Row(c).Height(37).Padding(6, 16, 6, 16).Gap(12).
		BorderWidth(1, 0, 0, 0).BorderColor(p.hairline.color()).
		AlignItems(ui.Center).Children(func() {
		ui.Text(c, tr("run.timeout")).FontSize(11).SingleLine().
			TextColor(p.mutedFg.color())
		ui.TextInput(c, &a.run.timeout).Width(56).Height(24).FontSize(12)
		ui.Text(c, tr("run.seconds")).FontSize(11).SingleLine().TextColor(p.mutedFg.color())
		ui.Checkbox(c, &a.run.follow, tr("run.followRedirects"))
		ui.Checkbox(c, &a.run.verify, tr("run.verifyTls"))
		ui.Row(c).Grow(1)
		// The web's text-muted-foreground/60 on the summary.
		ui.Text(c, a.edit.method+" · "+a.edit.bodyType).FontSize(10).
			Font("JetBrains Mono").SingleLine().
			TextColor(p.mutedFg.color().Alpha(0.6))
	})
}

// responsePane is everything right of (or under) the run bar: waiting
// skeleton, empty state, or the loaded response with its tabs.
func (a *app) responsePane(c *ui.Context, p palette) {
	// The web paints this band as bg-background with bg-panel/40 on top.
	ui.Column(c).FillHeight().Background(
		p.panel.color().Alpha(0.4).Over(p.background.color())).
		Children(func() {
			switch {
			case a.run.running && a.run.resp == nil && a.run.errText == "":
				a.respWaiting(c, p)
			case a.run.resp == nil && a.run.errText == "":
				a.respCenter(c, p, tr("resp.noResponse"),
					tr("resp.noResponseHint"), "⌘ / Ctrl + Enter")
			default:
				a.respLoaded(c, p)
			}
		})
}

// respWaiting is the web's loading skeleton: a h-9 waiting bar over five
// pulse bars. MyGo has no pulse animation, so the bars sit still.
func (a *app) respWaiting(c *ui.Context, p palette) {
	ui.Row(c).Height(36).Padding(0, 16, 0, 16).AlignItems(ui.Center).
		BorderWidth(0, 0, 1, 0).BorderColor(p.border.color()).
		Children(func() {
			ui.Text(c, tr("resp.waiting")).FontSize(11).
				Font("JetBrains Mono").SingleLine().TextColor(p.mutedFg.color())
		})
	ui.Column(c).Grow(1).FillHeight().Padding(16, 16, 16, 16).Gap(8).
		Children(func() {
			for _, w := range []float32{92, 78, 84, 60, 70} {
				ui.Row(c).FillWidth().Height(10).Children(func() {
					ui.Row(c).WidthPercent(w).Height(10).Radius(4).
						Background(p.muted.color().Alpha(0.7))
				})
			}
		})
}

// respCenter is the web's centred empty state: a mono eyebrow, the hint
// sentence, and the shortcut line the response panel hardcodes.
func (a *app) respCenter(c *ui.Context, p palette, eyebrow, msg, keys string) {
	ui.Column(c).Grow(1).FillHeight().AlignItems(ui.Center).Gap(8).
		Children(func() {
			ui.Row(c).Grow(1)
			ui.Text(c, eyebrow).FontSize(11).LetterSpacing(2.2).
				Font("JetBrains Mono").SingleLine().
				TextColor(p.mutedFg.color().Alpha(0.5))
			ui.Text(c, msg).FontSize(12).SingleLine().TextColor(p.mutedFg.color())
			if keys != "" {
				ui.Text(c, keys).FontSize(10).Margin(8, 0, 0, 0).
					Font("JetBrains Mono").SingleLine().
					TextColor(p.mutedFg.color().Alpha(0.5))
			}
			ui.Row(c).Grow(1)
		})
}

// statusColor is the web's vh: zero (no response) is muted, 2xx ok, 3xx
// warn, everything else bad.
func statusColor(p palette, code int) lch {
	switch {
	case code == 0:
		return p.mutedFg
	case code >= 200 && code < 300:
		return p.ok
	case code >= 300 && code < 400:
		return p.warn
	default:
		return p.bad
	}
}

// webBytes is the web's yh: B, KB with one decimal, MB with two.
func webBytes(n int64) string {
	switch {
	case n == 0:
		return "0 B"
	case n < 1024:
		return fmt.Sprintf("%d B", n)
	case n < 1048576:
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	default:
		return fmt.Sprintf("%.2f MB", float64(n)/1048576)
	}
}

// webDuration is the web's bh: milliseconds under a second, seconds after.
func webDuration(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%d ms", ms)
	}
	return fmt.Sprintf("%.2f s", float64(ms)/1000)
}

var respTabKeys = []string{"headers", "request", "preview", "response"}

// respTabLabels names each response tab; the request one has its own key.
var respTabLabels = map[string]string{
	"headers": "tabs.headers", "request": "tabs.requestBody",
	"preview": "tabs.preview", "response": "tabs.response",
}

// respLoaded is a response (or a failure) with the status bar and the four
// line tabs, as the web's cF lays out.
func (a *app) respLoaded(c *ui.Context, p palette) {
	r := a.run.resp
	ui.Row(c).Height(36).Padding(0, 16, 0, 16).Gap(8).
		AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).
		BorderColor(p.border.color()).Children(func() {
		if r == nil {
			// The web's failure status row: a bad X, the label, the error.
			ui.Icon(c, iconX).Size(14, 14).TextColor(p.bad.color())
			ui.Text(c, tr("resp.requestFailed")).FontSize(11.5).FontWeight(600).
				Font("JetBrains Mono").SingleLine().TextColor(p.bad.color())
			ui.Text(c, a.run.errText).FontSize(11).SingleLine().Grow(1).
				TextColor(p.mutedFg.color())
			return
		}
		ink := statusColor(p, r.Status).color()
		ui.Text(c, strconv.Itoa(r.Status)).FontSize(12).Bold().
			Font("JetBrains Mono").FontFeatures("tnum").SingleLine().TextColor(ink)
		ui.Text(c, r.StatusText).FontSize(11.5).SingleLine().
			TextColor(p.mutedFg.color())
		respDot(c, p)
		ui.Text(c, webDuration(r.DurationMs)).FontSize(11).
			Font("JetBrains Mono").FontFeatures("tnum").SingleLine().
			TextColor(p.mutedFg.color())
		respDot(c, p)
		ui.Text(c, webBytes(r.SizeBytes)).FontSize(11).
			Font("JetBrains Mono").SingleLine().TextColor(p.mutedFg.color())
		if a.run.envName != "" {
			respDot(c, p)
			ui.Text(c, tr("sidebar.envPrefix")+" "+a.run.envName).FontSize(10.5).
				SingleLine().TextColor(p.mutedFg.color())
		}
		if r.Truncated {
			ui.Row(c).Height(18).Padding(0, 4, 0, 4).Radius(4).
				Background(p.warn.color().Alpha(0.15)).AlignItems(ui.Center).
				Children(func() {
					ui.Text(c, tr("resp.truncated")).FontSize(10).
						Font("JetBrains Mono").SingleLine().TextColor(p.warn.color())
				})
		}
		ui.Row(c).Grow(1)
		a.respDownloadBtn(c, p, r)
		a.respCopyMenu(c, p, r)
	})
	a.respTabs(c, p, r)
}

// respDot is the status row's thin separator, the web's text-border dot.
func respDot(c *ui.Context, p palette) {
	ui.Text(c, "·").FontSize(11).SingleLine().TextColor(p.border.color())
}

// respDownloadBtn offers the body through a save dialog, as the web's
// download link does.
func (a *app) respDownloadBtn(c *ui.Context, p palette, r *runner.Response) {
	e := ui.Row(c).Height(24).Padding(0, 8, 0, 8).Gap(4).Radius(6).
		AlignItems(ui.Center)
	ink := p.mutedFg.color()
	if e.Hovered() {
		e.Background(p.muted.color())
		ink = p.foreground.color()
	}
	e.Children(func() {
		ui.Icon(c, iconDownload).Size(14, 14).TextColor(ink)
		ui.Text(c, tr("resp.download")).FontSize(12).SingleLine().TextColor(ink)
	})
	if e.Clicked() {
		a.saveResponseBody(r)
	}
}

func (a *app) saveResponseBody(r *runner.Response) {
	data := r.Body
	if len(data) == 0 {
		data = []byte(r.BodyText)
	}
	if r.Encoding == "base64" {
		if raw, err := base64.StdEncoding.DecodeString(r.BodyText); err == nil {
			data = raw
		}
	}
	ext := ".txt"
	if strings.Contains(r.ContentType, "json") {
		ext = ".json"
	}
	go func() {
		path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
			Title:       tr("resp.download"),
			DefaultPath: "response" + ext,
		})
		if err != nil || path == "" {
			return
		}
		err = os.WriteFile(path, data, 0o644)
		a.win.Update(func() {
			if err != nil {
				a.toastErr(tr("toast.exportFailed"))
			} else {
				a.toastOK(tr("common.copied"))
			}
		})
	}()
}

// respCopyMenu is the web's hF dropdown: Copy ▾ with a single Body item.
func (a *app) respCopyMenu(c *ui.Context, p palette, r *runner.Response) {
	anchor := ui.Row(c).Height(24).Padding(0, 8, 0, 8).Gap(4).Radius(6).
		AlignItems(ui.Center)
	ink := p.mutedFg.color()
	if anchor.Hovered() {
		anchor.Background(p.muted.color())
		ink = p.foreground.color()
	}
	anchor.Children(func() {
		ui.Icon(c, iconCopy).Size(14, 14).TextColor(ink)
		ui.Text(c, tr("common.copy")).FontSize(12).SingleLine().TextColor(ink)
		ui.Icon(c, iconChevronDown).Size(12, 12).TextColor(ink)
	})
	if anchor.Clicked() {
		a.run.respCopyOpen = !a.run.respCopyOpen
	}
	ui.PopoverBase(c, anchor, &a.run.respCopyOpen, func(panel ui.Element) {
		panel.MinWidth(160).Margin(4, 0, 0, 0).Padding(4).Background(p.card.color()).
			Radius(6).Border(1, p.border.color())
		it := ui.Row(c).FillWidth().Height(28).Padding(0, 8, 0, 8).Gap(6).
			AlignItems(ui.Center).Radius(4)
		ink := p.foreground.color()
		if it.Hovered() {
			it.Background(p.accent.color())
		}
		it.Children(func() {
			ui.Text(c, tr("tabs.body")).FontSize(12).SingleLine().TextColor(ink)
		})
		if it.Clicked() {
			mygo.Clipboard.WriteText(r.BodyText)
			a.run.respCopyOpen = false
		}
	})
}

// respTabs is the response pane's line tab strip: Headers (count), Request
// (sent size), Preview, Response.
func (a *app) respTabs(c *ui.Context, p palette, r *runner.Response) {
	if a.run.respTab == "" {
		a.run.respTab = "preview"
	}
	counts := map[string]string{"headers": "", "request": ""}
	if r != nil {
		counts["headers"] = fmt.Sprint(len(r.Headers))
		if n := int64(len(r.RequestBody)); n > 0 {
			counts["request"] = webBytes(n)
		}
	}
	ui.Row(c).Height(32).Padding(0, 16, 0, 16).Gap(16).
		AlignItems(ui.End).BorderWidth(0, 0, 1, 0).
		BorderColor(p.border.color().Alpha(0.6)).Children(func() {
		for _, tab := range respTabKeys {
			t := tab
			active := a.run.respTab == t
			ink := p.foreground.color().Alpha(0.6)
			if p.isDark {
				ink = p.mutedFg.color()
			}
			col := ui.Column(c).Height(32).
				AlignItems(ui.Start).Children(func() {
				ui.Row(c).Grow(1).Gap(4).AlignItems(ui.Center).Children(func() {
					tt := ui.Text(c, tr(respTabLabels[t])).FontSize(12).FontWeight(500).
						SingleLine()
					if active {
						tt.TextColor(p.foreground.color())
					} else {
						tt.TextColor(ink)
					}
					// The web marks these with a plain mono count, not the
					// chip the request tabs use.
					if n := counts[t]; n != "" {
						ui.Text(c, n).FontSize(10).Font("JetBrains Mono").
							SingleLine().TextColor(p.mutedFg.color())
					}
				})
			})
			if active {
				col.BorderWidth(0, 0, 2, 0).BorderColor(p.foreground.color())
			}
			if col.Clicked() {
				a.run.respTab = t
			}
		}
	})
	// A tab switch remounts the web's panel, so its scroll starts over.
	if a.run.respLastTab != a.run.respTab {
		a.run.respLastTab = a.run.respTab
		a.run.respScroll = ui.ScrollState{}
	}
	ui.Scroll(c).FillHeight().FillWidth().TrackScroll(&a.run.respScroll).
		Children(func() {
			ui.Column(c).FillWidth().Padding(12, 16, 12, 16).Children(func() {
				switch a.run.respTab {
				case "headers":
					a.respHeadersTab(c, p, r)
				case "request":
					a.respRequestTab(c, p, r)
				case "response":
					a.respRawTab(c, p, r)
				default:
					a.respPreviewTab(c, p, r)
				}
			})
		})
}

// respErrBox is the web's error card: the message in bad mono over the
// check-URL hint.
func respErrBox(c *ui.Context, p palette, err, hint string) {
	ui.Column(c).FillWidth().Radius(6).Border(1, p.bad.color().Alpha(0.3)).
		Background(p.bad.color().Alpha(0.05)).Padding(10, 12, 10, 12).
		Children(func() {
			ui.Text(c, err).FontSize(11).Font("JetBrains Mono").
				TextColor(p.bad.color())
			ui.Row(c).Height(6)
			ui.Text(c, hint).FontSize(10.5).TextColor(p.mutedFg.color())
		})
}

// respEditor draws a read-only editor in the web's $P frame: a hairline
// bordered box on a tinted ground with 10px of lead above the text and 8px
// below it, tall enough for every line at the editor's 12.5px/1.65 leading
// and never shorter than minH. The web editor grows past its min height and
// the panel scrolls; the line count is capped so an enormous body stays a
// scroll rather than a layout cost.
func respEditor(c *ui.Context, p palette, st *editor.State, text string, minH float32) {
	lines := float32(strings.Count(text, "\n") + 1)
	if lines > 2000 {
		lines = 2000
	}
	h := lines*12.5*1.65 + 10 + 8
	if h < minH {
		h = minH
	}
	ui.Column(c).FillWidth().Height(h).Radius(6).
		Border(1, p.border.color().Alpha(0.7)).
		Background(p.background.color().Alpha(0.4)).Clip().Children(func() {
		ui.Row(c).Height(10)
		editor.View(c, st).Fill()
	})
}

// respPreviewTab shows the body as the web's preview does: an error card,
// an image, a binary note, or the highlighted editor.
func (a *app) respPreviewTab(c *ui.Context, p palette, r *runner.Response) {
	if r == nil {
		respErrBox(c, p, a.run.errText, tr("resp.checkUrl"))
		return
	}
	image := strings.HasPrefix(r.ContentType, "image/")
	switch {
	case image:
		a.respImage(c, p, r)
	case r.Encoding == "base64":
		a.respBinaryNote(c, p, r)
	default:
		if a.run.bodySt != nil {
			respEditor(c, p, a.run.bodySt, r.BodyText, 224)
		}
	}
}

// respImage decodes a base64 image body and shows it, captioned as the web
// does.
func (a *app) respImage(c *ui.Context, p palette, r *runner.Response) {
	raw, err := base64.StdEncoding.DecodeString(r.BodyText)
	if err != nil {
		raw = []byte(r.BodyText)
	}
	bmp, err := ui.DecodeBitmap(raw)
	if err != nil {
		a.respBinaryNote(c, p, r)
		return
	}
	ui.Image(c, bmp)
	ui.Row(c).Height(6)
	ui.Text(c, r.ContentType+" · "+webBytes(r.SizeBytes)).FontSize(10).
		Font("JetBrains Mono").SingleLine().TextColor(p.mutedFg.color())
}

// respBinaryNote is the web's dashed binary card with its download button.
func (a *app) respBinaryNote(c *ui.Context, p palette, r *runner.Response) {
	kind := r.ContentType
	if kind == "" {
		kind = tr("resp.unknownType")
	}
	ui.Row(c).FillWidth().Height(128).Radius(6).
		Border(1, p.border.color().Alpha(0.7)).BorderStyle(ui.BorderDashed).
		AlignItems(ui.Center).Children(func() {
		ui.Column(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconFileText).Size(20, 20).
				TextColor(p.mutedFg.color().Alpha(0.6))
			ui.Text(c, tr("resp.binaryResponse", "type", kind)).FontSize(12).
				TextColor(p.mutedFg.color())
		})
	})
}

// respHeadersTab is the web's pF: General, then the header groups and the
// timing, each a collapsible section.
func (a *app) respHeadersTab(c *ui.Context, p palette, r *runner.Response) {
	a.respGeneral(c, p, r)
	if r == nil {
		return
	}
	a.respSection(c, p, "respHeaders", tr("resp.responseHeaders"), len(r.Headers), func() {
		a.respHeaderList(c, p, r.Headers)
	})
	a.respSection(c, p, "reqHeaders", tr("resp.requestHeaders"), len(r.RequestHeaders), func() {
		a.respHeaderList(c, p, r.RequestHeaders)
	})
	if len(r.Cookies) > 0 {
		a.respSection(c, p, "cookies", tr("resp.cookiesSet"), len(r.Cookies), func() {
			for _, ck := range r.Cookies {
				ui.Row(c).FillWidth().Height(18).Children(func() {
					ui.Text(c, ck).FontSize(11).Font("JetBrains Mono").
						TextColor(p.mutedFg.color())
				})
			}
		})
	}
	if r.Timing != nil {
		a.respSection(c, p, "timing", tr("resp.timing"), 0, func() {
			a.respTiming(c, p, r)
		})
	}
}

// respGeneral is the General block's fact rows, the web's grid of label
// over value with hairlines between.
func (a *app) respGeneral(c *ui.Context, p palette, r *runner.Response) {
	type fact struct {
		label, value string
		mono, muted  bool
		color        lch
		bold         bool
	}
	var rows []fact
	url := ""
	if r != nil {
		url = r.RequestURL
	}
	rows = append(rows, fact{label: tr("resp.requestUrl"), value: url, mono: true})
	if r != nil {
		if r.RemoteAddr != "" {
			rows = append(rows, fact{label: tr("resp.remoteAddress"), value: r.RemoteAddr, mono: true})
		}
		if r.HTTPVersion != "" {
			rows = append(rows, fact{label: tr("resp.httpVersion"), value: r.HTTPVersion, mono: true})
		}
		if r.Status > 0 {
			rows = append(rows, fact{label: tr("resp.status"),
				value: strconv.Itoa(r.Status) + " " + r.StatusText,
				mono:  true, bold: true, color: statusColor(p, r.Status)})
		} else {
			rows = append(rows, fact{label: tr("resp.status"), value: tr("common.failed"), color: p.bad})
		}
		rows = append(rows, fact{label: tr("resp.duration"), value: webDuration(r.DurationMs), mono: true})
		rows = append(rows, fact{label: tr("resp.size"), value: webBytes(r.SizeBytes), mono: true})
		if r.FinalURL != "" && r.FinalURL != url {
			rows = append(rows, fact{label: tr("resp.finalUrl"), value: r.FinalURL, mono: true, muted: true})
		}
		if !r.ReceivedAt.IsZero() {
			rows = append(rows, fact{label: tr("resp.received"),
				value: r.ReceivedAt.Format("2006/01/02 15:04:05"), mono: true})
		}
	} else {
		rows = append(rows, fact{label: tr("resp.status"), value: tr("common.failed"), color: p.bad})
	}
	a.respSection(c, p, "general", tr("resp.general"), 0, func() {
		for i, f := range rows {
			row := ui.Row(c).FillWidth().Padding(3, 0, 3, 0).Gap(12).
				AlignItems(ui.Center)
			if i < len(rows)-1 {
				row.BorderWidth(0, 0, 1, 0).BorderColor(p.border.color().Alpha(0.4))
			}
			row.Children(func() {
				ui.Text(c, f.label).FontSize(11).SingleLine().
					TextColor(p.mutedFg.color()).Grow(2).Basis(0).MinWidth(0)
				t := ui.Text(c, f.value).FontSize(11).Grow(5).Basis(0).MinWidth(0)
				if f.mono {
					t.Font("JetBrains Mono")
				}
				if f.muted {
					t.TextColor(p.mutedFg.color())
				}
				if f.color != (lch{}) {
					t.TextColor(f.color.color())
				}
				if f.bold {
					t.FontWeight(600)
				}
			})
		}
	})
}

// respHeaderList is the web's fF: entries sorted by name, key and joined
// values in mono on a two-track grid.
func (a *app) respHeaderList(c *ui.Context, p palette, headers map[string][]string) {
	names := make([]string, 0, len(headers))
	for k := range headers {
		names = append(names, k)
	}
	sort.Strings(names)
	if len(names) == 0 {
		ui.Row(c).FillWidth().Padding(8, 0, 8, 0).Children(func() {
			ui.Text(c, tr("resp.noneInGroup")).FontSize(11).
				TextColor(p.mutedFg.color())
		})
		return
	}
	for _, k := range names {
		ui.Row(c).FillWidth().Padding(3, 0, 3, 0).Gap(12).AlignItems(ui.Center).
			Children(func() {
				ui.Text(c, k).FontSize(11).Font("JetBrains Mono").SingleLine().
					TextColor(p.mutedFg.color()).Grow(1).Basis(0).MinWidth(0)
				ui.Text(c, strings.Join(headers[k], ", ")).FontSize(11).
					Font("JetBrains Mono").Grow(1.7).Basis(0).MinWidth(0)
			})
	}
}

// respTiming draws the per-phase bars, the web timing panel's grid.
func (a *app) respTiming(c *ui.Context, p palette, r *runner.Response) {
	t := r.Timing
	phases := []struct {
		label string
		ms    int64
		note  string
	}{
		{tr("resp.timing.dns"), t.DNSMs, ""},
		{tr("resp.timing.connect"), t.ConnectMs, reusedNote(t.Reused)},
		{tr("resp.timing.tls"), t.TLSMs, ""},
		{tr("resp.timing.ttfb"), t.TTFBMs, tr("resp.timing.includes")},
		{tr("resp.timing.download"), t.DownloadMs, ""},
	}
	total := r.DurationMs
	if total < 1 {
		total = 1
	}
	notes := map[string]bool{}
	for _, ph := range phases {
		pct := float32(ph.ms) / float32(total) * 100
		if pct > 100 {
			pct = 100
		}
		zero := ph.ms == 0
		ui.Row(c).FillWidth().Height(18).Gap(8).AlignItems(ui.Center).
			Children(func() {
				ui.Text(c, ph.label).FontSize(11).Width(120).SingleLine().
					TextColor(p.mutedFg.color())
				ui.Row(c).Grow(1).Height(6).Radius(3).
					Background(p.muted.color().Alpha(0.6)).Children(func() {
					if !zero {
						frac := pct / 100
						if frac < 0.01 {
							frac = 0.01
						}
						ui.Row(c).FillHeight().WidthPercent(frac * 100).
							Radius(3).Background(p.foreground.color().Alpha(0.4))
					}
				})
				ms := "—"
				if !zero {
					ms = fmt.Sprintf("%d ms", ph.ms)
				}
				ui.Text(c, ms).FontSize(10.5).Font("JetBrains Mono").
					SingleLine().TextColor(p.mutedFg.color())
			})
		if ph.note != "" {
			notes[ph.note] = true
		}
	}
	if len(notes) > 0 {
		joined := make([]string, 0, len(notes))
		for _, ph := range phases {
			if ph.note != "" && notes[ph.note] {
				joined = append(joined, ph.note)
				notes[ph.note] = false
			}
		}
		ui.Row(c).FillWidth().Padding(2, 0, 0, 0).Children(func() {
			ui.Text(c, strings.Join(joined, " · ")).FontSize(10).
				Font("JetBrains Mono").TextColor(p.mutedFg.color().Alpha(0.7))
		})
	}
}

func reusedNote(reused bool) string {
	if reused {
		return tr("resp.timing.reused")
	}
	return ""
}

// respSection is the web's dF collapsible: a header row over an indented
// body with a border-l spine, open unless the user closed it.
func (a *app) respSection(c *ui.Context, p palette, key, title string, count int, body func()) {
	if a.run.respClosed == nil {
		a.run.respClosed = map[string]bool{}
	}
	open := !a.run.respClosed[key]
	hdr := ui.Row(c).FillWidth().Height(24).Padding(0, 4, 0, 4).Gap(6).
		AlignItems(ui.Center).Radius(4)
	if hdr.Hovered() {
		hdr.Background(p.accent.color().Alpha(0.5))
	}
	hdr.Children(func() {
		if open {
			ui.Icon(c, iconChevronDown).Size(12, 12).TextColor(p.mutedFg.color())
		} else {
			ui.Icon(c, iconChevronRt).Size(12, 12).TextColor(p.mutedFg.color())
		}
		ui.Text(c, title).FontSize(11.5).FontWeight(500).SingleLine()
		ui.Text(c, fmt.Sprint(count)).FontSize(10).Font("JetBrains Mono").
			SingleLine().TextColor(p.mutedFg.color())
	})
	if hdr.Clicked() {
		a.run.respClosed[key] = open // closed when it was open
	}
	if !open {
		return
	}
	ui.Column(c).FillWidth().Padding(2, 0, 0, 14).
		BorderWidth(0, 0, 0, 1).BorderColor(p.border.color().Alpha(0.6)).
		Children(body)
}

// respRequestTab shows the body that actually went out, or the dashed note
// when none did.
func (a *app) respRequestTab(c *ui.Context, p palette, r *runner.Response) {
	if r == nil || r.RequestBody == "" {
		kind := tr("common.none")
		if r != nil && r.RequestType != "" {
			kind = r.RequestType
		}
		ui.Row(c).FillWidth().Height(128).Radius(6).
			Border(1, p.border.color().Alpha(0.7)).BorderStyle(ui.BorderDashed).
			AlignItems(ui.Center).Children(func() {
			ui.Column(c).Gap(4).AlignItems(ui.Center).Children(func() {
				ui.Text(c, tr("resp.noRequestBody")).FontSize(12).
					TextColor(p.mutedFg.color())
				ui.Text(c, tr("resp.bodyType")+": "+kind).FontSize(10.5).
					Font("JetBrains Mono").
					TextColor(p.mutedFg.color().Alpha(0.7))
			})
		})
		return
	}
	ui.Row(c).FillWidth().Height(22).Gap(8).AlignItems(ui.Center).
		Children(func() {
			ui.Text(c, strings.ToUpper(tr("resp.sentAfterSubstitution"))).
				FontSize(11).LetterSpacing(1.54).LineHeight(1.4286).
				Font("JetBrains Mono").SingleLine().TextColor(p.mutedFg.color())
			ui.Text(c, r.RequestType+" · "+webBytes(int64(len(r.RequestBody)))).
				FontSize(10).Font("JetBrains Mono").SingleLine().
				TextColor(p.mutedFg.color())
		})
	if a.run.reqSt != nil {
		respEditor(c, p, a.run.reqSt, r.RequestBody, 192)
	}
}

// respRawTab shows the body verbatim, as the web's Response tab does.
func (a *app) respRawTab(c *ui.Context, p palette, r *runner.Response) {
	if r == nil {
		respErrBox(c, p, a.run.errText, tr("resp.noResponseReceived"))
		return
	}
	if r.Encoding == "base64" && !strings.HasPrefix(r.ContentType, "image/") {
		kind := r.ContentType
		if kind == "" {
			kind = tr("resp.unknownType")
		}
		ui.Text(c, tr("resp.binaryHint", "type", kind)).FontSize(11).
			TextColor(p.mutedFg.color())
		ui.Row(c).Height(6)
	} else {
		ui.Row(c).FillWidth().Height(20).Children(func() {
			ui.Text(c, strings.ToUpper(tr("resp.rawHint"))).FontSize(11).
				LetterSpacing(1.54).LineHeight(1.4286).Font("JetBrains Mono").
				SingleLine().TextColor(p.mutedFg.color())
		})
	}
	if a.run.rawSt != nil {
		respEditor(c, p, a.run.rawSt, r.BodyText, 192)
	}
}
