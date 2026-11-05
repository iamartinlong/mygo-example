# mygo 调研报告

> 调研日期：2026-11-05，版本 v0.7.4，commit `ee3d4bd9`（当日仍在活跃提交）

## 一、项目概况

[egoist/mygo](https://github.com/egoist/mygo)（MIT）是 Go 桌面应用框架，对标 Electron/Tauri，主打**纯 Go、低内存、单二进制**。要求 Go ≥ 1.27.3。

## 二、两种窗口模式

| 模式 | 技术 | 特点 |
|---|---|---|
| Web 前端 | 系统 WebView（macOS WKWebView / Linux WebKitGTK / Windows WebView2） | TS 前端 + 从 Go 代码**自动生成类型化 TS 客户端**的 IPC；官方 `examples/todo` 即此模式 |
| **原生 UI**（`package ui`） | 纯 Go 直接 GPU 绘制 | 零 HTML/JS/WebView，`WindowOptions.Content = ui.View(fn)`；自带 80+ 控件（docs/ui/），还支持编译到浏览器（WASM + WebGL2） |

## 三、核心架构特点

- **无 cgo**：purego/syscall 调系统 API，`CGO_ENABLED=0` 任意平台交叉编译
- **类型化 IPC**：`mygo.Bind(service)` + `mygo.NewEvent[T]`，前端客户端代码生成
- **发布链路**：安装包/DMG/Debian 包、签名自动更新、单实例、深链接、托盘/菜单/通知、剪贴板/密钥库/文件监听/全局快捷键等桌面 API（docs/native.md）
- **主题与无障碍**：跟随系统深浅色、accent color、Reduce Motion、TextScale
- **工程化极严**（AGENTS.md）：每次提交测量基准 app 二进制体积；测试禁止依赖墙钟时间；purego 回调须启动时一次性创建

## 四、本机验证结论（Windows 10.0.19045）

- go.mod 要求 Go ≥ 1.27.3，本机 Go 1.26.5 通过 `GOTOOLCHAIN=auto` 自动下载官方工具链解决，**无需手动安装**
- webview 模式的前端工具链绑定 Bun/Vite；无 Bun 时选原生 UI 模式即可完全绕开
- Windows 上 webview 模式依赖系统 WebView2 Runtime；**原生 UI 模式无此依赖**

## 五、体积数据

| 构建 | 体积 |
|---|---|
| todo-app release（`-trimpath -ldflags "-s -w"`） | **10.3 MB** |
| todo-app 调试构建 | 14.8 MB |

官方用 `examples/counter-native` 与 `examples/hello` 在 CI 持续测体积。

## 六、评价

**优点**：无 cgo 交叉编译是真差异化；原生 UI 模式一个二进制搞定一切，无前端工具链；API 为即时模式（状态→视图函数），DX 好；文档极全。

**注意**：项目早期（v0.7.x），API 可能变动；webview 模式绑定 Bun；生态插件（terminal/game/glass/automation 等）多数 Go-only，跨平台支持程度不一。

## 七、官方示例清单（examples/）

todo、counter-native（原生 UI 最小例）、notes（富文本）、code-editor、terminal、markdown-chat、gallery（原生 UI 控件巡礼）、menubar-native、liquid-glass 等 40+。
