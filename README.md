# mygo-example

[mygo](https://github.com/egoist/mygo) 桌面框架的调研与示例：纯 Go、无 WebView 的原生 UI 桌面应用（todo 与 Reqbox 迁移），附完整的调研知识库。

## 内容

- [`todo-app/`](todo-app/) — todo 桌面应用（mygo 原生 UI 模式：增删、筛选、持久化、单实例、应用图标、NSIS 安装包）
- [`reqbox-desktop/`](reqbox-desktop/) — Reqbox 桌面版路线 A：webview 复用（Go 后端 + 原 React 前端）
- [`reqbox-native/`](reqbox-native/) — Reqbox 桌面版路线 B：mygo 原生 UI 重写（2026-11-10 迁至独立仓库 [Reqbox-go](https://github.com/iamartinlong/Reqbox-go)，此处仅作对照）
- [`docs/`](docs/README.md) — 知识库（按用途分类）：
  - [01-research](docs/01-research/) — mygo 调研报告、原生 UI API 速查
  - [02-guides](docs/02-guides/) — todo 应用构建/运行/打包指南
  - [03-pitfalls](docs/03-pitfalls/) — 踩坑录（工具链、即时模式 API、并发写盘）
  - [04-verification](docs/04-verification/) — 验收记录（测试、打包、图标、单实例）

## 快速开始

```sh
cd todo-app
go run .        # 开发运行（无 WebView、无前端工具链）
go test ./...   # 存储层测试
```

要求 Go ≥ 1.27.3（低版本 GOTOOLCHAIN=auto 会自动拉取）。打包安装器：

```sh
# 在 mygo 仓库克隆内
go run ./cmd/mygo build ../../todo-app
```

Reqbox 原生版已迁至 [Reqbox-go](https://github.com/iamartinlong/Reqbox-go)（2026-11-10）；本仓库的 `reqbox-native/` 目录仅作历史对照。

## 产物数据（todo-app）

| 构建 | 体积 |
|---|---|
| release exe（`-trimpath -ldflags "-s -w"`） | ~9.7 MB |
| NSIS 安装器 | ~4.5 MB |

应用图标为 AI 生成的占位设计，发布前请替换为自有品牌。
