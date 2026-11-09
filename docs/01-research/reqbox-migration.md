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

## 九、交接点（2026-11-09，批19–25 完成）

目标：把参数/请求头 KV 编辑器与发布面板按 web 基线重做（批15–18 已把请求框、标签条、Body/Auth 对齐）。**批19 已完成**：`reqbar.go` 的 `kvEditor` 重写为 tF 网格（eyebrow 表头 + border/60 发丝线、24/1fr/1.4fr/32 列 + gap 8、行高 32 radius 4、未启用整行 Opacity(0.45)、14px 复选框、28px mono 14 输入框无框/hover border/60/聚焦 border、hover 显 trash 14 删除钮 24×24、空态虚线 h-16 与非空小按钮 `mt-1.5` 双态、参数页脚 mono 10.5 按计数显隐、请求头页脚恒挂非 mono）；`publish.go` 的 `publishPane` 重写为 sF 堆叠（MaxWidth 512 gap 12：intro 12/1.625、开关卡=整卡 CheckboxBase radius 6 px-3 py-2.5 gap 10、路径名 iF（eyebrow/32px mono 14 输入 + outline sm 自动生成钮，暗色走 input token）、Mock URL 条 h-40 muted/50（globe 14 text-ok + mono 11 + ghost xs 复制钮）、覆盖卡 mt-1 radius 6 border-border/60 bg-muted/30）。明暗 × 参数/请求头/Body=form/发布 八张截图对照通过，截图仅本地 temp 不入库。

### 推送链

- 2026-11-09 交接后已推送（用户「提交推送」）：批19–25 七次（3d244b2…282216a）+ 交接文档更新一次；推前 `git fetch` 比对过，远端无新提交，`git status -sb` 推后与 `origin/main` 齐平。批14–18 与批19–25 的分界：批 14–18 对齐请求框/标签条/Body/Auth（2026-11-07 已推），批 19–25 重做 KV 编辑器/发布/响应面板/历史/四对话框/环境管理器（本次推送）。
- 2026-11-07 交接后已推送（用户「提交推送」）：批14–18 六次 + 交接文档两次 + 图标预备一次。
- 交接默认不推送；远端只有 `iamartinlong/mygo-example`，推之前先 `git fetch` 比对。
- 接手先 `git fetch` 复核，别按记忆里的 hash 行事。

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

批19–25 均已落地（批19：kvEditor/publishPane 重写+对照；批20：tab 面板滚动容器；批21：xl 并排布局；批22：响应面板按 web cF 重建；批23：历史标签页按 web Eh 重建；批24：确认/Prompt/cURL/运行设置四对话框按 shadcn 基线重建；批25：环境管理器按 web _F 重建为 768 双栏）。留给下一批的已知差距：

- ~~保存按钮偶发不响应~~ **批26 已定位并修复**：真实 bug——新环境态 Save 被 `envEditID != ""` 条件块藏掉（web 无条件渲染 Save，仅删除依 env 存在），叠加调试期红底 sed 误删按钮的 primary 背景导致 Save 隐形；两处已修，新建→保存→列表出现→删除→编辑器清空全链真机闭环。
- 响应面板的已知近似：长响应体的编辑器原生固定高（224/192px）内部滚动，web 是编辑器自增长、面板滚动；历史重放的请求头/请求体来自快照（替换前的草稿）而非 web history API 存的实际发送值；等待骨架无 pulse 动画（mygo 无对应动画）。
- 键盘路径：web 删除钮有 `focus-visible:opacity-100` 兜底、输入有 ring，原生未做 focus-visible 显隐与 ring 绘制。
- web 用 `crypto.randomUUID()`，原生用 8 字节随机 hex 作 KV id——格式不同但都是唯一串，库间互换无碍，别当缺陷改。

### 批27 记录（响应面板编辑器自增长 + $P 边框框）

web 的响应编辑器是 `$P`：`overflow-hidden rounded-md border border-border/70 bg-background/40` 的边框框，内容 10px 顶距，`minHeight` 预览 14rem、请求体/原始 12rem，且随内容**自增长、由面板滚动**（不是编辑器内部滚动）。原生此前是固定高 224/192 的裸编辑器内部滚动。

- 新增 `respEditor`：边框框（radius 6、border/70、bg/40、10px 顶距、8px 底）+ 高度 = `行数 × 12.5px × 1.65 + 18`（预览 min 224、其余 192；行数封顶 2000 防超大响应撑爆布局）。
- 三处替换：预览、请求体（替换后）、原始（Response tab）。
- 验证：临时 Go server（127.0.0.1:5199/big 返回 42 行 JSON）→ 原生请求成功（200 · 1.3 KB）→ 预览编辑器自增长、面板滚轮滚到 42/43 行末尾；响应 tab 同框同增长。运行条与状态行位置不变。
### 批26 记录（环境管理器保存钮修复）

批25 遗留的「保存按钮偶发不响应」实为两个真实 bug 叠加（交接误判为注入伪影，已更正）：

1. **新环境态根本没有 Save 按钮**：批25 把删除钮与 Save 一起放进 `if a.envEditID != ""` 条件块——新建（envEditID="")时两者都不渲染。web 的 `_F` 只在删除上判 `y&&`，Save 无条件渲染。修复：Save 移出条件块，删除保持仅限已有环境。
2. **Save 隐形**：调试「偶发不响应」时用红底定位法临时替换 `.Background(p.primary.color())`，事后 sed 反向删除时把整个 Background 调用删掉——按钮有 `primaryFg` 深色字但无 primary 底，暗色主题下几乎不可见。修复：恢复 `.Background(p.primary.color())`。

顺带对齐一处 web 行为：删除环境后编辑器应清空（web `l(null)` + 空名称/空变量），批25 只清了 envEditID 留下名称与变量残留——`deleteEnv` 现同时清空 envEditName/envEditVars。

真机闭环：打开管理器（dev 选中）→ +新建（Save 可见）→ 输入 staging → 保存（toast 环境已创建、列表出现 staging、名称行出现「启用」）→ trash 删除（确认框「删除"staging"?」）→ 删除环境（toast 环境已删除、列表回 dev、编辑器清空）→ 选中 dev 点保存（toast 环境已保存）。全程通过。

**踩坑**：临时「红底定位法」收尾时不能简单反向 sed——`.Background(ui.RGB(255,0,0))` 的删除会连原调用一起铲掉；改色前先备份原行文本，还原时整段替换。
### 批25 记录（环境管理器按 web _F 重建）

web _F：`gap-0 p-0 sm:max-w-3xl`=768 无内距卡片：头部带（环境 14 medium + 副标 11 mutedFg + ml-auto 新建 outline xs）+ `h-[420px] grid-cols-[190px_1fr]` 双栏 + 带底页脚（`env.footer` 10.5 + Done outline sm）。原生 env.go 全量重建：

- **左栏 190px**（bg-muted/30、border-r、p-2）：环境按钮 rounded-md px-2 py-1.5 text-12——选中 paper 底、非选中 mutedFg hover accent/60；active 环境挂 `mono rounded bg-ok/15 px-1 text-[9px] text-ok` 的「当前」badge；空态 `env.noEnvironments`。
- **右编辑器**：名称行（border-b px-3 py-2：名称输入 220×28 12px medium placeholder、非 active 时 `env.useThis` outline xs（激活+toast）、右端 trash ghost icon-xs（确认框走 deleteEnvironment.*，删 active 用 descActive 文案）+ 保存 default xs（save 图标 12 + common.save））；变量滚动区 p-3：行 grid `24/1fr/1.4fr/48`——14px webCheckbox、key/value mono 11.5 28 高 input-token 边框输入（value 按 secret 走密码态）、trash ghost 12px；空态 py-6 居中；`+ 添加变量` ghost xs mt-2。
- **行为对齐**：打开时选择 fallback（上次编辑→active→首个，`openEnvMgr`）；`+新建` 只把编辑器置为未保存态（名称=env.newEnvironment、一行空变量）不落库；保存=Create（新）/Update（旧）且对话框保持开（web 同）；删除走确认框后回到空编辑态。
- **验证**：对话框全结构真机渲染 ✓、选择 fallback（自动选中 dev）✓、`+新建` 进未保存编辑态 ✓、保存 Clicked+toast 出现过 ✓、`CreateEnvironment` API 直调 ✓；保存按钮注入点击偶发不响应见「剩余步骤」首条。

### 批24 记录（对话框按 shadcn 基线重建）

web 对话框内容基线：`sm:max-w-sm`=384（cURL 是 672），`rounded-xl`=12、`bg-popover`、`ring-1 ring-foreground/10`、p-4、部件间 gap-4；标题 text-sm 14 medium（Prompt 保持默认 16）；confirm 描述 11.5 leading-relaxed；页脚两态——confirm/prompt 是透明页脚（pt-1、右对齐、取消 outline sm + 确认 default/destructive sm），运行设置是 muted/50 带底页脚（border-t、rounded-b-xl、Done default sm）。原生新建 dialogs.go：`dialogShell`（含 Clip 让带底页脚贴圆角）、outline/default/destructive sm 按钮三件套、`promptState`/`curlState`。

- **confirmDialog**：384、红 X 16 标题（web confirm 钩子默认 destructive）、11.5 描述、取消 outline sm + **删除请求 destructive**（bad/10 淡底 + bad 字——旧实现是实心红块）。
- **promptDialog（新增功能）**：新建集合不再立即以默认名落库——`askPrompt` 走 web dh（eyebrow 字段 + 32px 输入 AutoFocus + hint 位），确认后才 `CreateCollection(trimmed)`；`prompt.newCollection.*` 五键全部接上。
- **curlDialog（新增功能）**：cURL 行点击从「直接复制」改为开 web vF 的 672px 对话框（终端图标标题 + 11px 副标 + `runner.EncodeCurl` 只读编辑器（10px 顶距的边框框）+ 右对齐复制 outline sm，复制后 toast）；请求栏的复制钮仍直接复制 URL（web bar.copyUrl 语义，顺带修正旧实现复制的是 curl）。
- **runSettingsDialog**：384 shell、标题 14/描述 11、grid gap-3 三行（超时输入 80×28 mono、跟随重定向、校验 TLS 双行标签），复选框换 14px `webCheckbox`，带底页脚 Done。
- **验证**：四对话框真机截图全部按基线渲染；新建集合全流程（打开→输入「支付」→创建→侧栏出现集合+toast）通过；确认框取消路径通过。

### 批24 踩坑

- **ui.List 的可选行会吞掉行内控件的点击**（flagClickable 在行级，hit-test 先于行内元素）——历史行的 trash/重放点击在 List 里全部失效；改行点击=行选择（重放）、删除走右键菜单（web 的点 trash 删除无原生对应），列表套在 `Column(Grow(1))+List(Fill)` 里（批20 嵌套公式）钉住页脚。
- **`ui.Scroll` 溢出内容会盖住同层后绘制的兄弟**——历史页 Scroll 版列表溢出绘制到页脚区（页脚其实每帧都在画，暗色下 mutedFg 文字对比低+采样点选在空白处，连误判三轮）；临时红底定位法（给怀疑元素涂 `ui.RGB(255,0,0)`）一次定位。最终历史页回退 ui.List。
- **`gofmt -w .` 会把 i18n_table.go 的历史 CRLF 全量改写**（交接纪律明令别动）——已 `git checkout` 回滚；今后格式化只点名本批文件。
- ui.Modal 按 Escape 即关——注入粘贴流程里的 ESC 会把刚打开的对话框关掉；`^v` 粘贴不受 IME 影响，对话框流程可省 ESC。
### 批23 记录（历史标签页按 web Eh 重建）

web 历史页是独立片段：**自己的搜索行**（muted/50 底、12px 放大镜、`sidebar.searchHistory` 占位、右侧 trash 清空钮——清空走确认对话框）**替换**请求筛选行；列表行三行布局（状态 mono 10 粗体着色 `2xx ok/0 mutedFg/其余 bad`、方法、`ml-auto` 相对时间 `xh`（zh 双语硬编码：刚刚/N 分钟前/N 小时前/N 天前/日期）、名称 12px fg/85、`{duration} ms · {Sh(url)}`——Sh 是「路径或根主机、空为 no URL」）；删除是 hover 显隐 trash（无右键菜单）；空态是时钟图标（20px mutedFg/50）+ 居中文案。原生重建：`histFilter` 独立过滤词（`ListHistory(a.histFilter)`）、`sidebarPane` 里 history 分支**替换** filterRow（修掉了旧实现两行搜索叠排的 bug）、行 hover accent/50 + trash 显隐（批19 惯用法）、新转写 lucide history/search 图标。实测：13 条数据渲染（状态着色/中文相对时间/时长+路径）、hover 删钮、点击条目重放（选中请求 + 响应回放）、webview 空态基线对照（图标+文案同构）全过；displayURL 语义顺带对齐 Sh（根 URL 显主机、空显 no URL——请求行副标同步受益）。

### 批23 踩坑

- **共用的筛选行会被第二个标签页「替换」而非共存**——旧代码 filterRow 无条件渲染，历史页自己的搜索行叠在其下（截图发现两行搜索）。重建某个 tab 时先看 web 里它是替换还是追加。
- 环境下拉和历史 pill 的 y 都在 66–103 一带，盲点点击容易点到环境下拉——先截图定位再点。

### 批22 记录（响应面板按 web cF 重建）

web 响应区结构：状态行（h-9 border-b）+ 4 个线型标签（请求头/请求体/预览/响应，纯 mono 计数）+ 滚动面板。原生 response.go 全部重建：

- **状态行**：状态 mono 12 粗体按 web vh 着色（**0 mutedFg / 2xx ok / 3xx warn / 其余 bad**——修正了旧实现的背景药丸与 3xx accent、4xx warn 两处错配）；statusText、`·` 分隔、时长 bh（≥1s 显示 `x.xx s`）、大小 yh（**MB 两位小数**，旧实现一位）、环境名段、truncated 徽标（bg-warn/15）；右侧下载（存对话框）+ 复制▾（PopoverBase 单项菜单）。
- **四个标签页**：Headers = 概要折叠区（8 行 fact 两列网格 + 发丝分隔；状态行值着色粗体）+ 响应头/请求头折叠列表（字典序、1fr/1.7fr 网格）+ Cookies + 计时折叠条形图（120px/1fr/56px 三列、track muted/60、fill fg/40、备注行 `· ` 连接）；Request = 替换后请求体编辑器或虚线空注（`类型：无`）；Preview = 错误框/图片（base64 解码 `ui.Image`）/二进制虚线注/高亮编辑器；Response = raw 纯文本编辑器或 binary 提示。全部 `editor.State{ReadOnly: true}`；折叠区默认开（`respClosed` 反集）；tab 切换滚动复位（TrackScroll 惯用法）。
- **验证**：200 路径（状态行/tabs/JSON 预览）、404 路径、错误路径（失败状态行 + 概要失败态 + 预览/响应错误框 + 请求体虚线注）、Headers 折叠区与头列表（排序、着色、粗体）真机全过；成功路径与 webview 同响应并排放大对比一致。

### 批22 踩坑

- mygo v0.7.4 **没有 `MenuBase`**（_research 克隆是更新的源码）——自定义触发器的下拉用 `PopoverBase(anchor, open, fn)`：面板 `Margin(4,0,0,0)` 留顶隙、外部点击/Escape 自动关（`*open` 置回）。
- **webview（reqbox-desktop）侧的 UI 自动化不可靠**：`SendKeys` 的 `^a` 能到、`^v` 剪贴板粘贴不生效；注入点击有时不触发按钮。对照时以 native 侧为主，webview 只取静态态（如空态）或改用其数据文件。
- **截图要复核"截到的是哪个窗口"**：截图脚本按目标窗口 rect 裁剪，但若 `SetForegroundWindow` 失败、目标窗口被同位置的原生窗口盖住，裁剪到的是别人的像素（批22 的 r3 假成功就是这么来的）——用内容特征（标题栏、语言、布局）确认后再采信。
- 两 app 的 `PathUserData` 不同（exe 名不同 → 不同 DB）：webview 的 locale/环境/历史是它自己的，别拿 native 的 DB 状态去解释 webview 的显示。

### 批21 记录（xl 并排布局）

web 右列在 cURL 行之下是 `flex min-h-0 flex-1 flex-col xl:flex-row`：<1280 客户端宽堆叠，≥1280 请求 pane 与响应 pane **并排**（各 flex-1，请求侧 `xl:border-r`），且运行条在请求侧 Tabs 内 `hairline-t mt-auto` 钉底、面板 `xl:max-h-none`（去 46vh 上限、`overflow-y-auto` 仍在）。原生实现（mainPane）：视图内每帧 `a.win.ContentSize()`（`onMain` 主线程内联，安全）≥1280 时走 `Row`：左 `Column(Grow(1) FillHeight, border-r)` 装 tabStrip+panelArea(false)+runBar，右 `Column(Grow(1) FillHeight)` 装 responsePane；否则维持堆叠（panelArea(true)）。panelArea 从 mainPane 抽出，cap/无 cap 两态。实测：默认 1264 堆叠与批20 无差；拉宽到 1584 并排（左右各半、hairline、运行条钉底）、18 行溢出后左栏独立滚动、右栏不动、明暗两态通过。

### 批21 踩坑

- **Row 里子元素宽度是 fit-content**：响应面板外壳 Column 直接放进并排 Row，宽度塌成文字宽（265px）——`FillHeight` 只管高，Row 的 main 是横向。要用一层 `Column(Grow(1) FillHeight)` 包住（Column 的 cross=宽，stretch 铺满）。
- `Window.ContentSize()`/`Size()` 可在视图帧内直接调（`onMain` 检测主线程后内联执行，不入队）；`OnResize` 回调也可用但此场景不需要。
- mygo 内置 automation CLI 仍未接入；resize 注入用 `SetWindowPos`（注意截图脚本先读 rect 后做动作的话，crop 前要重读 rect）。

### 批20 记录（tab 面板滚动容器）

web 面板是 `min-h-0 max-h-[46vh] flex-1 overflow-y-auto px-4 py-3 xl:max-h-none`。原生实现（mainPane）：外包一层自测高的 `Column(MaxHeightPercent(46))`，内放 `Scroll(FillHeight().FillWidth().TrackScroll(&a.panelScroll))`，再内是带 20/20 内距的内容列——内容不超限时面板=内容高，超限封顶滚动。10 行参数实测：封顶 ~354px、滚轮滚动、滚动态下继续加行、切 tab 双向复位全部通过；明暗回归发布/请求头/表单与批19 无差。

- **cap 基准**：mygo 视图拿不到窗口高，`MaxHeightPercent` 按父容器（Split 右 pane）解析——46% × 771 = 354，web 是 46% × 视口 812 = 373，差 ~19px（滚动起点略早）。已知近似，别当缺陷修。
- **tab 复位**：`TrackScroll` 挂 app 自有的 `ui.ScrollState`，`a.panelTab != a.edit.tab` 时换新零值 state——对应 web TabsContent 每次切换重挂、滚动归位的语义。
- mygo 滚动条与 web 的细滚动条（`scrollbar-width:thin`）观感不同（悬停瞬显），未深究。

### 批20 踩坑（复用价值高）

- **`ui.Scroll` 不按内容自测高**：文档明说「Give it a size, or Grow it within its parent」，示例全部 `Grow(1)`/`Fill()`。直接塞进竖排 Column 会塌成几十像素（首版面板只剩一行高）。要「内容高、到顶封顶」必须外包一层能自测高的 `Column(MaxHeightPercent(n))`，Scroll 用 `FillHeight()` 填满它——Column 的 cap 与内容测量都正常（对照实验证实）。
- **滚轮注入**：`mouse_event` 的 wheel `dwData` 是**带符号 DWORD**（P/Invoke 签名必须 `int`，uint 收不了负数）；正 delta = 向上滚。Win32 滚轮发给焦点窗口，注入前 SetForegroundWindow。
- mygo 内置 agent 自动化 CLI（`internal/automationcli`，配 `plugins/automation` 的 socket：snapshot/click/wheel/insertText/screenshot/diff）。本应用未挂该插件，继续用 PowerShell 屏幕注入（shot 脚本在 %TEMP%\reqbox-batch19）；哪天要挂，app `mygo.Use` 该插件 + `$MYGO_AUTOMATION` 指向 socket 即可。

### 批19 实证结论（此前标的未验证假设，全部有了答案）

- **小数 Grow 成立**：`Grow(1.4).Basis(0).MinWidth(0)` 按 `main = base + free*grow/sum`（`layout.go:1146`）精确复现 `minmax(0,1.4fr)`，列宽比与 web 一致。
- **hover 显隐删除钮**：`row.Hovered()` 在子元素 Children 闭包里读父行有效（`input.go:1350` 实时 hover 链）；实现上未 hover 就不画图标（比 `Opacity(0)` 更稳，不可见即不可点）。
- **14px 复选框**：`CheckboxBase` + `Box.Size(14,14).Radius(2)` + `DrawOver(StrokePath)` 自绘；勾路径复刻主题未导出的 `checkPath`（`widgets.go:216`）。选中填充色与 web 逐像素一致（亮 ring = R58 G117 B175）。
- **复选框明暗规则（本批新发现，逐像素实测）**：页面 CSS 没有 `color-scheme` 声明，Chromium 按页面底色自适应控件——未选中框**跟应用主题**而非 OS：亮主题白底 #767676 框，暗主题 #222325 底 #434446 框；选中 = ring 填充 + 对比色勾（亮 ring 白勾 / 暗 ring 深勾）。原生已按此实现（`webBox`）。
- **Mock URL**：web 显示 `window.location.origin`（即 5178），原生显示 `a.mockAddr`（网关实际端口，5178 被占时是随机端口）——有意差异，对照时别当缺陷改。
- ~~web 面板 `overflow-y-auto` 的滚动容器原生侧仍未做~~ **批20 已落地**（见下「批20 记录」）。

### 批19 踩坑（复用价值高的三条）

- mygo `Fill()` = **宽高都 100%**（`element.go:986`）。在竖排 Column 里给卡片用 `Fill()` 会把剩余高度整个吃掉（首版开关卡变 330px 巨框）；横向铺满用 `FillWidth()` 或干脆靠默认 cross-stretch。
- Children 闭包里的 `return` 会**跳过闭包里它之后的所有兄弟节点**——kvEditor 空态分支用 `return` 跳过行循环时把页脚 switch 也吞了。分支要用 if/else，别用 return。
- UI 自动化：`GetWindowRect` 含 DWM 不可见边框，客户区原点在 rect 内缩 **(8,0)**（Win10 隐藏标题栏）——点 14px 小目标必须用 `ClientToScreen` 客户区坐标，rect 坐标系统性偏左 8px。中文 IME 会拦截 `SendKeys` 组合成候选词，注入文本用「ESC 取消候选 + Set-Clipboard + ^v 粘贴」。本环境 python3 不可用（exit 49），文本处理用 sed/heredoc。
