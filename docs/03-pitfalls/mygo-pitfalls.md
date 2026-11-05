# mygo 踩坑录

开发 todo 应用过程中实际踩到的问题，按"现象 → 原因 → 解法"记录。

## 工具链

### Go 版本不够

- **现象**：go.mod 要求 `go 1.27.3`，本机 1.26.5 直接构建报版本不满足
- **解法**：什么都不用装。Go 1.21+ 默认 `GOTOOLCHAIN=auto`，构建时自动从官方下载所需工具链（缓存于 GOPATH，不改系统 PATH）。输出会显示 `switching to go1.27.3`

## package ui API 误用

### `Changed()` 是查询，不是回调

- **现象**：写 `.Changed(func(){...})` 编译不过
- **原因**：即时模式设计——`Changed()` / `Clicked()` / `Submitted()` 都是**每帧调用的 bool 查询**，用 `if` 接住
- **解法**：`if ui.Checkbox(c, &b, "x").Changed() { do() }`

### 没有 `.Align()` 方法

- **现象**：`ui.Row(c).Align(ui.Center)` 编译不过（Element 无此方法）
- **解法**：交叉轴对齐用 `.AlignItems(ui.Center)`；主轴是 `.Justify(...)`

### `c.Key()` 只收一个参数

- **现象**：`c.Key("del", it.ID)` 报 too many arguments
- **解法**：`c.Key(fmt.Sprintf("del-%d", it.ID))`；key 需在同一父容器内唯一

### 复选框绑到切片副本内部

- **现象**：`it := items[i]; ui.Checkbox(c, &it.Done, ...)` 勾选后状态丢失
- **原因**：`Snapshot()` 返回拷贝，`&it.Done` 指向拷贝的临时变量
- **解法**：绑定局部 `done := it.Done`，`Changed()` 时显式写回存储（`SetDone(id, done)`）

## 并发与持久化

### 异步写盘导致旧数据覆盖新数据

- **现象**：测试 `SetDone` 后立即 `ClearDone`，落盘文件偶发仍是"1 条已完成"而不是空列表
- **原因**：`Save()` 里 `go func(){ os.WriteFile(...) }()`——多个写**完成顺序无保证**，晚完成的旧快照覆盖新快照
- **解法**：同步写。小 JSON 文件（几 KB）写盘在主线程可接受；若状态变大，改为单 writer goroutine + channel 序列化，而不是并发 goroutine 直写

## UI 细节

### 标题渲染两遍

- **现象**：待办行显示 "456 456"
- **原因**：`ui.Checkbox(c, &done, it.Title)` 的 label 已是标题，行内又加了 `ui.Text(it.Title)`
- **解法**：复选框 label 用 `""` + `.Label(it.Title)`（可访问性仍读得到标题），标题只由 `ui.Text` 显示，便于完成态置灰（`TextColor(t.TextMuted)`）

## 验证方法

### 截图验证窗口渲染

```sh
./todo-app.exe & sleep 4
powershell -NoProfile -Command "Add-Type -AssemblyName System.Windows.Forms,System.Drawing; ..."  # CopyFromScreen 全屏截图
taskkill //IM todo-app.exe //F
```

注意：全屏截图包含桌面其他内容（浏览器等个人信息），不要提交进仓库。
