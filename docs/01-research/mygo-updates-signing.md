# mygo 分发信任：签名与自动更新

> 调研日期：2026-11-05，基于 mygo v0.7.4 文档（docs/updates.md、docs/distribution.md、docs/plugins/updater.md）。实测结果见文末「三、实测结果」（todo 多轮 + reqbox-native 0.1.1→0.1.2）。

## 一、自动更新链路

mygo 内置更新器（对标 Sparkle），链路为「密钥签名 → 发布清单 → 应用内校验替换」：

1. **密钥对**：`mygo keygen` 生成 `mygo-update.key`（私钥）+ `mygo-update.pub`（公钥）。私钥丢失 = 已装用户永远收不到更新，泄露 = 任何人可向用户推代码——按 CI secret 管理。
2. **配置**：`updates.publicKey` + `updates.github`（GitHub Releases）或 `updates.url`（任意 HTTPS/S3/R2）。
3. **构建**：`MYGO_UPDATER_PRIVATE_KEY` 传私钥，`mygo build` 产出各平台 tar.gz + **bsdiff 增量包**（对比最近 3 个已发布版本）+ `update-<platform>.json` 清单。增量包预检要读 GitHub Releases（匿名 60 请求/小时/IP），配额耗尽时构建照常成功但不带 delta；设 `GH_TOKEN`/`GITHUB_TOKEN` 会让 mygo 对 api.github.com 请求带 Bearer（5000/h）。
4. **应用侧**：`mygo.Updater.Check/Install` 下载后先验签名（公钥内嵌于应用）再替换应用，失败自动回退整包下载。

发布渠道：GitHub Releases（`-upload` 上传草稿，发布后生效）、S3/R2（`-upload` 直传，清单最后传）、自建服务器（注意清单最后上传，避免清单指向缺失文件）。预发布（beta）走 `SetPrereleases(true)`，语义化版本比较。还可用 `SetSource` 自定义更新服务（如需鉴权的下载源）。

### updater 插件（现成的更新窗口 UI）

```go
mygo.Use(updater.Plugin)   // 原生 UI 应用用 updater 插件的 native.Plugin
```

启动 10 秒后台检查、之后每天一次；弹更新窗口（安装/跳过/稍后提醒），偏好存 `PathUserData/updater.json`；`updater.CheckForUpdates()` 供菜单项调用。开发构建与包管理器安装的构建自动禁用（`Enabled()` 为 false）。

### 关键约束：更新要求应用可写自身安装位置

- Windows：必须装在 `%LOCALAPPDATA%\Programs`（mygo 安装器的 per-user 装法），Program Files 装法无法自更新
- macOS：/Applications（管理员）或 ~/Applications
- Linux：用户目录（install.sh 的装法）；Debian 包等包管理器装法由包管理器更新

## 二、代码签名

### Windows

mygo 支持两种接入（mygo.config.ts 的 `windows` 字段）：

| 方式 | 配置 | 工具 |
|---|---|---|
| PFX 证书文件 | `windows: { certificate: "certs/code-signing.pfx" }` | Windows 上用 signtool（Windows SDK），其他平台 osslsigncode |
| 外部签名命令 | `windows: { signCommand: "signtool sign /fd sha256 /tr … /a %1" }` | 适配硬件令牌 / 云签名服务（如 Azure Trusted Signing），`%1` 为待签文件 |

`mygo build` 会把资源里所有未签名的 exe/DLL 连同安装器、卸载器一起签；时间戳服务器默认 DigiCert，可用 `timestampUrl` 换。

**证书现实**（2026-11-05 结论）：

- 正式分发必须有证书，无捷径；自签名对分发无用
- OV 证书约 $100–400/年，SmartScreen 信誉从零积累；EV 更贵需硬件令牌
- **最便宜正规途径：Azure Trusted Signing（约 $10/月）**，走 `signCommand` 接入，需微软个人/企业身份审核（个人相对容易）
- 未签名的后果：SmartScreen「未知发布者」警告，用户需「更多信息 → 仍要运行」；该提示无签名永远消不掉

### macOS

正式分发必须 Apple Developer Program（$99/年）+ Developer ID 签名 + 公证（notarization）。mygo 默认 ad hoc 签名（Apple Silicon 本机可跑，不可分发）。`mygo build` 在 macOS 上自动完成 bundle 内从里到外的签名与公证，`-skip-notarize` 可跳过。

## 三、实测结果（原待实测项）

2026-11-05 起在 todo 与 reqbox-native 上多轮验证，详见[更新链路 E2E 验收](../04-verification/update-e2e.md)：

1. **完整 E2E 升级**：✅ todo v0.2.0→0.2.1→0.3.0→0.3.1（含基座漂移回退演练）；reqbox-native v0.1.1→0.1.2 走 `updater.Plugin` 窗口（2026-11-10）。用户数据保留、失败回退整包均通过
2. **增量包真实节省比例**：✅ todo 13KB/6MB、675KB/6MB；reqbox-native 22KB/9MB。收益取决于版本差异，`updates.deltas: 0` 可关
3. **Azure Trusted Signing**：⏳ 仍未实测（证书方案见第二节）
4. **Windows per-user 下 `Updater.Enabled()`**：✅ `%LOCALAPPDATA%\Programs` 安装版正常检查并完成升级
