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

## 遗留

- 增量包失败回退整包的路径未演练（代码里有，触发条件是本机安装被改动）
- 深链接/通知点击等安装器注册的能力待人工
