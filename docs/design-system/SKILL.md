---
name: icloud-privacy-mail-design-system
description: iCloud Privacy Mail 运维面板设计系统。构建自托管工具类后台、数据密集表格界面、邮箱/凭据管理台或任何需要长时间注视的运维界面时使用。提供 Light/Dark 双主题、舒展/紧凑双密度的纯 CSS 令牌层与已应用组件层。
user-invocable: true
---

# iCloud Privacy Mail 设计系统

## What is inside

包内文件与各自职责（tokens / components / preview / assets / ui kit 五层）：

| 路径 | 内容 |
| --- | --- |
| `DESIGN.md` | 权威规范，17 节。产品上下文、色彩、排版、间距、组件、动效、语气、反模式 |
| `README.md` | 面向人的包说明：产品概述、审阅顺序、preview 清单 |
| `SKILL.md` | 本文件。面向代理的硬性规则与交付前检查 |
| `colors_and_type.css` | colors + typography 语义令牌层，Light / Dark 双主题，oklch 定义 + sRGB 回退 |
| `tokens.css` | spacing / radius / shadows / motion / density 令牌 |
| `ui_kits/app/` | 已应用界面套件：入口页、六个模块化 components、三个可交互屏幕 |
| `preview/` | 7 张聚焦预览卡，覆盖颜色、排版、间距、圆角阴影、组件、品牌资产、已应用界面 |
| `assets/icons/` | 从源实现提取的 6 个 24×24 描边 SVG 图标 |
| `examples/templates/` | 三个源模板逐字保留（159KB / 47KB / 12KB），未做删减 |
| `context/` | 交接说明、领域语言、证据出处（含缺失证据说明） |

无 `build/`：源仓库不含运行时图标资源。无 `fonts/`：字体栈全部依赖系统字体，源仓库无字体文件。两处缺失在 `context/provenance.md` 中有说明。

## Source Context

本系统不是凭空设计的，全部结论基于真实 source evidence：源项目 `iCloud-Privacy-Mail` 是 Go 后端 + 服务端渲染 HTML 模板的自托管隐私邮箱运维面板，本地 repository 由代理直接读取。三个模板逐字保留在 `examples/templates/`，令牌、类名、状态语义、界面文案都可追溯到那里。每条结论的出处、以及缺失的证据（无位图资产、无字体文件、无设计源文件、无既有设计系统），都记在 `context/provenance.md`。

领域术语来自源仓库的统一业务语言，逐字保留在 `context/source-domain-language.md`，是硬约束而非风格建议。

## When to use this skill

适用：自托管运维面板、数据密集的表格界面、凭据与账号管理台、内网工具、任何用户会长时间盯着列表等状态变化的界面。

不适用：营销页、品牌官网、演示文稿、面向消费者的产品页。这套系统刻意压制了表现力，用在需要品牌张力的场合会显得寡淡。

## Design System Highlights

一屏看懂这套系统的取向（详规见 `DESIGN.md`）：

| 维度 | 取向 |
| --- | --- |
| colors | 界面主体是中性灰阶；唯一强调色 `--accent`，实心填充一屏 ≤2 次；绿 / 琥珀 / 红只表达邮箱与登录态的真实状态。Light / Dark 两套主题（源实现有 10 套） |
| typography | 三套系统字体栈、11 级 `font:` 简写阶梯、字重只用 400 / 500 / 600；全部数字 `tabular-nums`；邮箱 / Token / 验证码用等宽 |
| spacing | 4px 栅格 + 双档 density（comfortable / compact），密度只改 padding 与 gap |
| radius | 取值集合 ⊆ {4, 6, 8, 999}px，面板与卡片上限 8px |
| shadows | 只有 dropdown / modal / sticky 三种，只服务浮层；页面内卡片无阴影；全站无渐变 |
| layout | 240px 侧栏 + 1440px 内容上限；≤1120px 侧栏转顶部导航；≤820px 表格转卡片 |
| interaction | 140ms 只过渡颜色类属性；无 transform 悬浮、无入场关键帧；每视图一个主操作；不可逆操作两步确认 |
| icons | 24×24 viewBox、`stroke-width: 1.8`、`fill: none`、`stroke: currentColor`，见 `assets/icons/` |

## 开始前必读

1. `DESIGN.md` — 权威规范。§0 产品上下文、§13 动效、§14 语气、§15 反模式是最常被忽略但最容易出错的四节。
2. `context/source-domain-language.md` — 领域术语硬约束。写任何界面文案前必须读。
3. `colors_and_type.css` + `tokens.css` — 令牌。不要重新定义，不要引入这两个文件之外的色值。

## How to use

按顺序引入三个 CSS，组件层依赖前两个：

```html
<link rel="stylesheet" href="colors_and_type.css">
<link rel="stylesheet" href="tokens.css">
<link rel="stylesheet" href="ui_kits/app/components.css">
```

目标环境无构建链时（源项目就是这种情况），按同样顺序把三个文件内容粘进第一个 `<style>`。

根元素挂主题与密度：

```html
<html lang="zh-CN" data-theme="light" data-density="comfortable">
```

`data-theme` 取 `light | dark`；`data-density` 取 `comfortable | compact`。两档密度只改 padding 与 gap，不改字号、圆角、边框宽度。

## 复用哪些文件

需要一个完整屏幕时，从 `ui_kits/app/` 复制最接近的那个，替换内容，不要从零写 CSS：

| 需求 | 起点 |
| --- | --- |
| 带侧栏的多视图工作台 | `ui_kits/app/workbench.html` |
| 管理后台 + 弹窗 + 危险操作 | `ui_kits/app/account-manage.html` |
| 登录 / 认证页 | `ui_kits/app/login.html` |
| 多屏索引页 | `ui_kits/app/index.html` |

组件类名全部定义在 `ui_kits/app/components.css`，直接用：
`.app-shell` `.sidebar` `.brand` `.side-nav` `.nav-item` `.count-dot` `.content-head` `.content-main`
`.btn`（`.primary` `.secondary` `.ghost` `.danger` `.danger-solid` × `.sm` `.lg`）
`.card` `.metric-grid` `.metric-card` `.panel` `.panel-head` `.panel-body`
`.table-scroll` `.cell-email` `.cell-time` `.num` `.pill`（`.status-available` `.status-used` `.status-failed` `.status-disabled` `.status-unknown`）`.empty-state` `.skeleton-bar` `.pager` `.row-actions`
`.field` `.field-label` `.required` `.input` `.select` `.textarea` `.field-error` `.field-hint` `.field-grid` `.form-actions` `.input-num` `.switch` `.alert` `.tabs` `.tab`
`.danger-zone` `.dropdown` `.dropdown-item` `.modal` `.log`
`.mono` `.code-value` `.text-display` `.text-metric` `.text-body-strong` `.text-label` `.text-caption`

图标从 `assets/icons/` 取，或按同样规格新画：24×24 viewBox、`stroke-width: 1.8`、圆头圆角连接、`fill: none`、`stroke: currentColor`。不要用 emoji 或几何字符当图标。

## 硬性规则

违反其中任何一条即为交付失败。

**颜色**
- 只用 `colors_and_type.css` 里的变量，不写任何字面色值（十六进制、rgb、命名色）。
- 一屏之内 `--accent` 实心填充最多 2 次。
- 白底小号文字用 `--accent-text`，不用 `--accent`。
- 表格行悬浮用 `--surface-hover`（中性），选中态才用 `--accent-subtle`。
- 状态胶囊必须带文字，不能只靠颜色表意。

**排版**
- 字重只用 400 / 500 / 600。
- 不对中文用 `text-transform: uppercase` 或 `letter-spacing`。
- 全部数字容器加 `font-variant-numeric: tabular-nums`。
- 邮箱 / Token / 验证码 / 时间戳 / 日志用 `--font-mono`。
- 正文最小 13px，表格单元格 14px。

**结构**
- 无 `linear-gradient` / `radial-gradient`，任何位置。
- 无 `backdrop-filter`，仅粘性顶栏例外。
- 圆角只用 `--radius-sm|md|lg|full`；面板与卡片上限 8px；嵌套内层 = 外层 − 2px。
- 页面内卡片、面板、表格 `box-shadow: none`；阴影只给下拉、模态、粘性表头。
- 无 `inset 0 1px 0 rgba(255,255,255,…)` 假高光，无 `.card::before` 顶部彩条。

**动效**
- 只过渡 `background-color` `border-color` `color` `opacity`，浮层可加 `box-shadow`。
- 无 `transform` 悬浮位移 / 缩放 / 旋转 / 扫光。
- 无入场关键帧，无 `nth-child` 延迟。
- 保留 `prefers-reduced-motion` 兜底。

**可访问性**
- 所有 `button` / `a` / `.nav-item` / 可点卡片必须有 `:focus-visible` 焦点环（`outline: 2px solid var(--focus-ring); outline-offset: 2px`）。
- 可点击卡片必须是 `<button>` 或 `<a>`，不是带 onclick 的 `<div>`。
- 悬浮后文字对比度只能升不能降；`--muted` 不允许作为任何 hover 文字色。
- 移动端（≤680px）触达目标 `min-height: 44px`。

**操作经济**
- 每个视图 `primary` 按钮计数 = 1。长页面底部可重复一次，同一视口内不得有第二个。
- 相邻按钮组最多一个实心按钮。
- 互斥流程拆成独立分组卡，各自一个主操作。
- 不可逆操作隔离进 `.danger-zone` 并要求二次确认。

**表格**
- `.table-scroll` 必须有 `max-height`，否则粘性表头不生效。
- 长文本固定列宽 + 省略号 + `title`，禁止 `overflow-wrap: anywhere`。
- 数字列右对齐。
- ≤820px 转卡片列表：给每个 `<td>` 加 `data-label="字段名"`，CSS 已处理其余部分。
- 空态区分「无数据」与「筛选无结果」两种文案。
- 加载用骨架行，保持行数与行高，不用整表遮罩。

**表单**
- 标签 13px/500 `--fg-secondary`，不用 `--muted`。
- 必填加 `<span class="required">*</span>`。
- 错误落在字段旁（`.field-error` + `aria-invalid="true"` + `aria-describedby`），只有表单级错误用顶部 `.alert`。
- 输入框实底 `--surface`，不用半透明背景。

## 文案规则

界面文案简体中文。产品名 `iCloud Privacy Mail` 与技术标识符（Apple ID、API、Token、UID、2FA）保持原文。日志用英文 + 结构化键值。

术语受硬约束，最常踩的几条：

| 必须 | 禁止 |
| --- | --- |
| 隐私邮箱 | 临时邮箱、一次性邮箱 |
| 验证码查询 | 领取、发放、消费 |
| 有效验证码 | 未消费验证码 |
| 同步游标 | 最新 UID、最大 UID |
| 用户开通 / 关闭 | 匿名注册、强制删除 |

完整表见 `DESIGN.md` §14.2。

按钮文案：动词 + 对象，≤6 字。字段说明只写不看就会填错的信息，不复述标签。危险操作明写不可逆和影响范围。

**不编造数据。** 无真实数值时省略整行，不显示 `+0` 或占位符。

## 交付前检查

```
[ ] 全文搜索 gradient → 0 命中
[ ] 全文搜索 backdrop-filter → 仅粘性顶栏
[ ] 全文搜索 font-weight → 无 >600 取值
[ ] 全文搜索 #  → 无字面十六进制色值
[ ] 圆角取值 ⊆ {4,6,8,999}px
[ ] 阴影取值 ⊆ {dropdown, modal, sticky}
[ ] 每个视图 primary 按钮计数 = 1
[ ] 所有可交互元素有 :focus-visible
[ ] 表格有 data-label，≤820px 转卡片
[ ] 数字容器有 tabular-nums
[ ] 术语与 §14.2 对照表一致
[ ] 无编造的指标或填充文案
[ ] Tab 键可走完主流程，焦点全程可见
[ ] 360 / 768 / 1024 / 1440 / 1920px 无横向滚动
[ ] Light / Dark × 舒展 / 紧凑 四组合下正文对比度 ≥4.5:1
```

## 参考

- `preview/` — 7 张聚焦预览卡，`preview/index.html` 是索引
- `examples/templates/` — 三个源模板逐字保留，用于核对真实产品结构与文案
- `context/provenance.md` — 每条结论的证据出处，含缺失证据说明
