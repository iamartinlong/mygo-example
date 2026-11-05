# 验收记录

日期：2026-11-05 · 环境：Windows 10.0.19045 x64 · Go 1.26.5（GOTOOLCHAIN 自动切 1.27.3）· mygo v0.7.4

## 编译

```sh
cd todo-app && go vet ./...   # 通过
go build .                    # 通过
```

## 测试

```sh
go test ./... -count=5
# ok  todo-app  1.233s   （修复异步写盘顺序 bug 后 5 次全过）
```

覆盖：持久化往返（Add → SetDone → 重开读取）、空标题拒绝、Remove、ClearDone、Stats。

## 运行验证

- 启动后 6 秒进程存活，无 panic（两次独立运行）
- 窗口渲染正确：标题栏、输入框 + Add、All/Active/Done 分段、待办行（复选框 + 标题 + 删除）、页脚统计
- 持久化验证：手动添加的 "456" 待办（已完成）在完全退出后重新启动仍存在
- 截图：`todo-app/screenshot.png`（初版，发现标题重复渲染缺陷）、`todo-app/screenshot2.png`（修复后）
  - 截图为全屏截图、含桌面个人信息，仅本地留存，**不入库**

## 已修复的问题

1. 异步写盘无顺序保证 → `Save()` 改同步（详见 [踩坑录](../03-pitfalls/mygo-pitfalls.md)）
2. 标题在复选框 label 与 Text 中重复渲染 → 复选框空 label + 可访问性 Label
3. 复选框绑定到切片副本内部 → 局部变量绑定 + `SetDone` 显式写回
4. 单数复数："1 items" → "1 item"

## 打包验证

```sh
cd _research/mygo && go run ./cmd/mygo build ../../todo-app
# ✓ Compiled windows/amd64 5.7s；✓ Built Todo Native 0.1.0 in 12.7s（一次通过）
```

- `todo-app/build/windows-amd64/Todo Native.exe`（10,668,544 B）：烟测启动正常，进程存活
- `todo-app/build/windows-amd64/todo-native-setup-0.1.0-amd64.exe`（3,387,511 B，NSIS）：已产出，未实际安装（会改系统状态：%LOCALAPPDATA%、开始菜单、卸载项）

## 图标验证（第二次打包）

- 首次打包未配图标（`mygo.json` 无 `icon`，`resources/icon.png` 不存在），安装后无图标——已补 1024×1024 图标至 `todo-app/resources/icon.png`（ImageGen 生成，去除水印后入库）
- 重打包：安装器 3.2 → 4.4 MB；`[System.Drawing.Icon]::ExtractAssociatedIcon` 从新 exe 提取出的资源图标与设计一致，嵌入确认生效
- 已安装的旧版本用新安装器覆盖安装即更新；用户数据（`%APPDATA%\Todo Native`）不受重装影响

## 体积

- `todo-app.exe`（调试）：14.8 MB
- `todo-app-release.exe`（`-trimpath -ldflags "-s -w"`）：10.3 MB
