package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const itemCount = 10_000

type item struct {
	title string
	done  bool
}

type app struct {
	items   []item
	list    ui.ListState
	mu      sync.Mutex
	visited map[int]bool // 被构建过的行号——虚拟化应远小于 itemCount
	frames  int
	start   time.Time
}

var a = &app{visited: map[int]bool{}, start: time.Now()}

func (a *app) mark(i int) {
	a.mu.Lock()
	a.visited[i] = true
	a.mu.Unlock()
}

func (a *app) visitedCount() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.visited)
}

func view(c *ui.Context) {
	a.frames++
	t := c.Theme()
	ui.Column(c).Fill().Padding(12).Gap(8).Children(func() {
		ui.Textf(c, "10k 列表性能 · 运行 %.1fs · 第 %d 帧 · 历史构建过 %d / %d 行",
			time.Since(a.start).Seconds(), a.frames, a.visitedCount(), itemCount).
			FontSize(13).Bold()
		ui.Textf(c, "完成 %d / %d", countDone(), itemCount).FontSize(12).TextColor(t.TextMuted)
		ui.List(c, &a.list, len(a.items), func(i int) {
			a.mark(i)
			it := a.items[i]
			ui.Row(c).Gap(8).AlignItems(ui.Center).Padding(4, 8).Children(func() {
				done := it.done
				if ui.Checkbox(c.Key(fmt.Sprintf("chk-%d", i)), &done, "").Changed() {
					a.items[i].done = done
				}
				ui.Textf(c, "#%05d %s", i, it.title).FontSize(13)
			})
		}).Grow(1)
	})
}

func countDone() int {
	n := 0
	for i := range a.items {
		if a.items[i].done {
			n++
		}
	}
	return n
}

func main() {
	a.items = make([]item, itemCount)
	for i := range a.items {
		a.items[i] = item{title: fmt.Sprintf("待办事项——第 %d 条，含中文与 mixed text", i+1), done: i%3 == 0}
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Perf List 10k",
			Width:   640,
			Height:  720,
			Content: ui.View(view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
