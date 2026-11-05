// Todo is a desktop todo app in native UI: MyGo draws the window itself
// from Go, with no web page, no HTML and no JavaScript.
//
//	go run .
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// appState is the state the window shows.
type appState struct {
	store     *Todos
	draft     string
	filter    int // 0 all, 1 active, 2 done
	list      ui.ListState
	selected  int // chosen row, -1 for none; EditableText in a row reads it
	dueOpen   bool      // the due-date popover of the row dueFor is open
	dueFor    int64     // todo whose due-date picker is open, 0 for none
	dueDraft  time.Time // the date the open calendar edits
	updateMsg string
	crashed   bool // crashDrill fired once
}

// filtered returns the todos the filter shows, done last.
func (a *appState) filtered() []Todo {
	all := a.store.Snapshot()
	if a.filter == 0 {
		return all
	}
	want := a.filter == 2
	out := make([]Todo, 0, len(all))
	for _, it := range all {
		if it.Done == want {
			out = append(out, it)
		}
	}
	return out
}

// view builds the interface from the state, on the main thread, whenever the
// window needs a frame.
func (a *appState) view(c *ui.Context) {
	defer recoverFrame(&mainWin)
	if crashDrill && !a.crashed {
		a.crashed = true
		panic("crash-guard drill")
	}
	t := c.Theme()
	s := a.store.Stats()
	ui.Column(c).Fill().Children(func() {
		// Header.
		ui.Row(c).Padding(16, 16, 8, 16).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Todos").FontSize(20).Bold().Grow(1)
			ui.Textf(c, "%d active · %d done", s.Active, s.Done).FontSize(12).TextColor(t.TextMuted)
		})

		// New todo input.
		ui.Row(c).Padding(0, 16, 12, 16).Gap(8).Children(func() {
			if ui.TextInput(c, &a.draft).Placeholder("What needs doing?").AutoFocus().Submitted() {
				a.add()
			}
			if ui.PrimaryButton(c.Key("add"), "Add").Clicked() {
				a.add()
			}
		})

		// Filter.
		ui.Row(c).Padding(0, 16, 8, 16).Children(func() {
			ui.Segmented(c, &a.filter, "All", "Active", "Done")
		})

		// The list.
		items := a.filtered()
		ui.List(c, &a.list, len(items), func(i int) {
			it := items[i]
			ui.Row(c).Padding(6, 16).Gap(10).AlignItems(ui.Center).Children(func() {
				done := it.Done
				if ui.Checkbox(c, &done, "").Label(it.Title).Changed() {
					a.store.SetDone(it.ID, done)
				}
				title := it.Title
				if it.Done {
					ui.Text(c, title).Grow(1).TextColor(t.TextMuted)
				} else if ui.EditableText(c.Key(fmt.Sprintf("title-%d", it.ID)), &title).Changed() {
					a.store.SetTitle(it.ID, title)
				}
				a.dueUI(c, t, it)
				if ui.Button(c.Key(fmt.Sprintf("del-%d", it.ID)), "✕").Label("Delete").Clicked() {
					a.store.Remove(it.ID)
				}
			})
		}).Grow(1).Children(func() {
			ui.Text(c, "Nothing here yet").TextColor(t.TextMuted).Padding(24)
		})

		// Footer.
		total := "items"
		if s.Total == 1 {
			total = "item"
		}
		ui.Row(c).Padding(8, 16, 16, 16).AlignItems(ui.Center).Children(func() {
			ver := mygo.App.Version()
			if ver == "" {
				ver = "dev"
			}
			if a.updateMsg != "" {
				ui.Text(c, a.updateMsg).FontSize(11).TextColor(t.TextMuted).Grow(1)
			} else {
				ui.Textf(c, "%d %s · v%s", s.Total, total, ver).FontSize(12).TextColor(t.TextMuted).Grow(1)
			}
			if ui.Button(c.Key("clear"), "Clear done").Disabled(s.Done == 0).Clicked() {
				a.store.ClearDone()
			}
		})
	})
}

// dueUI shows a todo's due date (red when overdue) and the picker button
// whose popover holds a calendar. Choosing a day saves it at once.
func (a *appState) dueUI(c *ui.Context, t *ui.Theme, it Todo) {
	if d, ok := parseDue(it.Due); ok {
		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
		txt := ui.Text(c, d.Format("01-02")).FontSize(12)
		if !it.Done && d.Before(today) {
			txt.TextColor(t.Danger)
		} else {
			txt.TextColor(t.TextMuted)
		}
	}
	btn := ui.Button(c.Key(fmt.Sprintf("due-%d", it.ID)), "📅").Label("Due date")
	if btn.Clicked() {
		a.dueFor = it.ID
		a.dueDraft, _ = parseDue(it.Due)
		if a.dueDraft.IsZero() {
			a.dueDraft = time.Now()
		}
		a.dueOpen = true
	}
	if !a.dueOpen || a.dueFor != it.ID {
		return
	}
	ui.Popover(c, btn, &a.dueOpen, func() {
		prev := a.dueDraft
		ui.Calendar(c, &a.dueDraft)
		if !a.dueDraft.Equal(prev) {
			a.store.SetDue(it.ID, a.dueDraft.Format("2006-01-02"))
			a.dueOpen = false
		}
		ui.Row(c).Padding(8, 0, 0, 0).Gap(8).Children(func() {
			if ui.Button(c, "清除").Clicked() {
				a.store.SetDue(it.ID, "")
				a.dueOpen = false
			}
			if ui.PrimaryButton(c.Key("due-done"), "关闭").Clicked() {
				a.dueOpen = false
			}
		})
	})
}

// parseDue reads a stored due date as a local date.
func parseDue(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	d, err := time.ParseInLocation("2006-01-02", s, time.Local)
	return d, err == nil
}

func (a *appState) add() {
	if _, err := a.store.Add(a.draft); err == nil {
		a.draft = ""
	}
}

// checkUpdate checks, installs and relaunches, reporting progress into the
// window. Runs in a goroutine; state changes go through win.Update so they
// land on the main thread.
func (a *appState) checkUpdate(win *mygo.Window) {
	go func() {
		ctx := context.Background()
		set := func(msg string) {
			win.Update(func() { a.updateMsg = msg })
		}
		up, err := mygo.Updater.Check(ctx)
		if err != nil {
			set("更新检查失败: " + err.Error())
			return
		}
		if up == nil {
			set("已是最新版本 v" + mygo.App.Version())
			return
		}
		set("发现 v" + up.Version + "，下载安装中…")
		if err := up.Install(ctx, nil); err != nil {
			set("更新失败: " + err.Error())
			return
		}
		mygo.App.Relaunch()
	}()
}

// statsView draws the stats window. It redraws itself whenever a store
// change pushes a frame through statsWin.Update.
func (a *appState) statsView(c *ui.Context) {
	defer recoverFrame(&statsWin)
	t := c.Theme()
	s := a.store.Stats()
	ui.Column(c).Fill().Padding(20).Gap(12).Children(func() {
		ui.Text(c, "统计").FontSize(16).Bold()
		ui.Textf(c, "全部 %d", s.Total).FontSize(26).Bold()
		ui.Textf(c, "进行中 %d", s.Active).FontSize(26).Bold().TextColor(t.TextMuted)
		ui.Textf(c, "已完成 %d", s.Done).FontSize(26).Bold().TextColor(t.TextMuted)
	})
}

// mainWin is the window a second instance's launch brings to the front.
var mainWin *mygo.Window

// statsWin is the second window; it mirrors the store and refreshes
// through the store's change notification.
var statsWin *mygo.Window

func openStatsWin(a *appState) {
	if statsWin != nil {
		statsWin.Focus()
		return
	}
	statsWin = mygo.NewWindow(mygo.WindowOptions{
		Title:   "统计",
		Width:   220,
		Height:  260,
		Content: ui.View(a.statsView),
	})
}

func main() {
	if !mygo.App.RequestSingleInstanceLock() {
		return // the running instance takes over
	}
	store, err := OpenStore()
	if err != nil {
		log.Fatal(err)
	}
	a := &appState{store: store, selected: -1}
	a.list.Selected = &a.selected
	// A change saved from any thread pushes the stats window a new frame.
	store.OnChange(func() {
		if statsWin != nil {
			statsWin.Update(func() {})
		}
	})
	mygo.App.OnSecondInstance(func(args []string, workingDir string) {
		if mainWin != nil {
			mainWin.Restore()
			mainWin.Focus()
		}
	})
	mygo.App.WhenReady(func() {
		mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
			{Label: "文件", Submenu: []*mygo.MenuItem{
				{Label: "检查更新…", Accelerator: "CmdOrCtrl+U", Click: func(*mygo.MenuItem, *mygo.Window) {
					a.checkUpdate(mainWin)
				}},
				{Label: "统计窗口", Click: func(*mygo.MenuItem, *mygo.Window) {
					openStatsWin(a)
				}},
				mygo.Separator(),
				{Role: mygo.RoleQuit},
			}},
		}))
		mainWin = mygo.NewWindow(mygo.WindowOptions{
			Title:   "Todos",
			Width:   460,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
