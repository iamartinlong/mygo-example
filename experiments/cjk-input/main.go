package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type app struct {
	draft     string
	notes     string
	submitted []string
}

var a = &app{
	draft: "预填中文文本 ABC 123",
	notes: "多行文本：\n第二行中文\nThird line mixed 中英混排",
}

func view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Padding(16).Gap(10).Children(func() {
		ui.Text(c, "CJK 输入渲染验证").FontSize(18).Bold()

		ui.Text(c, "1. 中文：你好，待办事项清单").FontSize(14)
		ui.Text(c, "2. 混排：Go 语言 mygo 框架 v0.7.4 原生 UI").FontSize(14)
		ui.Text(c, "3. Emoji：✅ 🚀 ⚠️ 完成任务").FontSize(14)
		ui.Text(c, "4. 全角标点：「引号」、顿号、省略号……！").FontSize(14)
		ui.Text(c, "5. 日文假名：テスト カタカナ ひらがな").FontSize(14)
		ui.Textf(c, "6. 统计：草稿 %d 字符（rune）", len([]rune(a.draft))).FontSize(12).TextColor(t.TextMuted)

		in := ui.TextInput(c, &a.draft).Placeholder("在此输入中文…").AutoFocus()
		if in.Composing() {
			ui.Text(c, "● 输入法组字中").FontSize(12).TextColor(t.Accent)
		}
		if in.Submitted() {
			a.submitted = append(a.submitted, a.draft)
			a.draft = ""
		}

		ui.Text(c, "文本域（TextArea）：").FontSize(12).TextColor(t.TextMuted)
		ui.TextArea(c, &a.notes).Lines(3, 6)

		if len(a.submitted) > 0 {
			ui.Textf(c, "已提交 %d 条：", len(a.submitted)).FontSize(12)
			for _, s := range a.submitted {
				ui.Textf(c, "  · %s", s).FontSize(12)
			}
		}
	})
}

func main() {
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "CJK Input Research",
			Width:   520,
			Height:  560,
			Content: ui.View(view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
