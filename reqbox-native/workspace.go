package main

import (
	"encoding/json"
	"os"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"reqbox-desktop/pkg/store"
)

// The sidebar footer, as the web's: env label, then import, export and
// run-settings icon buttons. Import/export move the whole workspace
// through store Snap/Restore, the format the web's import/export uses.

func (a *app) footerRow(c *ui.Context, p palette) {
	ui.Row(c).Height(37).Padding(6, 8, 6, 8).Gap(4).
		BorderWidth(1, 0, 0, 0).BorderColor(p.hairline.color()).
		AlignItems(ui.Center).Children(func() {
		label := tr("sidebar.noEnvironment")
		if e := a.envByID(a.activeEnv); e != nil {
			label = tr("sidebar.envPrefix") + " · " + e.Name
		}
		ui.Text(c, label).FontSize(10.5).Font("JetBrains Mono").
			SingleLine().Grow(1).TextColor(p.mutedFg.color())
		a.footerBtn(c, p, iconUpload, func() { a.pickImport() })
		a.footerBtn(c, p, iconDownload, func() { a.pickExport() })
		a.footerBtn(c, p, iconSettings2, func() { a.runDlg = true })
	})
}

func (a *app) footerBtn(c *ui.Context, p palette, icon *ui.SVG, fn func()) {
	e := ui.Row(c).Width(24).Height(24).AlignItems(ui.Center).Center().
		Children(func() {
			ui.Icon(c, icon).Size(12, 12).TextColor(p.mutedFg.color())
		})
	if e.Clicked() {
		fn()
	}
}

func (a *app) envByID(id string) *store.Environment {
	for i := range a.envs {
		if a.envs[i].ID == id {
			return &a.envs[i]
		}
	}
	return nil
}

func (a *app) pickExport() {
	if a.st == nil {
		return
	}
	go func() {
		path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
			Title:       tr("sidebar.export"),
			DefaultPath: "reqbox-workspace.json",
			Filters:     []mygo.FileFilter{{Name: "JSON", Extensions: []string{".json"}}},
		})
		if err != nil || path == "" {
			return
		}
		snap, err := a.st.Snap()
		if err == nil {
			var raw []byte
			raw, err = json.MarshalIndent(snap, "", "  ")
			if err == nil {
				err = os.WriteFile(path, raw, 0o644)
			}
		}
		a.win.Update(func() {
			if err != nil {
				a.toastErr(tr("toast.exportFailed"))
			} else {
				a.toastOK(tr("toast.workspaceExported"))
			}
		})
	}()
}

func (a *app) pickImport() {
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Title:   tr("sidebar.import"),
			Filters: []mygo.FileFilter{{Name: "JSON", Extensions: []string{".json"}}},
		})
		if err != nil || len(paths) == 0 {
			return
		}
		raw, err := os.ReadFile(paths[0])
		var snap store.Export
		if err == nil {
			err = json.Unmarshal(raw, &snap)
		}
		if err != nil {
			a.win.Update(func() { a.toastErr(tr("toast.invalidExport")) })
			return
		}
		count := len(snap.Requests) + len(snap.Collections) + len(snap.Environments)
		a.win.Update(func() {
			desc := tr("confirm.import.desc", "count", count)
			if count == 0 {
				desc = tr("confirm.import.descEmpty")
			}
			a.askConfirm(tr("confirm.import.title"), desc,
				tr("confirm.import.confirm"), func() { a.doImport(snap) })
		})
	}()
}

func (a *app) doImport(snap store.Export) {
	if a.st == nil {
		return
	}
	if err := a.st.Restore(snap); err != nil {
		a.toastErr(tr("toast.importFailed"))
		return
	}
	a.selectedID = ""
	a.refresh()
	a.toastOK(tr("toast.imported", "count", len(snap.Requests)))
}

// runSettingsDialog edits the run defaults live, as the web's yF dialog:
// changes apply at once, Done only closes. They persist in the settings
// table, the desktop counterpart of the web's localStorage.
func (a *app) runSettingsDialog(c *ui.Context, p palette) {
	ui.Modal(c, &a.runDlg, func() {
		dialogShell(c, p, 384, func() {
			ui.Column(c).FillWidth().Gap(8).Children(func() {
				ui.Text(c, tr("run.title")).FontSize(14).FontWeight(500)
				ui.Text(c, tr("run.desc")).FontSize(11).
					TextColor(p.mutedFg.color())
			})
			ui.Column(c).FillWidth().Gap(12).Children(func() {
				ui.Row(c).FillWidth().Gap(12).AlignItems(ui.Center).Children(func() {
					ui.Text(c, tr("run.timeout")).FontSize(12).Grow(1)
					timeoutIn := ui.TextInput(c, &a.run.timeout).Width(80).Height(28).
						FontSize(12).Font("JetBrains Mono")
					focusedRing(p, timeoutIn)
					ui.Text(c, tr("run.seconds")).FontSize(11).
						TextColor(p.mutedFg.color())
				})
				ui.Row(c).FillWidth().Gap(12).AlignItems(ui.Center).Children(func() {
					ui.Text(c, tr("run.followRedirects")).FontSize(12).Grow(1)
					webCheckbox(c, p, &a.run.follow)
				})
				ui.Row(c).FillWidth().Gap(12).AlignItems(ui.Start).Children(func() {
					ui.Column(c).Grow(1).Gap(2).Children(func() {
						ui.Text(c, tr("run.verifyTls")).FontSize(12)
						ui.Text(c, tr("run.verifyTlsHint")).FontSize(10.5).
							TextColor(p.mutedFg.color())
					})
					ui.Row(c).Padding(2, 0, 0, 0).Children(func() {
						webCheckbox(c, p, &a.run.verify)
					})
				})
			})
		}, func() {
			// The web's band footer: muted ground, hairline top, Done at
			// the right.
			ui.Row(c).FillWidth().Padding(16, 16, 16, 16).Gap(8).
				Justify(ui.End).Background(p.muted.color().Alpha(0.5)).
				BorderWidth(1, 0, 0, 0).BorderColor(p.border.color()).
				AlignItems(ui.Center).Children(func() {
				btnPrimarySm(c, p, tr("common.done"), func() {
					a.runDlg = false
					a.saveRunSettings()
				})
			})
		})
	})
}

const (
	setRunTimeout = "runTimeout"
	setRunFollow  = "runFollowRedirects"
	setRunVerify  = "runVerifyTls"
)

// loadRunSettings seeds the run bar from the settings table; the typed
// timeout keeps the web's 30s fallback when absent or malformed.
func (a *app) loadRunSettings() {
	if a.st == nil {
		return
	}
	if v := a.st.GetSetting(setRunTimeout, ""); v != "" {
		a.run.timeout = v
	}
	a.run.follow = a.st.GetSetting(setRunFollow, "1") == "1"
	a.run.verify = a.st.GetSetting(setRunVerify, "1") == "1"
}

func (a *app) saveRunSettings() {
	if a.st == nil {
		return
	}
	follow, verify := "0", "0"
	if a.run.follow {
		follow = "1"
	}
	if a.run.verify {
		verify = "1"
	}
	_ = a.st.SetSetting(setRunTimeout, a.run.timeout)
	_ = a.st.SetSetting(setRunFollow, follow)
	_ = a.st.SetSetting(setRunVerify, verify)
}
