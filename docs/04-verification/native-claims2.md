# 宣称功能验收 2：通知/快捷键/密钥库/文件监听

> 日期：2026-11-05，Windows 10.0.19045，mygo v0.7.4。实验应用：`experiments/native-claims2/`

## 结果

| 宣称功能 | 验证方式 | 结果 |
|---|---|---|
| 密钥库 `mygo.Secrets`（Windows Credential Manager） | Set→Get 程序断言 | ✅ 读写一致（`s3cret-123`） |
| 全局快捷键 `mygo.GlobalShortcut.Register` | 注册 Ctrl+Alt+J → PowerShell SendKeys 模拟按下 | ✅ 回调触发（窗口日志「触发 ✓」） |
| 通知 `mygo.NewNotification().Show()` | 全屏截图 | ✅ Toast 在屏幕右下角可见（标题+正文+应用名） |
| 文件监听 `mygo.WatchFiles`（原生 UI 可用，无需 watch 插件） | 外部写文件 → 事件回调 | ✅ `文件事件: write probe.txt`（50ms 防抖） |

## 踩坑

- `WatchFiles` 的 context 不能在 `App.WhenReady` 闭包里 `defer cancel()`：闭包返回即取消，监听立刻以 EOF 结束。context 生命周期应挂到应用整体
- PowerShell SendKeys 需先 `Add-Type -AssemblyName System.Windows.Forms`（每次新进程都要）

## 仍未验证

- 深链接（需安装器注册协议，须装包后人工验证）
- 通知点击回调（`App.OnNotificationClick`）
- macOS 专属能力（SF Symbols、权限流等）
