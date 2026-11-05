// Todo is a desktop todo app in native UI: MyGo draws the window itself
// from Go, with no web page, no HTML and no JavaScript.
//
//	go run .
package main

import (
	"fmt"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// appState is the state the window shows.
type appState struct {
	store  *Todos
	draft  string
	filter int // 0 all, 1 active, 2 done
	list   ui.ListState
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
				if it.Done {
					ui.Text(c, it.Title).Grow(1).TextColor(t.TextMuted)
				} else {
					ui.Text(c, it.Title).Grow(1)
				}
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
			ui.Textf(c, "%d %s", s.Total, total).FontSize(12).TextColor(t.TextMuted).Grow(1)
			if ui.Button(c.Key("clear"), "Clear done").Disabled(s.Done == 0).Clicked() {
				a.store.ClearDone()
			}
		})
	})
}

func (a *appState) add() {
	if _, err := a.store.Add(a.draft); err == nil {
		a.draft = ""
	}
}

func main() {
	store, err := OpenStore()
	if err != nil {
		log.Fatal(err)
	}
	a := &appState{store: store}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
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
