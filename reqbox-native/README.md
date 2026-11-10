# Reqbox Native（已迁出）

本目录的代码已迁到独立仓库 **[iamartinlong/Reqbox-go](https://github.com/iamartinlong/Reqbox-go)**（2026-11-10，v0.2.0）。

- 新仓库是唯一维护入口：模块路径 `github.com/iamartinlong/Reqbox-go`，`pkg/store`、`pkg/runner` 已内联
- 剩余待办见新仓库 [TODO.md](https://github.com/iamartinlong/Reqbox-go/blob/main/TODO.md)
- 构建、发布与自动更新都在新仓库（`mygo.json` 指向新仓库，`tagPrefix: v`）
- 存量 0.1.2 及更早的安装版 feed 仍指向本仓库，需手动安装一次新仓库版本
- 本目录保留作历史对照，后续改动不要在这里做

迁移记录见 [`docs/01-research/reqbox-migration.md`](../docs/01-research/reqbox-migration.md) 的批34。
