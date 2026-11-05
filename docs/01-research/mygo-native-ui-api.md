# mygo 原生 UI API 速查

todo 应用实际用到的 API，均已在本机编译运行验证（v0.7.4）。完整文档见克隆仓库 `docs/ui/`。

## 窗口与视图

```go
mygo.App.WhenReady(func() {
    mygo.NewWindow(mygo.WindowOptions{
        Title:   "Todos",
        Width:   460, Height: 640,
        Content: ui.View(viewFn), // viewFn(c *ui.Context)，主线程每帧调用
    })
})
if err := mygo.App.Run(); err != nil { log.Fatal(err) }
```

视图函数是**即时模式**：从状态（struct 字段）直接构建 UI，状态变了下一帧自动重绘。

## 布局（ui.Element 链式方法）

```go
ui.Column(c).Fill().Children(func() { ... })          // 纵向
ui.Row(c).Gap(8).AlignItems(ui.Center).Children(...)  // 横向，交叉轴居中
// 注意：没有 .Align()，交叉轴对齐用 .AlignItems(ui.Center)
```

常用链式：`.Fill() .Grow(1) .Padding(...) .Gap(n) .Width(n) .Disabled(bool)`

## 文本

```go
ui.Text(c, "hi").FontSize(12).TextColor(t.TextMuted).Bold().Grow(1)
ui.Textf(c, "%d active", n).FontSize(12)
t := c.Theme()   // t.TextMuted 等；ui 默认跟随系统深浅色与 accent
```

## 输入框

```go
if ui.TextInput(c, &a.draft).Placeholder("...").AutoFocus().Submitted() {
    a.add()   // Submitted()：回车提交，bool 查询
}
```

## 按钮

```go
if ui.PrimaryButton(c.Key("add"), "Add").Clicked() { ... }
if ui.Button(c.Key("del-7"), "✕").Label("Delete").Clicked() { ... }
// Key(any) 只收一个参数；多字段用 fmt.Sprintf 拼进 key
```

## 复选框

```go
done := it.Done  // 绑定到局部副本，避免指向切片拷贝内部
if ui.Checkbox(c, &done, "").Label(it.Title).Changed() { // Changed() 是 bool 查询，不收回调
    a.store.SetDone(it.ID, done)
}
```

## 分段控件

```go
ui.Segmented(c, &a.filter, "All", "Active", "Done") // *int 为选中下标
```

## 虚拟化列表

```go
ui.List(c, &a.list, len(items), func(i int) {
    // 行内容；只构建可见行
}).Grow(1).Children(func() {
    // 列表为空时的占位内容
})
// &a.list 是 ui.ListState，保存在状态里以记住滚动位置
```

## 持久化路径

```go
dir, _ := mygo.App.Path(mygo.PathUserData) // Windows: %APPDATA%\<AppName>
app.SetName("Todo Native")                  // 决定该目录名
```
