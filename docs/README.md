# 知识库索引

围绕 [egoist/mygo](https://github.com/egoist/mygo) 桌面框架的调研与本仓库 todo 应用的沉淀，按用途分类：

## 01-research — 框架调研

- [mygo 调研报告](01-research/mygo-research.md) — 定位、双模式架构、工程化、本机验证结论、评价
- [原生 UI API 速查](01-research/mygo-native-ui-api.md) — todo 应用实际用到的 `package ui` API 与模式
- [分发信任：签名与自动更新](01-research/mygo-updates-signing.md) — 更新链路、updater 插件、Windows/macOS 证书方案与成本
- [生产可用差距调研](01-research/mygo-production-gaps.md) — 交叉编译/IME/版本节奏结论 + 待办清单

## 02-guides — 使用指南

- [todo 应用指南](02-guides/todo-app.md) — 构建/运行/测试/持久化位置，以及如何扩展

## 03-pitfalls — 踩坑记录

- [mygo 踩坑录](03-pitfalls/mygo-pitfalls.md) — 工具链、API 误用、并发写盘等实际踩过的问题

## 04-verification — 验收记录

- [验收记录](04-verification/acceptance.md) — 编译、测试、运行截图的证据与命令
- [交叉编译验收](04-verification/cross-compile.md) — Windows 上一条命令产出 linux/darwin 产物的证据
- [CJK 渲染验收](04-verification/cjk-input.md) — 中文/日文/Emoji/混排渲染与输入控件实测（experiments/cjk-input）

## 本仓库布局

```
todo-app/            todo 桌面应用（mygo 原生 UI 模式）
  main.go            视图（状态 → ui.Context）
  store.go           JSON 持久化存储
  store_test.go      存储层测试
docs/                本知识库
_research/mygo/      mygo 仓库克隆（仅调研用，git 忽略，勿改动）
```
