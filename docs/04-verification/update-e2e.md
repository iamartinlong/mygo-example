# 更新链路 E2E 验收

> 日期：2026-11-05，Windows 10.0.19045，mygo v0.7.4。仓库 github.com/iamartinlong/mygo-example

## 链路

`mygo keygen`（密钥在 %APPDATA%\mygo\update-keys，不入库）→ mygo.json 配公钥+GitHub feed → 构建带 `MYGO_UPDATER_PRIVATE_KEY` → 发布产物到 GitHub Releases（gh CLI 缺失，用 REST API 手动上传）→ 已装应用 Ctrl+U 触发 Check/Install/Relaunch。

## 结果

| 环节 | 结果 |
|---|---|
| v0.2.0 构建+签名 | ✅ 安装器 5.1MB、整包 tar.gz 6.0MB、manifest、SHA256SUMS |
| GitHub release 发布（API 手动） | ✅ draft→published→latest 指向正确 |
| 静默安装 `/S` | ✅ 安装器遵循「留在原目录」：在 d:\soft\Todo Native（v0.1.1 旧位置）原地升级到 0.2.0 |
| 无新版时检查 | ✅ 页脚「已是最新版本 v0.2.0」 |
| **v0.2.0→v0.2.1 应用内升级** | ✅ **增量包仅 13 KB**（整包 6MB），下载→验签→自替换→重启，页脚 v0.2.1，磁盘 exe VersionInfo=0.2.1 |
| 增量包生成 | ✅ 构建时自动下载上一版已发布包、验签、bsdiff 产差量 |

## 关键事实

- **旧版不可升级**：不带 `updates` 配置构建的版本（v0.1.1）`Updater.Enabled()` 为 false，永远收不到更新——更新能力必须从首个发布版本就带上
- mygo 安装器是 per-user（%LOCALAPPDATA%\Programs 或用户自选目录），自更新写权限天然满足
- `-upload` 需要 gh CLI；本机无 gh，REST API 手动上传等价可行（manifest 最后传）
- 应用侧 UI 自己实现（菜单加速键 Ctrl+U + `mygo.Updater.Check/Install` + `App.Relaunch`），没用 updater 插件的现成窗口——插件窗口同样可用，为可自动化选了自绘
- 运行时 `App.Version()` 由 `mygo build` 注入；`go build` dev 运行为空，需兜底显示

## 第二轮：v0.3.0（2026-11-05）

三批功能（行内重命名/到期日/多窗口同步）后的例行发版，验证链路可重复。

| 环节 | 结果 |
|---|---|
| 版本号 0.2.1→0.3.0 提交推送 | ✅ `500257c` |
| `mygo build` 签名构建 | ✅ exe 11.9MB、安装器 5.2MB；**自动生成两条 delta**（0.2.1→0.3.0 与 0.2.0→0.3.0，各 675KB） |
| 烟测 | ✅ dev 运行页脚 v0.3.0，exe VersionInfo=0.3.0 |
| GitHub release v0.3.0（REST API） | ✅ id 434833143，上传 tar.gz/安装器/双 delta/manifest，published + latest |
| **v0.2.1→v0.3.0 应用内升级** | ✅ 675KB 增量包，页脚 v0.3.0，磁盘 exe VersionInfo=0.3.0 |

第二轮自动化新坑（记录在 pitfalls 的验证方法之外，属升级自动化）：

- **Ctrl+U 会被 Alt 态菜单栏吞掉**：用 keybd_event 的 Alt 置前台技巧后，菜单栏停留在「Alt 按下」状态，紧随的 `^u` 变成菜单导航。菜单项点击（文件→检查更新…）更可靠
- **窗口 rect 可能错位**：多窗口/重启后 `Get-Process` 的 `MainWindowHandle` 不保证指向预期窗口，截到的是叠在上面的其他窗口。先 `SetForegroundWindow` 并核对 `GetForegroundWindow() -eq $handle` 再截
- **无 BOM UTF-8 ps1 按 ANSI 解析**：脚本里写中文字符串会坏，中文正文用 base64 注入或存盘带 BOM

## 第三轮：v0.3.1——基座漂移回退演练（2026-11-05）

验证「增量包失败回退整包」：向已安装的 v0.3.0 exe 末尾附加 25 字节（PE overlay 区，应用仍正常运行），随后发 v0.3.1 并应用内升级。

| 环节 | 结果 |
|---|---|
| 基座漂移（附加 25 字节） | ✅ 哈希改变，应用照常启动（页脚 v0.3.0） |
| `mygo build` v0.3.1 签名构建 | ✅ 自动生成 3 条 delta（0.3.0→478KB / 0.2.1、0.2.0→777KB） |
| GitHub release v0.3.1（REST API） | ✅ id 434864573，6 资产，published + latest |
| **漂移基座应用内升级** | ✅ 下载→安装→重启，页脚 v0.3.1 |
| 漂移是否残留 | ✅ 无：安装后 exe 与本地官方 0.3.1 构建 **SHA-256 逐字节一致** |

**结论**：更新器没有把 delta 盲打在被改动的基座上（否则 bsdiff 输出哈希必不匹配 manifest），整包回退/基座校验路径真实生效。中间过程（先试 delta 再回退，还是先校验基座直接走整包）从外部不可观测，以终态为准。

## 遗留

- 增量包失败回退整包的路径未演练（代码里有，触发条件是本机安装被改动）
- 深链接/通知点击等安装器注册的能力待人工
