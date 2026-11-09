package main

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"reqbox-desktop/pkg/store"
)

// envVarEdit is one variable row in the environment editor.
type envVarEdit struct {
	key, value string
	enabled    bool
	secret     bool
}

func (a *app) loadEnvs() {
	if a.st == nil {
		return
	}
	a.envs, _ = a.st.ListEnvironments()
	a.activeEnv = a.st.GetSetting("activeEnvironmentId", "")
}

func (a *app) envLabel() string {
	for _, e := range a.envs {
		if e.ID == a.activeEnv {
			return e.Name
		}
	}
	return tr("sidebar.noEnvironmentPlaceholder")
}

// enabledVars counts as the web's picker badge does: enabled, key ignored.
func enabledVars(env store.Environment) int {
	n := 0
	for _, v := range env.Variables {
		if v.Enabled {
			n++
		}
	}
	return n
}

func (a *app) setActiveEnv(id string) {
	a.activeEnv = id
	if a.st != nil {
		_ = a.st.SetSetting("activeEnvironmentId", id)
	}
}

// resolveEnvVars collects the active environment's enabled variables; the
// empty map means requests run with their literal values.
func (a *app) resolveEnvVars() map[string]string {
	vars := map[string]string{}
	if a.st == nil || a.activeEnv == "" {
		return vars
	}
	if env, err := a.st.GetEnvironment(a.activeEnv); err == nil {
		for _, v := range env.Variables {
			if v.Enabled && v.Key != "" {
				vars[v.Key] = v.Value
			}
		}
	}
	return vars
}

// envRow is the sidebar environment picker: a select whose popup lists
// "No environment" and every environment, plus the web's separate manage
// button on its right.
func (a *app) envRow(c *ui.Context, p palette) {
	sel := ui.SelectBase(c, &a.envChoice)
	sel.Trigger.Grow(1).Height(28).Padding(0, 6, 0, 6).Gap(8).Radius(6).
		AlignItems(ui.Center).Children(func() {
		ui.Text(c, a.envLabel()).FontSize(12).FontWeight(500).Grow(1)
		ui.Icon(c, iconChevronDown).TextColor(p.mutedFg.color())
	})
	// The web's ghost icon-xs button (24px) that opens the manager directly.
	mgr := ui.Row(c).Width(24).Height(24).AlignItems(ui.Center).Center().
		Children(func() {
			ui.Icon(c, iconSettings2).Size(14, 14).TextColor(p.mutedFg.color())
		})
	if mgr.Clicked() {
		a.openEnvMgr()
	}
	sel.Popup(func(panel ui.Element) {
		panel.MinWidth(200).Padding(4).Background(p.card.color()).
			Radius(6).Border(1, p.border.color())
		it := sel.Item("").Padding(6, 10)
		it.Children(func() { ui.Text(c, tr("sidebar.noEnvironmentPlaceholder")).FontSize(12) })
		for _, e := range a.envs {
			it = sel.Item(e.ID).Padding(6, 10)
			it.Children(func(env store.Environment) func() {
				return func() {
					ui.Text(c, env.Name).FontSize(12).Grow(1)
					// The web's ml-auto count of the environment's variables.
					ui.Text(c, fmt.Sprint(enabledVars(env))).FontSize(10).
						Font("JetBrains Mono").SingleLine().
						TextColor(p.mutedFg.color())
				}
			}(e))
		}
	})
	if a.envChoice != a.activeEnv && a.envChoice != "" {
		a.setActiveEnv(a.envChoice)
	}
}

// envDialog is the web's _F: a 768px manager with a header band, a 190px
// environment list on a muted ground, and the name and variable editor of
// the selected environment, on a band footer.
func (a *app) envDialog(c *ui.Context, p palette) {
	ui.Modal(c, &a.envMgr, func() {
		ui.Column(c).Width(768).Clip().Background(p.popover.color()).Radius(12).
			Border(1, p.foreground.color().Alpha(0.1)).Children(func() {
			// Header band: title, subtitle, and the new-environment button.
			ui.Row(c).FillWidth().Padding(12, 16, 12, 16).Gap(8).
				AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).
				BorderColor(p.border.color()).Children(func() {
				ui.Text(c, tr("env.title")).FontSize(14).FontWeight(500)
				ui.Text(c, tr("env.subtitle")).FontSize(11).
					TextColor(p.mutedFg.color())
				ui.Row(c).Grow(1)
				a.btnNewEnv(c, p)
			})
			// Body: the fixed-height two-pane grid.
			ui.Row(c).FillWidth().Height(420).Children(func() {
				a.envListPane(c, p)
				a.envEditPane(c, p)
			})
			// Band footer: the hint at the left, Done at the right.
			ui.Row(c).FillWidth().Padding(10, 16, 10, 16).Gap(8).
				AlignItems(ui.Center).Background(p.muted.color().Alpha(0.5)).
				BorderWidth(1, 0, 0, 0).BorderColor(p.border.color()).
				Children(func() {
					ui.Text(c, tr("env.footer")).FontSize(10.5).Grow(1).
						TextColor(p.mutedFg.color())
					btnOutlineSm(c, p, tr("common.done"), func() { a.envMgr = false })
				})
		})
	})
}

// btnNewEnv starts a fresh, unsaved environment in the editor, as the
// web's S does.
func (a *app) btnNewEnv(c *ui.Context, p palette) {
	border, bg := p.border.color(), p.background.color()
	if p.isDark {
		border, bg = p.input.color(), p.input.color().Alpha(0.3)
	}
	e := ui.Row(c).Height(24).Padding(0, 8, 0, 8).Gap(4).Radius(6).
		AlignItems(ui.Center).Border(1, border).Background(bg)
	if e.Hovered() {
		if p.isDark {
			e.Background(p.input.color().Alpha(0.5))
		} else {
			e.Background(p.muted.color())
		}
	}
	e.Children(func() {
		ui.Icon(c, iconPlus).Size(12, 12)
		ui.Text(c, tr("env.new")).FontSize(12)
	})
	if e.Clicked() {
		a.envEditID = ""
		a.envEditName = tr("env.newEnvironment")
		a.envEditVars = []envVarEdit{{enabled: true}}
	}
}

// envListPane is the left rail of environments; the selected one gets the
// paper pill, the active one the ok badge.
func (a *app) envListPane(c *ui.Context, p palette) {
	ui.Column(c).Width(190).FillHeight().Background(p.muted.color().Alpha(0.3)).
		BorderWidth(0, 1, 0, 0).BorderColor(p.border.color()).Padding(8, 8, 8, 8).
		Children(func() {
			if len(a.envs) == 0 && a.envEditID == "" {
				ui.Row(c).FillWidth().Padding(8, 8, 12, 8).Children(func() {
					ui.Text(c, tr("env.noEnvironments")).FontSize(11).
						TextColor(p.mutedFg.color())
				})
			}
			for _, env := range a.envs {
				env := env
				sel := a.envEditID == env.ID
				b := ui.Row(c).FillWidth().Padding(6, 8, 6, 8).Gap(6).
					AlignItems(ui.Center).Radius(6)
				if sel {
					b.Background(p.background.color())
				} else if b.Hovered() {
					b.Background(p.accent.color().Alpha(0.6))
				}
				b.Children(func() {
					t := ui.Text(c, env.Name).FontSize(12).SingleLine().Grow(1)
					if sel {
						t.TextColor(p.foreground.color())
					} else {
						t.TextColor(p.mutedFg.color())
					}
					if env.ID == a.activeEnv {
						ui.Row(c).Height(15).Padding(0, 4, 0, 4).Radius(4).
							Background(p.ok.color().Alpha(0.15)).AlignItems(ui.Center).
							Children(func() {
								ui.Text(c, tr("env.activeBadge")).FontSize(9).
									Font("JetBrains Mono").TextColor(p.ok.color())
							})
					}
				})
				if b.Clicked() {
					a.loadEnvEdit(env.ID)
				}
			}
		})
}

// envEditPane is the right side: the name row with its actions, then the
// variable rows on a scrolling ground.
func (a *app) envEditPane(c *ui.Context, p palette) {
	ui.Column(c).Grow(1).FillHeight().Children(func() {
		ui.Row(c).FillWidth().Padding(8, 12, 8, 12).Gap(8).AlignItems(ui.Center).
			BorderWidth(0, 0, 1, 0).BorderColor(p.border.color()).Children(func() {
			ui.TextInputBase(c, &a.envEditName).Width(220).Height(28).Radius(8).
				FontSize(12).FontWeight(500).Padding(0, 10, 0, 10).
				Border(1, p.input.color()).Placeholder(tr("env.namePlaceholder"))
			if a.envEditID != "" && a.envEditID != a.activeEnv {
				a.btnUseThisEnv(c, p)
			}
			ui.Row(c).Grow(1)
			if a.envEditID != "" {
				del := ui.Row(c).Width(24).Height(24).Radius(6).
					AlignItems(ui.Center).Center()
				if del.Hovered() {
					del.Background(p.muted.color())
				}
				del.Children(func() {
					ui.Icon(c, iconTrash).Size(14, 14).TextColor(p.bad.color())
				})
				if del.Clicked() {
					id := a.envEditID
					desc := tr("confirm.deleteEnvironment.desc")
					if id == a.activeEnv {
						desc = tr("confirm.deleteEnvironment.descActive")
					}
					a.askConfirm(tr("confirm.deleteEnvironment.title", "name", a.envEditName),
						desc, tr("confirm.deleteEnvironment.confirm"), func() {
							a.deleteEnv(id)
						})
				}
				btnDefaultXsSave(c, p, func() { a.saveEnv() })
			}
		})
		ui.Scroll(c).FillHeight().FillWidth().Children(func() {
			ui.Column(c).FillWidth().Padding(12, 12, 12, 12).Gap(4).Children(func() {
				if len(a.envEditVars) == 0 {
					ui.Row(c).FillWidth().Padding(24, 0, 24, 0).Justify(ui.Center).
						Children(func() {
							ui.Text(c, tr("env.noVariables")).FontSize(11).
								TextColor(p.mutedFg.color())
						})
					return
				}
				for i := range a.envEditVars {
					a.envVarRow(c, p, i)
				}
				ui.Row(c).Padding(8, 0, 0, 0).Children(func() {
					add := ui.Row(c).Height(24).Padding(0, 8, 0, 8).Gap(4).Radius(6).
						AlignItems(ui.Center)
					ink := p.mutedFg.color()
					if add.Hovered() {
						add.Background(p.muted.color())
						ink = p.foreground.color()
					}
					add.Children(func() {
						ui.Icon(c, iconPlus).Size(12, 12).TextColor(ink)
						ui.Text(c, tr("env.addVariable")).FontSize(12).TextColor(ink)
					})
					if add.Clicked() {
						a.envEditVars = append(a.envEditVars, envVarEdit{enabled: true})
					}
				})
			})
		})
	})
}

// envVarRow is one variable line: enable check, key, value (masked when
// secret) and a remove button, on the web's 24/1fr/1.4fr/48 grid.
func (a *app) envVarRow(c *ui.Context, p palette, i int) {
	v := &a.envEditVars[i]
	ui.Row(c).FillWidth().Height(28).Gap(6).AlignItems(ui.Center).Children(func() {
		ui.Row(c).Width(24).AlignItems(ui.Center).Children(func() {
			webCheckbox(c, p, &v.enabled)
		})
		ui.TextInputBase(c, &v.key).Grow(1).Basis(0).MinWidth(0).
			Height(28).Radius(8).FontSize(11.5).Font("JetBrains Mono").
			Padding(0, 10, 0, 10).Border(1, p.input.color()).
			Placeholder("baseUrl")
		vin := ui.TextInputBase(c, &v.value).Grow(1.4).Basis(0).MinWidth(0).
			Height(28).Radius(8).FontSize(11.5).Font("JetBrains Mono").
			Padding(0, 10, 0, 10).Border(1, p.input.color()).
			Placeholder(tr("env.baseUrlPlaceholder"))
		if v.secret {
			vin = vin.Password()
		}
		rm := ui.Row(c).Width(24).Height(24).Radius(6).
			AlignItems(ui.Center).Center()
		if rm.Hovered() {
			rm.Background(p.muted.color())
		}
		rm.Children(func() {
			ui.Icon(c, iconTrash).Size(12, 12).TextColor(p.mutedFg.color())
		})
		if rm.Clicked() {
			a.envEditVars = append(a.envEditVars[:i], a.envEditVars[i+1:]...)
		}
	})
}

// btnUseThisEnv activates the edited environment, as the web's w does.
func (a *app) btnUseThisEnv(c *ui.Context, p palette) {
	border, bg := p.border.color(), p.background.color()
	if p.isDark {
		border, bg = p.input.color(), p.input.color().Alpha(0.3)
	}
	e := ui.Row(c).Height(24).Padding(0, 8, 0, 8).Gap(4).Radius(6).
		AlignItems(ui.Center).Border(1, border).Background(bg)
	if e.Hovered() {
		if p.isDark {
			e.Background(p.input.color().Alpha(0.5))
		} else {
			e.Background(p.muted.color())
		}
	}
	e.Children(func() {
		ui.Text(c, tr("env.useThis")).FontSize(12)
	})
	if e.Clicked() {
		a.setActiveEnv(a.envEditID)
		a.toastOK(tr("toast.usingEnvironment", "name", a.envEditName))
	}
}

// btnDefaultXsSave is the web's default xs button with the save glyph.
func btnDefaultXsSave(c *ui.Context, p palette, fn func()) {
	e := ui.Row(c).Height(24).Padding(0, 8, 0, 8).Gap(4).Radius(6).
		AlignItems(ui.Center)
	if e.Hovered() {
		e.Background(p.primary.color().Alpha(0.8))
	}
	e.Children(func() {
		ui.Icon(c, iconSave).Size(12, 12).TextColor(p.primaryFg.color())
		ui.Text(c, tr("common.save")).FontSize(12).TextColor(p.primaryFg.color())
	})
	if e.Clicked() && fn != nil {
		fn()
	}
}

// openEnvMgr opens the manager with the web's selection fallback: the
// last-edited environment, else the active one, else the first.
func (a *app) openEnvMgr() {
	a.envMgr = true
	found := false
	for _, env := range a.envs {
		if env.ID == a.envEditID {
			found = true
		}
	}
	if !found {
		id := a.activeEnv
		if id == "" && len(a.envs) > 0 {
			id = a.envs[0].ID
		}
		a.envEditID = id
	}
	a.loadEnvEdit(a.envEditID)
}

// loadEnvEdit fills the editor with an environment, or empties it for the
// unsaved new one.
func (a *app) loadEnvEdit(id string) {
	a.envEditID = id
	a.envEditName = ""
	a.envEditVars = nil
	if id == "" {
		return
	}
	if env, err := a.st.GetEnvironment(id); err == nil {
		a.envEditName = env.Name
		for _, v := range env.Variables {
			a.envEditVars = append(a.envEditVars, envVarEdit{
				key: v.Key, value: v.Value, enabled: v.Enabled, secret: v.Secret,
			})
		}
	}
}

// saveEnv writes the editor back: an update for an existing environment,
// a create for the unsaved new one. The dialog stays open, as on the web.
func (a *app) saveEnv() {
	vars := make([]store.EnvVar, 0, len(a.envEditVars))
	for _, v := range a.envEditVars {
		if v.key == "" {
			continue
		}
		vars = append(vars, store.EnvVar{
			Key: v.key, Value: v.value, Enabled: v.enabled, Secret: v.secret,
		})
	}
	name := a.envEditName
	if name == "" {
		name = tr("env.untitled")
	}
	if a.envEditID == "" {
		if env, err := a.st.CreateEnvironment(name, vars); err == nil {
			a.envEditID = env.ID
			a.loadEnvs()
			a.toastOK(tr("toast.environmentCreated"))
		}
		return
	}
	if _, err := a.st.UpdateEnvironment(a.envEditID, name, vars); err == nil {
		a.loadEnvs()
		a.toastOK(tr("toast.environmentSaved"))
	}
}

func (a *app) deleteEnv(id string) {
	_ = a.st.DeleteEnvironment(id)
	if a.activeEnv == id {
		a.setActiveEnv("")
	}
	a.envEditID = ""
	a.loadEnvs()
	a.toastOK(tr("toast.environmentDeleted"))
}
