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
	dark    bool    // follows the web .dark variant, toggled from the top bar
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
	// methodPick is the segmented box's method select binding; the box
	// keeps no look, so SelectBase needs the value it shows.
	methodPick string
	bodyBuf    *editor.Buffer
	bodyState  *editor.State

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
	langSel     string

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

// topBar mirrors the web chrome: Mark + wordmark + LOCAL badge left,
// health, the language menu and the theme toggle right.
func (a *app) topBar(c *ui.Context, p palette) {
	bar := c.TitleBar()
	ui.Column(c).Children(func() {
		// The window controls float over the right of this band; the band
		// drags the window, its buttons keep their own clicks.
		ui.Row(c).Height(max(bar.Height, 40)).Padding(0, bar.Right+12, 0, 12).
			Gap(12).AlignItems(ui.Center).Background(p.chrome.color()).DragWindow().
			Children(func() {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, iconMark).Size(18, 18).
						TextColor(p.foreground.color())
					ui.Text(c, tr("app.name")).FontSize(13).Bold().
						SingleLine()
					badge(c, p, tr("app.local"))
				})
				ui.Row(c).Grow(1)
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
						ui.Column(c).Width(6).Height(6).Radius(3).
							Background(p.ok.color())
						ui.Text(c, tr("app.online")).FontSize(10.5).
							TextColor(p.mutedFg.color())
					})
					a.langMenu(c, p)
					a.themeBtn(c, p)
				})
			})
		ui.Row(c).Height(1).Fill().Background(p.hairline.color())
	})
}

// badge is the web's LOCAL chip: mono, wide-tracked, hairline border.
func badge(c *ui.Context, p palette, label string) {
	ui.Row(c).Height(16).Padding(0, 4, 0, 4).AlignItems(ui.Center).
		Border(1, p.border.color()).Radius(4).Children(func() {
		ui.Text(c, label).FontSize(9.5).LetterSpacing(1.3).
			Font("JetBrains Mono").TextColor(p.mutedFg.color())
	})
}

// langMenu is the web header's language dropdown: English and 中文,
// radio-style, applied at once and remembered.
func (a *app) langMenu(c *ui.Context, p palette) {
	a.langSel = appLang
	sel := ui.SelectBase(c, &a.langSel)
	sel.Trigger.Width(28).Height(28).Radius(6).Center().
		Children(func() {
			ui.Icon(c, iconLanguages).Size(14, 14).
				TextColor(p.mutedFg.color())
		})
	sel.Popup(func(panel ui.Element) {
		panel.MinWidth(140).Padding(4).Background(p.card.color()).
			Radius(6).Border(1, p.border.color())
		for _, l := range []struct{ id, label string }{
			{"en", tr("app.language.en")}, {"zh", tr("app.language.zh")},
		} {
			it := sel.Item(l.id).Padding(6, 10)
			it.Children(func(label string) func() {
				return func() { ui.Text(c, label).FontSize(12) }
			}(l.label))
		}
	})
	if a.langSel != appLang {
		setLang(a.st, a.langSel)
		a.langSel = appLang
		// Group headers hold translated text, so they need re-deriving.
		a.rebuildRows()
	}
}

// themeBtn flips between the web's two variants; the sun shows in dark
// (it would move you to light) and the moon in light, as on the web.
func (a *app) themeBtn(c *ui.Context, p palette) {
	icon := iconMoon
	if a.dark {
		icon = iconSun
	}
	e := ui.Row(c).Width(28).Height(28).Radius(6).Center().Children(func() {
		ui.Icon(c, icon).Size(14, 14).TextColor(p.mutedFg.color())
	})
	if e.Clicked() {
		a.dark = !a.dark
		a.applyButtonsTheme()
		if a.st != nil {
			_ = a.st.SetSetting(setTheme, themeName(a.dark))
		}
	}
}

// applyButtonsTheme repaints the floating window controls for the variant
// they sit on: ThemeDark draws light glyphs. Windows-only; inert elsewhere.
func (a *app) applyButtonsTheme() {
	if a.win == nil {
		return
	}
	if a.dark {
		a.win.SetWindowButtonsTheme(mygo.ThemeDark)
	} else {
		a.win.SetWindowButtonsTheme(mygo.ThemeLight)
	}
}

func themeName(dark bool) string {
	if dark {
		return "dark"
	}
	return "light"
}

const setTheme = "theme"

// loadTheme reads the stored variant. The web starts dark unless the
// system asks for light; mygo exposes no prefers-color-scheme, so dark
// is the starting point here and the first toggle is what gets remembered.
func (a *app) loadTheme() {
	a.dark = true
	if a.st == nil {
		return
	}
	switch a.st.GetSetting(setTheme, "") {
	case "light":
		a.dark = false
	case "dark":
		a.dark = true
	}
}

func main() {
	a := &app{sidebar: 290, sideTab: "requests",
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
		// Before the first refresh: it caches translated group headers.
		applyLangSetting(a.st)
		a.refresh()
		a.loadRunSettings()
		a.loadTheme()
		a.startMockGateway()

		a.win = mygo.NewWindow(mygo.WindowOptions{
			Title:     "Reqbox",
			Width:     1280,
			Height:    820,
			MinWidth:  960,
			MinHeight: 600,
			// One bar instead of two: the system title bar goes, the window
			// controls float over our chrome band, and unlike Frameless
			// this keeps the shadow, corners and resize edges.
			TitleBarStyle:  mygo.TitleBarHidden,
			TitleBarHeight: 40,
			Content:        ui.View(a.view),
		})
		a.applyButtonsTheme()
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
