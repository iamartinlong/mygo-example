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

## 九、交接点（2026-11-07，批19 完成）

目标：把参数/请求头 KV 编辑器与发布面板按 web 基线重做（批15–18 已把请求框、标签条、Body/Auth 对齐）。**批19 已完成**：`reqbar.go` 的 `kvEditor` 重写为 tF 网格（eyebrow 表头 + border/60 发丝线、24/1fr/1.4fr/32 列 + gap 8、行高 32 radius 4、未启用整行 Opacity(0.45)、14px 复选框、28px mono 14 输入框无框/hover border/60/聚焦 border、hover 显 trash 14 删除钮 24×24、空态虚线 h-16 与非空小按钮 `mt-1.5` 双态、参数页脚 mono 10.5 按计数显隐、请求头页脚恒挂非 mono）；`publish.go` 的 `publishPane` 重写为 sF 堆叠（MaxWidth 512 gap 12：intro 12/1.625、开关卡=整卡 CheckboxBase radius 6 px-3 py-2.5 gap 10、路径名 iF（eyebrow/32px mono 14 输入 + outline sm 自动生成钮，暗色走 input token）、Mock URL 条 h-40 muted/50（globe 14 text-ok + mono 11 + ghost xs 复制钮）、覆盖卡 mt-1 radius 6 border-border/60 bg-muted/30）。明暗 × 参数/请求头/Body=form/发布 八张截图对照通过，截图仅本地 temp 不入库。

### 推送链

- 2026-11-07 交接后已推送（用户「提交推送」）：批14–18 六次 + 交接文档两次 + 图标预备一次，`git status -sb` 显示与 `origin/main` 齐平、工作区干净；接手先 `git fetch` 复核，别按记忆里的 hash 行事。
- 交接默认不推送；远端只有 `iamartinlong/mygo-example`，推之前先 `git fetch` 比对。

### 已落地的部分（提交 `04a8538`）

- `reqbox-native/icons/trash.svg`、`reqbox-native/icons/globe.svg`（lucide 原样转写，stroke=currentColor，24 viewBox）+ `icons.go` 注册 `iconTrash`/`iconGlobe`。
- 证据：`go vet`、`go build` 全过；启动后进程存活 8 秒无 panic，说明两张 SVG 的 `mustIcon` 解析通过。~~面板接入后仍需补验~~ **批19 已补验**：trash 用在 KV 行 hover 删除钮、globe 用在 Mock URL 条，明暗截图对照通过。
- `gofmt` 只标出 `i18n_table.go` 的历史 CRLF，非本批引入，别顺手改。

### 已核对的 web 基线事实源

源码：`reqbox-desktop/web/dist/assets/index-CZMKJqVm.js` 的 `tF`（KV 编辑器）、`sF`（发布面板）、`Ym`（输入框基类）、按钮 `variant`/`size` 字典；CSS 变量 `--radius:.5rem`、`.eyebrow{font-mono .6875rem/.14em}`、`.rounded-md{calc(var(--radius) - 2px)}`。
实测：`127.0.0.1:5178`（PID 1836）用 browser-use MCP 的 `evaluate_script` 取盒模型（`take_screenshot` 需先在应用内打开浏览器面板，否则报 `NATIVE_BROWSER_VIEWPORT_UNAVAILABLE`）。视口 1045×711，面板内容 x=306。

KV 编辑器：

| 元素 | web 值 |
|---|---|
| 表头行 | `grid-cols-[1.5rem_minmax(0,1fr)_minmax(0,1.4fr)_2rem]` gap 8，`px-1 pb-1.5`，`border-b border-border/60`，高 22.7 |
| 表头文字 | `.eyebrow`：mono 11 / 字距 1.54 / 行高 15.71 / mutedFg |
| 行容器 | `mt-1`（4px） |
| 行 | 高 32，`rounded`(4) `px-1 py-0.5` gap 8；未启用整行 `opacity-45` |
| 复选框 | 14×14，`accent-[var(--ring)]` |
| 键/值输入 | 高 28，radius 8，边框 transparent，`px-1.5 py-1`，mono **14px**（`md:text-sm` 压过 `text-xs`），hover 边框 border/60、聚焦 border；占位符 query=`limit`/`10`，header=`Authorization`/`Bearer …`，form=`field`/`value` |
| 删除钮 | 24×24 radius 6，trash 14，默认 `opacity-0`，行 hover 才显 |
| 列表非空 | 行下方 ghost xs 按钮 `mt-1.5 w-fit`：plus 14 + `common.addRow`，文字 mutedFg |
| 列表为空 | 全宽虚线按钮 `mt-2 h-16`（64px）radius 6 `border-border/70`，plus 14 + `kv.addQueryParam`/`kv.addHeader`；此时没有小按钮 |
| 页脚 | 参数：仅「启用且有键」计数 >0 时 `mono mt-3 text-[10.5px] text-muted-foreground/70` 的 `kv.appendedOnSend`；请求头：恒有同尺寸（非 mono）`kv.headerHint` |

发布面板（`grid max-w-lg gap-3`，宽 512）：

| 元素 | web 值 |
|---|---|
| intro | 12 / 行高 19.5 / mutedFg，512 宽下 3 行 = 39 |
| 开关卡 | h 54.8，radius 6，`px-3 py-2.5`，gap 10，`border-border/70`；复选框 14 带 `mt-0.5`；标题 12.5 medium（行高 17.86），副标 10.5 mutedFg（15） |
| 路径名 | `flex-col gap-1`：eyebrow 15.7 → `flex gap-1.5` 高 32（输入 432.8×32，border-input，`px-2.5`，mono 14；自动生成钮 73.2×28，outline sm：`px-2.5` radius 6 bg-background，文字 12.8）→ 提示 10.5 mutedFg/80 |
| Mock URL 条 | 仅 `exposed && slug` 非空时出现：h 40，radius 6，`bg-muted/50`，`px-2.5 py-2`，gap 8；globe 14 用 `text-ok`，URL mono 11 行高 15.7 截断，复制钮 ghost xs 42×24 |
| 调用方覆盖 | `mt-1` h 77.8，radius 6，`border-border/60` + `bg-muted/30`，`p-2.5`；eyebrow 带 `mb-1.5`，正文 10.5 行高 17.06 |

### 剩余步骤（按序）

批19 三步已全部落地（kvEditor 重写、publishPane 重写、vet/build/真机截图对照+提交）。留给下一批的已知差距：

- web 面板带 `max-h-[46vh] overflow-y-auto`，原生侧无滚动容器，KV 行数多时会撑高面板而非滚动——批20 候选。
- 键盘路径：web 删除钮有 `focus-visible:opacity-100` 兜底、输入有 ring，原生未做 focus-visible 显隐与 ring 绘制。
- web 用 `crypto.randomUUID()`，原生用 8 字节随机 hex 作 KV id——格式不同但都是唯一串，库间互换无碍，别当缺陷改。

### 批19 实证结论（此前标的未验证假设，全部有了答案）

- **小数 Grow 成立**：`Grow(1.4).Basis(0).MinWidth(0)` 按 `main = base + free*grow/sum`（`layout.go:1146`）精确复现 `minmax(0,1.4fr)`，列宽比与 web 一致。
- **hover 显隐删除钮**：`row.Hovered()` 在子元素 Children 闭包里读父行有效（`input.go:1350` 实时 hover 链）；实现上未 hover 就不画图标（比 `Opacity(0)` 更稳，不可见即不可点）。
- **14px 复选框**：`CheckboxBase` + `Box.Size(14,14).Radius(2)` + `DrawOver(StrokePath)` 自绘；勾路径复刻主题未导出的 `checkPath`（`widgets.go:216`）。选中填充色与 web 逐像素一致（亮 ring = R58 G117 B175）。
- **复选框明暗规则（本批新发现，逐像素实测）**：页面 CSS 没有 `color-scheme` 声明，Chromium 按页面底色自适应控件——未选中框**跟应用主题**而非 OS：亮主题白底 #767676 框，暗主题 #222325 底 #434446 框；选中 = ring 填充 + 对比色勾（亮 ring 白勾 / 暗 ring 深勾）。原生已按此实现（`webBox`）。
- **Mock URL**：web 显示 `window.location.origin`（即 5178），原生显示 `a.mockAddr`（网关实际端口，5178 被占时是随机端口）——有意差异，对照时别当缺陷改。
- web 面板 `overflow-y-auto` 的滚动容器原生侧仍未做（见上「剩余步骤」）。

### 批19 踩坑（复用价值高的三条）

- mygo `Fill()` = **宽高都 100%**（`element.go:986`）。在竖排 Column 里给卡片用 `Fill()` 会把剩余高度整个吃掉（首版开关卡变 330px 巨框）；横向铺满用 `FillWidth()` 或干脆靠默认 cross-stretch。
- Children 闭包里的 `return` 会**跳过闭包里它之后的所有兄弟节点**——kvEditor 空态分支用 `return` 跳过行循环时把页脚 switch 也吞了。分支要用 if/else，别用 return。
- UI 自动化：`GetWindowRect` 含 DWM 不可见边框，客户区原点在 rect 内缩 **(8,0)**（Win10 隐藏标题栏）——点 14px 小目标必须用 `ClientToScreen` 客户区坐标，rect 坐标系统性偏左 8px。中文 IME 会拦截 `SendKeys` 组合成候选词，注入文本用「ESC 取消候选 + Set-Clipboard + ^v 粘贴」。本环境 python3 不可用（exit 49），文本处理用 sed/heredoc。
