// Reqbox in native UI: the web frontend rebuilt with mygo's ui package,
// talking to the same store in-process. Route B of the migration
// (docs/01-research/reqbox-migration.md); the webview shell in
// reqbox-desktop/ stays as the 100%-fidelity baseline.
package main

import (
	"log"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"reqbox-desktop/pkg/runner"
	"reqbox-desktop/pkg/store"
)

type app struct {
	dark    bool    // follows the web .dark variant; a toggle arrives in B13
	sidebar float32 // sidebar width in DIPs, ui.Split drags it

	st          *store.Store
	reqs        []store.Request
	cols        []store.Collection
	rows        []sideRow
	list        ui.ListState
	selRow      int // ui.List choice; -1 none
	selectedID  string
	filter      string
	lastFilter  string
	pending     pending
	edit        reqEdit
	showFullURL bool
	bodyBuf     *editor.Buffer
	bodyState   *editor.State

	sideTab    string // "requests" or "history"
	hist       []store.HistoryEntry
	histList   ui.ListState
	histSel    int
	histOpenID string

	envs        []store.Environment
	activeEnv   string
	envChoice   string
	envMgr      bool
	envEditID   string
	envEditName string
	envEditVars []envVarEdit
	confirm     confirmState
	toasts      []toastMsg
	runDlg      bool

	win      *mygo.Window
	exec     *runner.Executor
	run      runState
	mockAddr string
}

func (a *app) view(c *ui.Context) {
	p := a.pal()
	c.SetTheme(theme(p, c.Theme()))

	// Window shortcuts, as on the web: Ctrl+Enter send, Ctrl+S save,
	// Ctrl+N new request. c.Shortcut is inert while a modal is open.
	if c.Shortcut(ui.Ctrl, ui.KeyEnter) {
		a.send()
	}
	if c.Shortcut(ui.Ctrl, ui.KeyS) {
		a.saveEdit()
	}
	if c.Shortcut(ui.Ctrl, ui.KeyN) {
		a.pending.newRequest = true
	}

	a.drainToasts(c, p)
	ui.Column(c).Fill().Children(func() {
		a.topBar(c, p)
		ui.Split(c, &a.sidebar, func() { a.sidebarPane(c, p) }, func() { a.mainPane(c, p) }).Fill()
		a.envDialog(c, p)
		a.confirmDialog(c, p)
		a.runSettingsDialog(c, p)
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
		ui.Text(c, tr("app.name")).Bold().Grow(1)
		ui.Text(c, tr("app.local")).FontSize(9).LetterSpacing(0.6).Font("JetBrains Mono").
			TextColor(p.mutedFg.color())
		ui.Row(c).Gap(5).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Width(7).Height(7).Background(p.ok.color())
			ui.Text(c, tr("app.online")).FontSize(11).TextColor(p.mutedFg.color())
		})
	})
}

func main() {
	a := &app{sidebar: 260, sideTab: "requests",
		run: runState{timeout: "30", follow: true, verify: true}}
	// Before Run: startLocale computes the shown locale with the languages
	// set at that point, so ours must be in before it does.
	mygo.App.SetLanguages("en", "zh-Hans")
	mygo.App.WhenReady(func() {
		initLanguages()
		a.exec = runner.New()
		dir, err := mygo.App.Path(mygo.PathUserData)
		if err == nil {
			a.st, err = store.Open(filepath.Join(dir, "reqbox.db"))
		}
		if err != nil {
			log.Printf("store: %v", err)
		}
		a.refresh()
		a.loadRunSettings()
		a.startMockGateway()

		a.win = mygo.NewWindow(mygo.WindowOptions{
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
