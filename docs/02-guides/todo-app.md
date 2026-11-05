# todo 应用指南

mygo 原生 UI 模式的桌面 todo 应用：窗口由 Go 直接 GPU 绘制，无 HTML/JS/WebView。

## 功能

- 输入框添加待办（回车或 Add 按钮）
- 复选框勾选完成，✕ 删除，Clear done 清除已完成
- All / Active / Done 分段筛选；页脚统计（总数 / active / done）
- JSON 持久化，跨运行保留；完成项沉底

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

## 数据位置

- Windows：`%APPDATA%\Todo Native\todos.json`
- 由 `app.SetName("Todo Native")` + `mygo.App.Path(mygo.PathUserData)` 决定

## 扩展方向

- 编辑已有待办标题：`ui.TextInput` + `OnTextKey`（参考 docs/ui/editable-text.md）
- 截止日期：docs/ui/calendar.md、date-input.md
- 多窗口同步：`mygo.NewEvent[T]` 广播变更（官方 examples/todo 的 store.go 有完整范式）
- 菜单与快捷键：`app.SetMenu(mygo.NewMenu(...))`，见官方 examples/todo/main.go
- 打包：mygo-cli（`go run ./cmd/mygo build`，在 _research/mygo 克隆内可用）
