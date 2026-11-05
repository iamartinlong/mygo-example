# todo 应用指南

mygo 原生 UI 模式的桌面 todo 应用：窗口由 Go 直接 GPU 绘制，无 HTML/JS/WebView。

## 功能

- 输入框添加待办（回车或 Add 按钮）
- 复选框勾选完成，✕ 删除，Clear done 清除已完成
- All / Active / Done 分段筛选；页脚统计（总数 / active / done）
- JSON 持久化，跨运行保留；完成项沉底
- 单实例：再次启动把已运行的窗口还原并聚焦，自身退出（`RequestSingleInstanceLock` + `OnSecondInstance`，锁按应用名 + identifier 识别）

## 构建 / 运行 / 测试

```sh
cd todo-app
go run .                      # 开发运行
go build .                    # 调试构建（~14.8 MB）
go build -trimpath -ldflags "-s -w" -o todo-app-release.exe .   # 发布构建（~10.3 MB）
go test ./...                 # 存储层测试
```

工具链：本机 Go 1.26.5 可直接构建——`GOTOOLCHAIN=auto` 会自动下载 go.mod 要求的 go1.27.3 官方工具链（仅缓存，不改系统）。

## 代码结构

| 文件 | 职责 |
|---|---|
| `main.go` | `appState` 状态 + `view(c *ui.Context)` 视图函数（即时模式，主线程每帧调用） |
| `store.go` | `Todos` 存储：互斥锁保护内存列表，`Save()` 同步写 `%APPDATA%\Todo Native\todos.json` |
| `store_test.go` | 持久化往返、增删清测试（写入是异步时序敏感的，测试等待文件内容匹配） |

## 打包（安装包）

`mygo build`（mygo-cli，在 `_research/mygo` 克隆内运行）产出免安装 exe + NSIS 安装器：

```sh
cd _research/mygo
go run ./cmd/mygo build ../../todo-app
```

- 配置：`todo-app/mygo.json`（name / identifier / version / out）
- 图标：`todo-app/resources/icon.png`（方形 PNG，最好 1024×1024）存在即自动嵌入 exe 资源并用作安装器图标；`icon-dev.png` 可单独给开发版。换图标后重打包即可，无需清缓存
- 产物在 `todo-app/build/windows-amd64/`：`Todo Native.exe`（~9.5 MB，带图标/版本信息/按显示器 DPI manifest，无控制台窗口）+ `todo-native-setup-0.1.0-amd64.exe`（~3.2 MB）
- 安装器装到 `%LOCALAPPDATA%\Programs\Todo Native`（当前用户，免管理员），含开始菜单快捷方式、卸载项、自更新支持；静默安装 `/S`
- NSIS 未安装时，mygo 首次需要会自动下载官方 NSIS 3.13 zip（SHA-256 校验后存于 `%LOCALAPPDATA%\mygo`）
- 跨平台：`-platform darwin/universal,windows/amd64,linux/amd64,...`；macOS/Linux 上打 Windows 安装包需自装 NSIS

## 数据位置

- Windows：`%APPDATA%\Todo Native\todos.json`
- 由 `app.SetName("Todo Native")` + `mygo.App.Path(mygo.PathUserData)` 决定

## 扩展方向

- 编辑已有待办标题：`ui.TextInput` + `OnTextKey`（参考 docs/ui/editable-text.md）
- 截止日期：docs/ui/calendar.md、date-input.md
- 多窗口同步：`mygo.NewEvent[T]` 广播变更（官方 examples/todo 的 store.go 有完整范式）
- 菜单与快捷键：`app.SetMenu(mygo.NewMenu(...))`，见官方 examples/todo/main.go
- 打包：mygo-cli（`go run ./cmd/mygo build`，在 _research/mygo 克隆内可用）
