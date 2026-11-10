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

### `EditableText` 放进 `ui.List` 行内启动即崩

- **现象**：列表行用 `ui.EditableText` 做行内重命名，应用一启动就 panic：`(*listFrame).chosen` nil pointer dereference（list.go:912）
- **原因**：`chosen()` 在 `Selection == nil` 时无条件解引用 `*f.s.Selected`，而 `ui.List` 的 `ListState.Selected` 默认不设（nil）。`EditableText` 每帧都会调 `chosen` 判断"已选中的行才进入编辑"，所以崩在首帧
- **解法**：给 `ListState` 挂一个选中指针并初始化 -1：
  ```go
  a := &appState{selected: -1}
  a.list.Selected = &a.selected
  ```
- **附带认知**：编辑交互是 Finder 式——第一次单击只是选中，**停顿超过 500ms（renameDelay）后再单击**才进入编辑；连续双击永远只是选中

### 行内编辑/点击坐标被 IME 劫持

- **现象**：PowerShell SendKeys 注入 "first todo"，落盘的却是「过河拆桥和秋灰」之类的随机汉字
- **原因**：系统中文输入法把注入的字母当作拼音组词
- **解法**：自动化验证一律走剪贴板：`Set-Clipboard` + `^v`，IME 不参与粘贴链路

### 带 BOM 的 JSON 载入失败

- **现象**：`json.Unmarshal: invalid character '\ufeff'`，应用启动 log.Fatal
- **原因**：Windows 记事本/PowerShell `Set-Content -Encoding UTF8` 写出的文件带 UTF-8 BOM
- **解法**：读盘后先剥 BOM：`data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))`

### 标题渲染两遍

- **现象**：待办行显示 "456 456"
- **原因**：`ui.Checkbox(c, &done, it.Title)` 的 label 已是标题，行内又加了 `ui.Text(it.Title)`
- **解法**：复选框 label 用 `""` + `.Label(it.Title)`（可访问性仍读得到标题），标题只由 `ui.Text` 显示，便于完成态置灰（`TextColor(t.TextMuted)`）

### PowerShell 双引号里的 `$var?` 插值吃掉问号

- **现象**：`"$up?name=$f"` 拼出的 URL 无效（Invalid URI），实际插值结果是 `$up` 后面的 `?` 被并进了变量名
- **原因**：PS 5.1 双引号字符串里 `?` 可以作为变量名字符，`$up?` 被当成一个（不存在的）变量
- **解法**：插值一律用花括号：`"${up}?name=${f}"`；含 `$` 的 PowerShell 命令不要写在 bash 双引号里（会被 bash 先吞一层），存成 .ps1 文件执行

## 发布与更新

### `mygo build` 没有 delta：匿名 GitHub API 配额耗尽

- **现象**：构建成功，日志却是 `No delta updates for windows-amd64: GET https://api.github.com/...: 403 Forbidden`
- **原因**：增量包预检要读 GitHub Releases（匿名 60 请求/小时/IP），当天配额可能已被应用更新检查、发布调用耗光
- **解法**：构建时设 `GH_TOKEN`（或 `GITHUB_TOKEN`）——mygo v0.7.4 会给 api.github.com 请求加 `Authorization: Bearer`，配额升到 5000/h；或等配额重置

### draft release 按 tag 查不到

- **现象**：`GET /repos/{owner}/{repo}/releases/tags/{tag}` 对刚创建的 draft 返回 404
- **原因**：tag 在发布（`draft=false`）时才可见/创建
- **解法**：用创建 release 响应里的 `id` 上传资产并 PATCH 发布；发布后再按 tag 查

### `updater.json` 为 `{}` 不代表没检查

- **现象**：检查失败后状态文件仍是 `{}`，误以为应用没执行检查
- **原因**：更新器只在**成功**检查后写 `lastCheck`；失败时状态文件照存，但 `lastCheck` 不变（失败时间只在内存，最多 1 小时后重试）
- **解法**：看 `lastCheck` 是否更新；结合启动后首个检查固定 10s（`firstCheckWait`）判断时序

### 更新窗口的点击自动化

- **现象**：`updater.Plugin` 的窗口是独立 webview 窗口，主窗口句柄抓不到
- **解法**：按进程枚举顶层窗口（`EnumWindows` + `GetWindowThreadProcessId`）取标题与矩形；被遮挡时用 `PrintWindow`（`PW_RENDERFULLCONTENT`）截图，不必置前；坐标按 DPI 缩放换算，点一次即可（随后窗口缩成进度条视图是正常状态）

## 验证方法

### 截图验证窗口渲染

```sh
./todo-app.exe & sleep 4
powershell -NoProfile -Command "Add-Type -AssemblyName System.Windows.Forms,System.Drawing; ..."  # CopyFromScreen 全屏截图
taskkill //IM todo-app.exe //F
```

注意：全屏截图包含桌面其他内容（浏览器等个人信息），不要提交进仓库。

### Git Bash 的 `/tmp` 与 Node 的 `/tmp` 不是同一目录

- **现象**：node 写 `/tmp/x.json` 后，curl `@/tmp/x.json` 报读不到文件
- **原因**：Git Bash 把 `/tmp` 映射到 `%TEMP%`，而 Windows 版 Node 把 `/tmp` 解释为 `C:\tmp`
- **解法**：两边共用的临时路径用 `cygpath -m /tmp` 取 `C:/...` 形式再传给 node 与 curl；或统一用 `$TEMP` 的 Windows 路径
