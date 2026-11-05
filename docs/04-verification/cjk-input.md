# CJK 渲染与输入验收

> 日期：2026-11-05，Windows 10.0.19045，mygo v0.7.4。实验应用：`experiments/cjk-input/`

## 实验设计

最小应用窗口内覆盖六类渲染样张 + 两种输入控件：

1. 中文纯文本　2. 中英混排　3. Emoji　4. 全角标点（「」、顿号、省略号）　5. 日文假名　6. rune 统计
   控件：`TextInput`（中文 placeholder + 预填中文 + AutoFocus + `Composing()` 组字指示）+ `TextArea`（多行中文）

## 结果（截图验证，截图仅本地留存）

| 项 | 结果 |
|---|---|
| 中文/日文渲染 | ✅ 正确，无豆腐块 |
| Emoji（✅🚀⚠️） | ✅ 彩色渲染（系统字体 fallback） |
| 全角标点 | ✅ 正确 |
| 中英混排 | ✅ 正确 |
| rune 统计 | ✅ 「预填中文文本 ABC 123」计 14，准确 |
| TextInput 预填+聚焦 | ✅ 预填中文可见，蓝色聚焦框 |
| TextArea 多行 | ✅ 换行保留，行高正常 |

## 边界与待人工项

- **IME 组字过程未验证**：候选框位置、组字下划线、Enter 选候选不误触 `Submitted()`——需要人工开中文输入法在实验应用里打字确认（`Composing()` 指示与 `text-input.md` 宣称的行为已就位，但宣称≠实测）
- 未测 TextScale 放大后的 CJK 布局

## 复现命令

```sh
cd experiments/cjk-input
go build -o cjk-input.exe .
./cjk-input.exe & sleep 5   # 截图后 taskkill //IM cjk-input.exe //F
```
