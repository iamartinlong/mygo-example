# Reqbox 迁移 mygo 可行性调研

> 调研日期：2026-11-05。对象：`D:\opencode\Reqbox`（Go + React 单二进制 API 工作台，SQLite 存储）。目标：迁移到 mygo 桌面应用，尽量还原功能与样式。本仓库（mygo-example）为宿主调研仓库。

## 一、Reqbox 现状盘点

### 后端（Go，`internal/`）

| 模块 | 内容 | mygo 复用性 |
|---|---|---|
| `internal/store` | modernc.org/sqlite（**纯 Go 无 CGO**），collections/requests/history/environments/settings 五域，`PRAGMA user_version` 迁移，Windows DACL 加固，`Snap/Restore` 导出导入 | ✅ **零改动直接 import**：无全局状态、无后台 goroutine、无 chi 依赖 |
| `internal/runner` | `Executor.Run`：http.Client 池、`{{var}}` 与内置变量替换、bearer/basic/apikey、httptrace 分阶段计时、2MiB body 上限、`EncodeCurl` | ✅ 零改动直接 import，只依赖 store 类型 |
| `internal/server` | chi 路由：`/api/*` 数据接口（约 30 个）、SPA 静态服务、`/mock/{slug}` 网关、token 鉴权（SHA-256 哈希入库） | 部分复用：API handler 可挂自定义协议；SPA 弃用；mock 需真 HTTP 监听（外部 curl 要能打到） |
| `cmd/reqbox` + `embed.go` | DB 路径约定（exe 旁/用户配置目录）、`//go:embed all:web/dist` | 路径逻辑弃用，改用 mygo PathUserData；embed FS 可直接喂 `mygo.SetFrontend` |

关键事实：**执行/存储/Mock 的全部业务逻辑与 UI 无关，且满足 mygo 的 CGO_ENABLED=0 约束**——迁移成本几乎全在前端。

### 前端（React 19 + Vite + Tailwind v4 + shadcn + zustand）

- 单屏应用无路由：解锁屏（口令解锁，401 驱动）+ Workspace（顶栏 / 左 Sidebar / RequestBar / Request|Response 分栏 / 5 个对话框 / 右下 toast）
- 业务组件仅约 8 个（sidebar 818 行最大），原子组件 15 个（shadcn）；无虚拟化、无拖拽、无可调分栏——**交互密度低，是友好的移植对象**
- 代码编辑用 CodeMirror 6（JSON 高亮、换行）；响应体大时靠 truncated 提示 + 下载规避渲染压力
- 视觉体系固化在 `src/index.css`（309 行，含设计意图注释）：**chromatic 极简**——chrome 全去饱和，唯一彩色是 HTTP 方法色带（get 绿/post 橙/put 蓝/patch 紫/delete 红）+ ok/warn/bad 语义色；oklch tokens、light=blueprint paper 冷调 + `.dark` 变体、Archivo Variable + JetBrains Mono、hairline/eyebrow/tnum 微排版、紧凑密度（12–13px 正文）
- i18n 自写运行时，en/zh 各 280 键 1:1 + parity 测试；快捷键 Ctrl+Enter 发送 / Ctrl+S 保存 / Ctrl+N 新建

## 二、两条迁移路线

### 路线 A：webview 复用（mygo 窗口 + 现有前端）

机制（mygo 文档已确认，**不依赖 Bun/Node 构建**——`devCommand` 只是模板示例）：

```go
//go:embed all:web/dist  ← Reqbox embed.go 已有现成 FS
web, _ := fs.Sub(reqbox.WebAssets, "web/dist")
mygo.SetFrontend(web)
mygo.Protocol.Handle("api", apiMux)   // 现有 /api chi handler 挂自定义协议
mygo.NewWindow(mygo.WindowOptions{URL: "/"})
```

- 前端 `fetch('/api/...')` 相对路径打到 `mygo://localhost/api/...`（Windows 下 `http://mygo.localhost`），由 Protocol.Handle 转给现有 handler——**React 代码零改动**
- 样式还原度 **100%**（就是原来的页面在 WebView2 里跑）
- Mock 网关必须保留真 HTTP 监听（外部工具要能访问）：起仅含 `/mock/{slug}` 的回环 chi server
- 鉴权简化：桌面单用户，可自动建 session 或关掉 token 流（原浏览器登录 URL 流不需要）；数据库 DACL 加固保留
- mygo 侧额外收益：原生窗口/菜单/托盘、安装器、自更新链路（已在本仓库全链验证过）
- mygo.json 的 `updates` 配置从首个发布版本就要带上（本仓库踩坑录的教训）

### 路线 B：原生 UI 重写（ui 包重建界面）

后端同样直接 import store+runner；前端按组件对照表重建：

| Reqbox (React) | mygo (ui 包) | 还原度预期 |
|---|---|---|
| Tabs（Params/Headers/Body/Auth…） | `ui.Tabs` | 高 |
| Select/Dropdown（方法选择、环境） | `ui.Select` / `ui.Combobox` / MenuBase | 高 |
| Dialog ×5 | `DialogBase` headless + 自绘 | 高 |
| Toast（sonner 堆叠动画） | `ToastBase` | 中（动画简化） |
| KVEditor 行编辑 | `ui.Grid`/`Table` 或行拼装 | 高 |
| CodeMirror JSON 编辑 | **ui Code editor（Zed 级）** | 高（能力反而更强） |
| 左右分栏 | `ui.Split` | **超原版**（原版不可拖拽） |
| 列表 | `ui.List`（虚拟化，10k 实测过） | 超原版 |
| oklch tokens + 方法色带 + 明暗 | `c.Theme()` 结构体 + `ui.Oklch` + `ui.Themed` | 高 |
| Archivo/JetBrains Mono 字体 | `ui.RegisterFont` 内嵌（OFL 授权允许） | 高 |
| hairline/eyebrow 微排版 | 元素样式 + 自绘 | 中（需逐个复刻） |
| Radix 焦点陷阱/下拉键盘导航/toast 动画 | headless 基座自带键盘/焦点/无障碍 | 中高 |

工作量估计：8 业务组件 + 15 原子组件 ≈ **10–14 个功能批**（每批实现+真机验证+截图）；样式还原度 ~85–95%，剩下的是 CSS 动画级细节。

## 三、风险与未知

1. **路线 A 的 WebView2 依赖**：Windows 需 WebView2 Runtime（Win10/11 通常自带），首件验证项
2. 路线 A 的 cookie/session 在自定义协议 origin 下的行为未验证——已准备绕法（关 token 流），风险低
3. 路线 B 的即时模式 API 在 v0.7.x 仍可能变动，自定义层越厚升级成本越大（组件调研结论）
4. `web/dist` 已提交但后续前端迭代需要 Node+pnpm 重构建（本机 Node 24 可装 pnpm，无需 Bun）
5. 两路线都需决策：**宿主仓库**——放进本仓库顶层 `reqbox-desktop/` 还是独立仓库（Reqbox 数据库格式向后兼容，迁移后可直读现有 `reqbox.db`）

## 四、结论与建议

**功能可行性：完全可行**（两条路线都成立，后端库零改动是决定性优势）。差异全在样式还原度与工作量：

| | 路线 A（webview 复用） | 路线 B（原生重写） |
|---|---|---|
| 样式还原 | 100% | 85–95% |
| 功能还原 | 100% | 100%（分栏/列表反而增强） |
| 工作量 | 2–3 批 | 10–14 批 |
| 性质 | mygo 壳 + web 界面 | 真原生桌面应用 |
| 长期演进 | 与 web 版同步演进容易 | 独立演进，不受 CSS/CodeMirror 约束 |

**建议分两步走**：先走路线 A 快速得到 100% 还原的桌面版（验证 WebView2、协议挂载、mock 监听、打包更新全链），把 Reqbox 正式跑在 mygo 上；原生化为路线 B 的可选后续，按「哪个界面用得多就原生化哪个」逐面板推进，不做一次性全重写。

若认可，第一步批计划：① `reqbox-desktop/` 最小 mygo 壳（SetFrontend + Protocol.Handle 挂 /api + WebView2 首验 + 截图对比）→ ② 解锁屏绕过 + mock 回环监听 + 数据目录切到 PathUserData（兼容直读现有 db）→ ③ mygo.json 打包 + 菜单/快捷键接入 + 更新链路接入。

## 七、路线 A 执行结果（2026-11-05）

三批全部完成，架构与原计划有一处偏差：Protocol.Handle 换成了 loopback HTTP 服务（mock 网关需要真实 HTTP，且 cookie/web 语义 100% 保留），窗口直接 LoadURL。

| 批 | 内容 | 结果 |
|---|---|---|
| ① mygo 壳 | SetFrontend 换 `//go:embed web/dist` + loopback `http.Server` + WebView2 窗口；store/runner/server 三包零改动 import（仅 import 路径改写） | ✅ UI 100% 还原，截图对比通过 |
| ② 数据 + 桌面化 | 数据目录切 `PathUserData`；mock 端口固定 127.0.0.1:5178（被占回退临时端口并大声告警） | ✅ **双向 db 兼容实测**：桌面版写的库上游 reqbox.exe 可读（token 按库存放，需显式 `-token`），上游库桌面版直开；WAL 注意：拷贝须 db+wal+shm 一起 |
| ③ 打包 + 更新链路 | mygo.json（v0.1.0）+ 图标（按上游 favicon 等比放大）+ CHANGELOG + 专用更新密钥 + `mygo build` 签名构建 + 静默安装 | ✅ exe 17.8MB、安装器 5.8MB、整包 + manifest；安装版实测 `/api/health` 报 `0.1.0`、UI 完整 |

更新链路接入方式：`mygo.Use(updater.Plugin)` 后台检查 + `updates.tagPrefix: "reqbox-v"`——**同仓库托管两个应用的关键**：reqbox-desktop 的发布走 `reqbox-v*` tag，不占 repo 的 latest，与 todo-app 的 `v*` 发布互不干扰（updates.md §41-48）。首个版本无 delta 属正常。

两点如实记录：
- ~~「安装后无更新弹窗」的验证是在 GitHub 匿名 API 限流耗尽时做的~~ **tagPrefix 隔离 E2E 已补验通过**（发布后）：干净限流窗口内启动安装版 0.1.0，`updater.json` 记录了 `lastCheck`（检查真实执行、配额充足），未弹任何更新窗——仓库 latest 是 todo-app 的 v0.3.1，updater 只认 `reqbox-v*`。
- 安装器是 NSIS 系，静默参数 `/S`；Git Bash 会把 `/S` 路径转换成磁盘路径导致静默失败——必须 `MSYS_NO_PATHCONV=1`。

发布记录：release id 434990438，tag `reqbox-v0.1.0`，5 个产物（tar.gz/安装器/install.ps1/SHA256SUMS/manifest，manifest 最后传），`draft=false` + `make_latest=false` 发布，仓库 latest 保持 todo 的 v0.3.1。发布坑：中文 release notes 经 Windows 命令行传 JSON 会被编码损坏（400 Problems parsing JSON）——必须 UTF-8 写文件 `--data-binary @file`。

发布属外部动作，等用户开口再执行（draft release + manifest 最后传的次序照 todo-app 流程）。

## 八、路线 B 批次计划（2026-11-05 用户批准「先 webview，再原生」后开工）

复用批①的 loopback server + store（webview 版的 `reqbox-desktop/` 保留为对照基线；原生版独立目录 `reqbox-native/`，同一 module 下平级包）。顺序按「先骨架后血肉、每批截图对照 web 版」：

| # | 批 | 内容 |
|---|---|---|
| B1 | 主题基座 + 原生壳 | oklch tokens、Archivo/JetBrains Mono 内嵌注册、hairline/eyebrow 工具、左右 `ui.Split` 骨架（空面板）+ 顶栏 |
| B2 | Sidebar | collections/requests `ui.List` + 方法色带 + 新建/重命名/删除 |
| B3 | RequestBar + Tabs | URL/方法 Select、Params/Headers KV 行编辑 |
| B4 | Body/Auth + Code editor | JSON 编辑器（CodeMirror 对位） |
| B5 | 发送 + Response | runner 接入、状态/分阶段计时/body 预览 |
| B6 | History | 运行历史 List |
| B7 | Mocks | mock 发布/复制 URL/停用 |
| B8 | Environments | 环境管理 + `{{var}}` 显示 |
| B9 | 对话框 ×5 | headless DialogBase 重建 |
| B10 | Toast + 快捷键 | Ctrl+Enter/Ctrl+S/Ctrl+N |
| B11 | i18n | en/zh 运行时对齐 web 版 280 键 |
| B12 | 设置/导出导入 | Snap/Restore 接入 |
| B13 | 明暗主题 + 微排版打磨 | `.dark` 变体、tnum、间距校准 |

每批完成 = 编译 + 真机运行 + 截图与 web 版同面板对照 + 提交。
