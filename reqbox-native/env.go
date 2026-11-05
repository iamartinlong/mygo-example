package main

import (
	"github.com/egoist/mygo/ui"
	"reqbox-desktop/pkg/store"
)

const envManageKey = "\x00manage"

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
// "No environment", every environment, and the manager entry.
func (a *app) envRow(c *ui.Context, p palette) {
	sel := ui.SelectBase(c, &a.envChoice)
	sel.Trigger.Grow(1).Height(28).Padding(0, 6, 0, 6).Gap(8).
		AlignItems(ui.Center).Children(func() {
		ui.Text(c, a.envLabel()).FontSize(12).Grow(1)
		ui.Icon(c, iconChevronDown).TextColor(p.mutedFg.color())
	})
	sel.Popup(func(panel ui.Element) {
		panel.MinWidth(200).Padding(4).Background(p.card.color()).
			Radius(6).Border(1, p.border.color())
		it := sel.Item("").Padding(6, 10)
		it.Children(func() { ui.Text(c, tr("sidebar.noEnvironmentPlaceholder")).FontSize(12) })
		for _, e := range a.envs {
			it = sel.Item(e.ID).Padding(6, 10)
			it.Children(func(name string) func() {
				return func() { ui.Text(c, name).FontSize(12) }
			}(e.Name))
		}
		it = sel.Item(envManageKey).Padding(6, 10)
		it.Children(func() {
			ui.Text(c, tr("sidebar.manageEnv")).FontSize(12).
				TextColor(p.accent.color())
		})
	})
	if a.envChoice != a.activeEnv && a.envChoice != "" {
		if a.envChoice == envManageKey {
			a.envMgr = true
		} else {
			a.setActiveEnv(a.envChoice)
		}
	}
	if a.envChoice == envManageKey {
		a.envChoice = a.activeEnv
	}
}

// envDialog is the Manage environments modal: a list of environments or,
// when one is selected, its name and variable editor.
func (a *app) envDialog(c *ui.Context, p palette) {
	ui.Modal(c, &a.envMgr, func() {
		if a.envEditID == "" {
			a.envListPane(c, p)
			return
		}
		a.envEditPane(c, p)
	})
}

func (a *app) envListPane(c *ui.Context, p palette) {
	ui.Column(c).Width(440).Padding(16, 20, 16, 20).Gap(10).
		Background(p.card.color()).Radius(8).Children(func() {
		ui.Row(c).Height(30).AlignItems(ui.Center).Children(func() {
			ui.Text(c, tr("env.title")).Bold().Grow(1)
			e := btnSecondary(c, p, tr("common.close"))
			if e.Clicked() {
				a.envMgr = false
			}
		})
		ui.Text(c, tr("env.footer")).
			FontSize(11.5).TextColor(p.mutedFg.color())
		if len(a.envs) == 0 {
			ui.Text(c, tr("env.noEnvironments")).FontSize(12).
				TextColor(p.mutedFg.color())
		}
		for _, env := range a.envs {
			env := env
			ui.Row(c).Height(38).Gap(10).AlignItems(ui.Center).Children(func() {
				e := ui.Row(c).Grow(1).Height(38).Padding(0, 12, 0, 12).Gap(10).
					AlignItems(ui.Center).Background(p.secondary.color()).
					Radius(5).Children(func() {
					ui.Text(c, env.Name).FontSize(12.5).Grow(1)
					if env.ID == a.activeEnv {
						ui.Row(c).Height(15).Padding(0, 4, 0, 4).
							AlignItems(ui.Center).
							Background(p.ok.color().Alpha(0.15)).Radius(3).
							Children(func() {
								ui.Text(c, tr("env.activeBadge")).FontSize(9).
									Font("JetBrains Mono").
									TextColor(p.ok.color())
							})
					}
				})
				if e.Clicked() {
					a.openEnvEdit(env.ID)
				}
				e = ui.Text(c, tr("common.delete")).FontSize(11.5).
					TextColor(p.bad.color())
				if e.Clicked() {
					title := tr("confirm.deleteEnvironment.title", "name", env.Name)
					desc := tr("confirm.deleteEnvironment.desc")
					if env.ID == a.activeEnv {
						desc = tr("confirm.deleteEnvironment.descActive")
					}
					id := env.ID
					a.askConfirm(title, desc,
						tr("confirm.deleteEnvironment.confirm"), func() { a.deleteEnv(id) })
				}
			})
		}
		e := btnPrimary(c, p, tr("env.newEnvironment"))
		if e.Clicked() {
			if env, err := a.st.CreateEnvironment(tr("env.newEnvironment"), nil); err == nil {
				a.loadEnvs()
				a.openEnvEdit(env.ID)
				a.toastOK(tr("toast.environmentCreated"))
			}
		}
	})
}

func (a *app) envEditPane(c *ui.Context, p palette) {
	ui.Column(c).Width(520).Padding(16, 20, 16, 20).Gap(10).
		Background(p.card.color()).Radius(8).Children(func() {
		ui.Row(c).Height(30).Gap(10).AlignItems(ui.Center).Children(func() {
			e := btnSecondary(c, p, "Back")
			if e.Clicked() {
				a.envEditID = ""
			}
			ui.TextInput(c, &a.envEditName).Grow(1).Height(28).FontSize(13).
				Placeholder(tr("env.namePlaceholder"))
		})
		ui.Text(c, tr("auth.variableHint")).FontSize(11.5).
			TextColor(p.mutedFg.color())
		for i := range a.envEditVars {
			i := i
			v := &a.envEditVars[i]
			ui.Row(c).Height(30).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.TextInput(c, &v.key).Width(140).Height(28).FontSize(12)
				in := ui.TextInput(c, &v.value).Grow(1).Height(28).FontSize(12)
				if v.secret {
					in = in.Password()
				}
				ui.Checkbox(c, &v.enabled, "")
				e := ui.Text(c, "secret").FontSize(11).
					TextColor(p.mutedFg.color())
				if e.Clicked() {
					v.secret = !v.secret
				}
				e = ui.Text(c, "×").FontSize(14).TextColor(p.bad.color())
				if e.Clicked() {
					a.envEditVars = append(a.envEditVars[:i], a.envEditVars[i+1:]...)
				}
			})
		}
		if len(a.envEditVars) == 0 {
			ui.Text(c, tr("env.noVariables")).FontSize(11.5).
				TextColor(p.mutedFg.color())
		}
		e := btnSecondary(c, p, tr("env.addVariable"))
		if e.Clicked() {
			a.envEditVars = append(a.envEditVars, envVarEdit{enabled: true})
		}
		ui.Row(c).Height(32).Gap(10).AlignItems(ui.Center).Children(func() {
			e := ui.Text(c, tr("env.deleteEnv")).FontSize(12).
				TextColor(p.bad.color())
			if e.Clicked() {
				id := a.envEditID
				desc := tr("confirm.deleteEnvironment.desc")
				if id == a.activeEnv {
					desc = tr("confirm.deleteEnvironment.descActive")
				}
				a.askConfirm(tr("confirm.deleteEnvironment.title", "name", a.envEditName),
					desc, tr("confirm.deleteEnvironment.confirm"), func() {
						a.deleteEnv(id)
						a.envMgr = false
					})
			}
			ui.Row(c).Grow(1)
			e = btnPrimary(c, p, tr("common.save"))
			if e.Clicked() {
				a.saveEnvEdit()
			}
		})
	})
}

func (a *app) openEnvEdit(id string) {
	a.envEditID = id
	a.envEditName = ""
	a.envEditVars = nil
	if env, err := a.st.GetEnvironment(id); err == nil {
		a.envEditName = env.Name
		for _, v := range env.Variables {
			a.envEditVars = append(a.envEditVars, envVarEdit{
				key: v.Key, value: v.Value, enabled: v.Enabled, secret: v.Secret,
			})
		}
	}
}

func (a *app) saveEnvEdit() {
	vars := make([]store.EnvVar, 0, len(a.envEditVars))
	for _, v := range a.envEditVars {
		if v.key == "" {
			continue
		}
		vars = append(vars, store.EnvVar{
			Key: v.key, Value: v.value, Enabled: v.enabled, Secret: v.secret,
		})
	}
	if _, err := a.st.UpdateEnvironment(a.envEditID, a.envEditName, vars); err == nil {
		a.loadEnvs()
		a.envEditID = ""
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

// btnSecondary is a small neutral button, as the web's ghost buttons.
func btnSecondary(c *ui.Context, p palette, label string) ui.Element {
	return ui.Row(c).Height(28).Padding(0, 12, 0, 12).
		AlignItems(ui.Center).Background(p.secondary.color()).Children(func() {
		ui.Text(c, label).FontSize(12).SingleLine().
			TextColor(p.secondaryFg.color())
	})
}

func btnPrimary(c *ui.Context, p palette, label string) ui.Element {
	return ui.Row(c).Height(28).Padding(0, 14, 0, 14).
		AlignItems(ui.Center).Background(p.foreground.color()).Children(func() {
		ui.Text(c, label).FontSize(12).SingleLine().
			TextColor(p.primaryFg.color())
	})
}
