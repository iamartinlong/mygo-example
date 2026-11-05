# 组件、主题与自定义组件调研

> 调研日期：2026-11-05，基于 mygo v0.7.4 文档（docs/ui/ 87 篇）。标注「实测」的见 docs/04-verification/。

## 一、组件清单（80+ 控件，官方分 12 类）

| 类别 | 控件 |
|---|---|
| 动作 | Button、Link、Menu button、Context menu、Toggle、Segmented、Toolbar |
| 选择 | Checkbox、Switch、Radio、Select、Slider、Range slider、Stepper、Rating |
| 文本输入 | TextInput/TextArea、Number/Decimal input、Search field、Combobox、Autocomplete、Token field、Editable text、Find bar、Rich text editor |
| 代码 | Code editor（Zed 级）、Diff viewer |
| 文档 | Markdown 渲染 |
| 日期与颜色 | Calendar、Date/Time input、Color picker |
| 集合 | List（虚拟化，实测 10k 行 26 次构建）、Table（排序/拖列）、Tree、Outline、Grid view |
| 容器 | Scroll、Grid（CSS grid 式）、Split、Collapsible、Accordion、Form、Web view |
| 导航 | Tabs、Sidebar、Breadcrumbs、History buttons、Router（带历史） |
| 弹层 | Dialog、Alert、Popover、Tooltip、Toast |
| 状态 | Progress、Spinner、Meter、Badge |
| 图表 | Chart（scales/axes/marks 无外观层，柱/线/面积/堆叠/环图） |
| 图像 | Image（位图+SVG）、Video（GIF/背景播放）、Icon、Avatar |

评价：覆盖桌面应用主流需求无短板；集合类全部虚拟化；输入类组件全部带完整文本编辑（选区/撤销/剪贴板/输入法）。todo 应用已实测其中 7 个（[API 速查](mygo-native-ui-api.md)）。

## 二、主题定制化

**主题是结构体，不是样式表**——`c.Theme()` 拷贝改字段后 `c.SetTheme(&t)` 全窗生效：

- **全局指标**：`Accent`、`Spacing`（4=标准/3=紧凑/5=宽松，全部控件间距是其倍数）、`FontSize`、`Radius`、`ScrollbarWidth`、焦点环三件套（宽/偏移/色）
- **局部主题**：`ui.Themed(c, t)` 让窗口内一块区域用独立主题（明暗预览并排、暗 chrome 内嵌亮面板），主题可嵌套，弹层跟随所属区域
- **跟随系统**：默认随系统深浅色与 accent color 实时切换；`c.Preferences()` 暴露系统偏好（HighContrast、TextScale、ReduceMotion）
- **元素级样式**：Background/Border/Radius/Shadow/InsetShadow/渐变（linear/conic/radial，Oklab 混色）/Stripes/Blur/Mask（渐变遮罩+多层合成）/ClipInset/Cache（位图缓存，类 RepaintBoundary）/Material（玻璃效果插件）
- **颜色系统**：`ui.Oklch/Hex/RGB`，广色域 P3 支持（sRGB 屏自动降采样）；无 CSS 选择器，状态即代码（`if row.Hovered() {...}`）

评价：主题定制能力**超出多数同类框架**——局部嵌套主题、系统偏好联动、Oklab 渐变、多层 mask 是 SwiftUI/Flutter 级别的表达力；「状态即代码」的代价是没有集中样式表，品牌统一靠封装自己的组件层。

## 三、自定义组件可行性：三层路径，从易到难

1. **无外观基座（headless bases）**——推荐路径。20+ 个基座处理指针/键盘/焦点/无障碍，外观完全自己拼：ButtonBase、CheckboxStateBase、SliderBase、TabsBase、SegmentedBase、TreeBase、SelectBase、ComboboxBase、MenuBase、DialogBase、ToastBase、TextInputBase/TextAreaBase……官方定位就是「web 的 headless 组件库」模式。禁用态样式也可接管（`DimDisabled(false)`）
2. **元素组装 + `ui.Local` 状态**——任意组件从元素拼出，`ui.Local` 给元素帧间状态，值实现 `io.Closer` 自动管理后台资源生命周期（组件销毁即停流/停订阅）；`SetChanged` 报告变更、`HandleInput` 接每键、`AttachTo/PressedOutside` 自造浮层
3. **Painter 自绘 + GPU shader**——`Draw` 拿 Painter 画路径/位图/文本整形（`ui.Shape`），SVG path 数据直接进路径；shader（类 WGSL 语言）算像素、做元素特效与 backdrop 效果，成本文档给了量化指引

配套支撑：文本控件自有缓冲区有 `HandleTextInput` + retained text geometry（可做自定义代码编辑器）；无障碍（role/name/states）与键盘导航有完整 API；`ui.Tester` 可无窗口单测组件。

**结论：自定义组件可行性高且路径清晰**——设计系统落地上限是「完全自绘的私有组件库」，下限是「换肤官方 headless 基座」，两种工作量差异约一个数量级。风险点：即时模式 API 在 v0.7.x 仍可能变动，自定义组件层越厚，升级迁移成本越大——自定义层应控制在业务需要的最小集合。

## 四、待实测

- 组件清单与主题能力均来自文档巡礼，未跑 `examples/gallery`（控件巡礼例）逐个过目
- 自定义组件三层各做一个最小例并截图（建议与「下一步」的功能开发合并进行，避免为验证而验证）
