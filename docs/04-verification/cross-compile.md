# 交叉编译验收

> 日期：2026-11-05，本机 Windows 10.0.19045，Go 1.26.5（GOTOOLCHAIN=auto 拉 1.27.3），mygo v0.7.4

## 命令与结果

在 `todo-app/` 下，无需任何额外工具链：

```sh
GOOS=linux  GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o build/todo-app-linux-amd64 .
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o build/todo-app-darwin-arm64 .
```

| 目标 | 产物 | 体积 |
|---|---|---|
| linux/amd64 | build/todo-app-linux-amd64 | 9.98 MB |
| darwin/arm64 | build/todo-app-darwin-arm64 | 9.76 MB |
| windows/amd64（本机） | todo-app-release.exe | 9.7 MB |

## 结论

- 「无 cgo、`CGO_ENABLED=0` 任意平台交叉编译」宣称验证成立：Windows 上直接产出 Linux 与 macOS（Apple Silicon）可执行文件，一条命令、零额外依赖
- 三平台体积一致（9.7–10 MB），无平台膨胀
- 注意：这只验证了**编译**。产物未在目标平台运行过，macOS 上还需签名/打包（`mygo build` 在 macOS 上做 bundle + 签名），实机验证列入[生产差距清单](../01-research/mygo-production-gaps.md)
