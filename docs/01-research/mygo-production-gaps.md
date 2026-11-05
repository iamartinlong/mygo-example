# mygo 生产可用差距调研

> 调研日期：2026-11-05，基于 v0.7.4。todo 应用已验证的能力之外，生产还差的调研项，标注状态。

## 已完成（本轮）

### 跨平台编译：宣称成立

`CGO_ENABLED=0` 在 Windows 上一条命令交叉编译成功，见 [验收记录](../04-verification/cross-compile.md)。但只验证了「能编译」，未在目标平台实际运行。

### IME / CJK 支持：渲染已实测通过，组字待人工

- 渲染实测（[验收记录](../04-verification/cjk-input.md)）：中文/日文/Emoji/全角标点/混排全部正确，Emoji 彩色 fallback，输入控件预填中文正常
- 文档层面支持输入法组字：组字期间按键归输入法（Enter 选候选不触发提交），`Composing()` 查询组字状态
- 字体渲染：默认系统界面字体，**CJK 与 emoji 自动 fallback 到系统字体**，RTL 支持；`Font("Inter, Noto Sans JP")` 族列表，`ui.RegisterFont` 内嵌自定义字体
- 待人工：真实输入法组字体验（候选框、下划线、误触）；TextScale 放大后布局

### 性能规模：实测通过

10k 行 `ui.List`：虚拟化实锤（10000 行只构建 26 行）、36 秒仅 30 帧（事件驱动重绘，空闲 CPU 趋零）、内存 ~54 MB 零增长。见[性能验收](../04-verification/perf-list.md)。

### 崩溃行为：关键观察已有，兜底未建

实测 view 函数 panic **直接击穿进程**（无 recover、无对话框）。框架不提供崩溃兜底，生产前需自建：view 函数防御性检查 + 崩溃日志落盘。见[性能验收](../04-verification/perf-list.md)同批记录。

### 宣称功能：本机可测的全部通过

剪贴板、`App.SetMenu`、`NewTray`（[验收 1](../04-verification/native-claims.md)）；`Secrets`、全局快捷键、通知（Toast 视觉确认）、`WatchFiles` 文件监听（[验收 2](../04-verification/native-claims2.md)）。仅剩深链接（需装包后人工验证）与通知点击回调。

### 版本节奏与项目健康度

- v0.7.0 → v0.7.4 仅 5 天（10-27 至 11-01），迭代极快；10+ 个 release
- breaking change 出现在 minor 版本边界（v0.6.0、v0.7.0），patch 无
- 活跃贡献者 30+；作者用 mygo 自建了生产级应用 [mygo-cloudmusic](https://github.com/egoist/mygo-cloudmusic)（336 星，AGPL）——框架有作者自己养的真实用户，是重要正面信号
- 工程纪律严格（AGENTS.md：体积基准测量、禁止墙钟测试、purego 回调约束）

### 锁版本策略

go.mod 可精确 pin 到版本；早期阶段建议 **pin 精确版本 + 升级前读 release notes**，不建议用 `latest`。升级验证跑 `go build && go test` 加 todo 应用截图验收即可。

## 待办（按优先级）

| # | 调研项 | 说明 | 状态 |
|---|---|---|---|
| 1 | 分发信任（签名+更新 E2E） | 方案已明确，见[更新与签名调研](mygo-updates-signing.md)，差实测 | 文档完成 |
| 2 | IME 组字人工实测 | 中文输入真实体验（渲染已机器验证） | 渲染 ✅ / 组字待人工 |
| 3 | 性能规模 | `ui.List` 万条数据的帧率与内存曲线 | ✅ 实测通过 |
| 4 | 崩溃与诊断 | view panic 行为已实测（击穿进程）；崩溃日志落盘待建 | 部分完成 |
| 5 | 数据完整性 | 原子写（temp+rename）、存储格式版本迁移 | 踩坑录已有并发写盘教训 |
| 6 | 宣称功能逐项验证 | 剪贴板/菜单/托盘/Secrets/快捷键/通知/文件监听全 ✅；深链接待装包人工验证 | ✅ 基本完成 |
| 7 | 目标平台实机运行 | 交叉编译产物在 macOS/Linux 真机跑 | 待办 |

## 结论

框架侧的更新/签名接口设计成熟（对标 Sparkle/MacUpdater 的完整方案），主要风险不在功能缺失而在**早期版本稳定性**——缓解手段是锁版本 + 每次升级走完整验收。
