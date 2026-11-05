# 宣称功能最小验证（菜单/托盘/剪贴板）

> 日期：2026-11-05，Windows 10.0.19045，mygo v0.7.4。实验应用：`experiments/native-claims/`

## 结果

| 宣称功能 | 验证方式 | 结果 |
|---|---|---|
| 剪贴板 `Clipboard.WriteText/ReadText` | 启动写入固定串 → 外部 `Get-Clipboard` 断言 | ✅ `mygo-claim-test-20261105` 读回一致 |
| 应用菜单 `App.SetMenu` | 窗口截图 | ✅ 窗口菜单栏出现「文件」+「Edit」（自定义中文菜单与 role 菜单并存） |
| 托盘 `NewTray(TrayOptions)` | 返回值 | ✅ 无错误，托盘已创建（图标在任务栏，未视觉确认） |
| PrimaryButton/Button 渲染 | 截图 | ✅ 正常 |

## 待人工项

- 菜单项 `Click` 回调实际触发（含 `CmdOrCtrl+B` 加速键）
- 托盘图标视觉确认与托盘菜单弹出
- 通知（Notification）、深链接、全局快捷键、密钥库、文件监听——本轮未覆盖

## 复现命令

```sh
cd experiments/native-claims
go build -o native-claims.exe .
./native-claims.exe & sleep 5
powershell -Command "Get-Clipboard"   # 断言 mygo-claim-test-20261105
taskkill //IM native-claims.exe //F
```
