package main

import (
	_ "embed"
	"fmt"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

//go:embed icon.png
var iconPNG []byte

type app struct {
	logs []string
}

var a = &app{}

func (a *app) note(s string) { a.logs = append(a.logs, s) }

func view(c *ui.Context) {
	ui.Column(c).Fill().Padding(14).Gap(8).Children(func() {
		ui.Text(c, "宣称功能验证").FontSize(17).Bold()
		for _, l := range a.logs {
			ui.Text(c, l).FontSize(12)
		}
		ui.Row(c).Gap(8).Children(func() {
			if ui.PrimaryButton(c.Key("clip"), "读回剪贴板").Clicked() {
				a.note(fmt.Sprintf("剪贴板读回: %q", mygo.Clipboard.ReadText()))
			}
			if ui.Button(c.Key("quit"), "退出").Clicked() {
				mygo.App.Quit()
			}
		})
	})
}

func main() {
	mygo.App.WhenReady(func() {
		// 剪贴板：启动写入，外部 PowerShell Get-Clipboard 断言
		mygo.Clipboard.WriteText("mygo-claim-test-20261105")

		// 菜单：窗口菜单栏
		mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
			{Label: "文件", Submenu: []*mygo.MenuItem{
				{Label: "写剪贴板", Accelerator: "CmdOrCtrl+B", Click: func(*mygo.MenuItem, *mygo.Window) {
					mygo.Clipboard.WriteText("from-menu-click")
					a.note("菜单项点击：已写剪贴板")
				}},
				mygo.Separator(),
				{Role: mygo.RoleQuit},
			}},
			{Role: mygo.RoleEditMenu},
		}))

		// 托盘
		tray, err := mygo.NewTray(mygo.TrayOptions{
			ID:   "claims-tray",
			Icon: iconPNG,
			Menu: mygo.NewMenu([]*mygo.MenuItem{
				{Label: "托盘菜单项", Click: func(*mygo.MenuItem, *mygo.Window) { a.note("托盘菜单点击") }},
			}),
		})
		if err != nil {
			a.note(fmt.Sprintf("托盘创建失败: %v", err))
		} else {
			a.note(fmt.Sprintf("托盘创建成功: %+v", tray != nil))
		}

		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Native Claims",
			Width:   480,
			Height:  360,
			Content: ui.View(view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
