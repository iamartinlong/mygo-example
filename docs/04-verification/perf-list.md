# 万条列表性能验收

> 日期：2026-11-05，Windows 10.0.19045，mygo v0.7.4，debug 构建。实验应用：`experiments/perf-list/`

## 实验设计

10,000 条待办（含中文标题），`ui.List` 虚拟化列表 + 每行 Checkbox + 文本。头部实时显示运行时长、累计帧数、**历史构建过的行数**（行构建器把行号记入 map，虚拟化正确时该数应远小于总数）。

## 结果

| 指标 | 实测 | 结论 |
|---|---|---|
| 虚拟化 | 10000 行中历史构建过 **26 行** | ✅ 只构建可见行，10k 数据不增加每帧构建量 |
| 累计帧数 | 约 36 秒共 **30 帧** | ✅ 事件驱动重绘：无交互不重绘，空闲 CPU 趋零 |
| 内存（工作集） | 6s=55.1MB → 36s=53.9MB | ✅ 30 秒空闲零增长，无泄漏信号 |
| 内存（私有提交） | 89.3MB → 89.2MB | 同上 |
| 万行中文渲染 | ✅ 正常，行高一致 | |

debug 构建 14.7 MB，内存与 todo 应用（3 条数据）同量级——数据规模对内存影响可忽略，符合虚拟化设计。

## 同批副产品：view 函数 panic 行为（待办 #4 的关键观察）

实验第一版有个 nil map bug，意外获得崩溃行为第一手数据：**view 函数中的 panic 直接击穿整个进程**——无 recover、无崩溃对话框、无优雅降级，Go panic 栈打印到 stderr 后进程立即退出（Windows 消息循环内 panic 沿 wndProc 冒出）。生产含义：

- 必须在应用层兜底：view 函数保持简单、状态访问加边界检查
- mygo 的 `internal/e2e` 有 GUI 测试设施，但运行时 panic 兜底（recover + 崩溃日志落盘）目前框架没给，需自建

## 复现命令

```sh
cd experiments/perf-list
go build -o perf-list.exe .
./perf-list.exe & sleep 6   # Get-Process perf-list 采样内存，截图后 taskkill
```
