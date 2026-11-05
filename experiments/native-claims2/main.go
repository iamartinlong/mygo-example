package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type app struct {
	mu   sync.Mutex
	logs []string
}

var a = &app{}

func (a *app) note(s string) {
	a.mu.Lock()
	a.logs = append(a.logs, s)
	a.mu.Unlock()
}

func (a *app) snapshot() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.logs...)
}

func view(c *ui.Context) {
	ui.Column(c).Fill().Padding(14).Gap(6).Children(func() {
		ui.Text(c, "宣称功能验证 2").FontSize(17).Bold()
		for _, l := range a.snapshot() {
			ui.Text(c, l).FontSize(12)
		}
	})
}

func main() {
	mygo.App.WhenReady(func() {
		// 1. Secrets（Windows Credential Manager）
		if err := mygo.Secrets.Set("claims2-test", []byte("s3cret-123")); err != nil {
			a.note(fmt.Sprintf("Secrets.Set 失败: %v", err))
		} else {
			got, err := mygo.Secrets.Get("claims2-test")
			switch {
			case err != nil:
				a.note(fmt.Sprintf("Secrets.Get 失败: %v", err))
			case string(got) == "s3cret-123":
				a.note("Secrets 读写一致: s3cret-123 ✓")
			default:
				a.note(fmt.Sprintf("Secrets 不一致: %q", got))
			}
		}

		// 2. 全局快捷键
		if err := mygo.GlobalShortcut.Register("Ctrl+Alt+J", func() {
			a.note("全局快捷键 Ctrl+Alt+J 触发 ✓")
		}); err != nil {
			a.note(fmt.Sprintf("快捷键注册失败: %v", err))
		} else {
			a.note("快捷键已注册: Ctrl+Alt+J")
		}

		// 3. 通知
		n := mygo.NewNotification(mygo.NotificationOptions{
			Title: "mygo claims2",
			Body:  "通知功能测试 2026-11-05",
		})
		if err := n.Show(); err != nil {
			a.note(fmt.Sprintf("通知失败: %v", err))
		} else {
			a.note("通知已提交系统（Show 无错误）")
		}

		// 4. 文件监听（原生 UI 可用的 mygo.WatchFiles）
		dir := os.TempDir()
		watchDir := filepath.Join(dir, "mygo-watch-test")
		ctx, cancel := context.WithCancel(context.Background())
		_ = cancel // 进程退出即结束监听；不能 defer——WhenReady 闭包返回会立刻取消
		w, err := mygo.WatchFiles(ctx, watchDir, mygo.FileWatchOptions{})
		if err != nil {
			a.note(fmt.Sprintf("WatchFiles 失败: %v", err))
		} else {
			a.note(fmt.Sprintf("监听已启动: %s", watchDir))
			go func() {
				for {
					evs, err := w.Next(ctx)
					if err != nil {
						if !errors.Is(err, context.Canceled) {
							a.note(fmt.Sprintf("监听错误: %v", err))
						}
						return
					}
					for _, e := range evs {
						a.note(fmt.Sprintf("文件事件: %s %s", e.Op, e.Path))
					}
				}
			}()
		}

		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Native Claims 2",
			Width:   520,
			Height:  460,
			Content: ui.View(view),
		})

		go func() {
			time.Sleep(3 * time.Second)
			_ = os.WriteFile(filepath.Join(watchDir, "probe.txt"), []byte("hello"), 0o644)
		}()
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
