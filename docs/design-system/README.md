# iCloud Privacy Mail 设计系统

面向自托管隐私邮箱运维面板的设计系统。风格取向是 Linear / Vercel / Notion 一类的现代 SaaS 后台：实色表面、发丝边框、克制的动效、唯一强调色。

系统从真实实现中提取，不是凭空设计的：源项目 `iCloud-Privacy-Mail`（Go 后端 + 服务端渲染 HTML 模板）的三个模板逐字保留在 `examples/templates/`，全部令牌与文案都可追溯到那里。

## Product Overview

源产品是一个自托管的隐私邮箱运维工作台（a self-hosted privacy-mail operations tool）。它对接 Apple 的隐私邮箱能力，provides 三件事：保存并检测 Apple 登录态、批量创建隐私邮箱、通过 IMAP 幂等读取邮箱里的验证码。使用者是自己部署这套服务的运维者，不是消费者。

产品由三个界面构成，本设计系统覆盖全部三个：

| 界面 | 源模板 | 职责 |
| --- | --- | --- |
| 工作台 | `internal/app/templates/index.html`（159KB） | 概览指标、登录态管理、创建邮箱、邮箱池取码 |
| 账号数据管理 | `internal/app/templates/manage.html`（47KB） | 用户列表、邮箱数据、运行配置、不可逆的清理操作 |
| 登录 | `internal/app/templates/login.html`（12KB） | 登录与注册 |

核心使用节奏决定了设计取向：用户每天长时间盯着邮箱池等验证码到达，反复扫读表格并复制验证码。所以这套系统 designed for 长时间注视——密度高于呼吸感，稳定高于活泼，信息层级靠留白与发丝边框而不是渐变与动效。

## 三条原则

**结构优先于装饰。** 层级由留白、1px 发丝边框和字号建立，不由渐变、阴影和玻璃建立。表面一律实色。

**颜色是信号，不是氛围。** 界面主体是中性灰阶；蓝色强调色只用于「当前所在」和「主操作」，一屏之内实心填充最多出现两次；绿 / 琥珀 / 红只表达邮箱与登录态的真实状态。

**为长时间注视而设计。** 不做入场动画，不做悬浮位移，只做 140ms 的颜色与边框过渡。

## Source Context

源证据是可直接读取的本地 repository（local folder，非 GitHub 远端），交接说明见 `context/source-context.md`。设计结论的来源分三类：

- **源模板逐字保留** — `examples/templates/` 下三个文件是源模板的完整副本（159KB / 47KB / 12KB），未做删减，用于核对真实结构与文案。
- **领域语言逐字保留** — `context/source-domain-language.md` 是源仓库统一业务语言的副本，术语是硬约束。
- **证据出处清单** — `context/provenance.md` 记录每条结论对应的源文件与实测数据（含令牌采用率的 grep 计数），以及缺失的证据。

缺失的证据（不编造，直接记录）：源仓库无任何位图资产（无 favicon、应用图标、托盘图标）、无字体文件、无设计源文件、无既有设计系统文档。因此本包无 `build/`、无 `fonts/`，这两处缺失在 `context/provenance.md` 与 `preview/brand-assets.html` 中均有说明。

## Package Contents

```
DESIGN.md                          设计决策的唯一权威来源（17 节）
README.md                          本文件
SKILL.md                           代理使用指令与检查清单
colors_and_type.css                颜色与排版语义层（Light / Dark，oklch + sRGB 回退）
tokens.css                         间距 / 圆角 / 阴影 / 动效 / 密度令牌

ui_kits/app/
  README.md                        套件说明：结构、组件、复用流程、设计约束
  index.html                       套件入口：组合示例 + 屏幕索引 + 组件清单
  components.css                   聚合入口，@import components/ 全部文件
  components/app-shell.css         AppShell / Sidebar / 粘性顶栏
  components/buttons.css           Button 五变体 × 三尺寸
  components/cards-panels.css      Card / MetricCard / Panel
  components/data-table.css        DataTable / 状态胶囊 / 空态 / 骨架 / 分页
  components/forms.css             Field / Input / Switch / Alert / Tabs
  components/overlays.css          DangerZone / Dropdown / Modal / Log
  workbench.html                   工作台：概览 / 登录态 / 创建邮箱 / 邮箱池
  account-manage.html              账号数据管理 + 运行配置弹窗
  login.html                       登录页

preview/                           7 张聚焦预览卡（清单见下节 Preview Manifest）
  preview.css                      预览卡共享外壳
  index.html                       预览索引与审阅顺序

assets/icons/                      源实现提取的品牌图标（preserved brand assets）
  nav-overview.svg                 概览（四宫格）
  nav-session.svg                  登录态（放大镜）
  nav-create.svg                   创建邮箱（方框加号）
  nav-mailboxes.svg                邮箱池（收件托盘）
  caret-down.svg                   下拉展开
  caret-right.svg                  折叠展开

examples/templates/                源模板逐字保留（preserved source components）
  workbench-index.html             159KB
  account-manage.html              47KB
  login.html                       12KB

context/
  source-context.md                源项目交接说明
  source-domain-language.md        源仓库统一业务语言（逐字保留）
  provenance.md                    每条结论的证据出处与缺失说明
```

关于 `assets/`、`build/`、`fonts/`：`assets/icons/` 下 6 个 SVG 是从源实现提取的 preserved brand 图标，是源仓库里唯一存在的可复用视觉资产。无 `build/`（源仓库无 runtime 图标资源），无 `fonts/`（字体栈全部依赖系统字体，源仓库无 font 文件）。

## Preview Manifest

七张聚焦预览卡，按建议审阅顺序排列：

| 顺序 | 文件 | 内容 |
| --- | --- | --- |
| 01 | `preview/applied-surfaces.html` | 已应用界面。内嵌加载 `ui_kits/app/` 三个真实屏幕与保留的源模板，可直接交互对比。**先看这张。** |
| 02 | `preview/colors-primary.html` | 颜色。从计算样式实时读取令牌值，可切换 Light / Dark，含对比度表 |
| 03 | `preview/typography-specimens.html` | 排版。11 级阶梯、三套字体栈、表格数字对齐 |
| 04 | `preview/spacing-tokens.html` | 间距与密度。4px 栅格、双档密度实测值、布局断点 |
| 05 | `preview/radius-and-shadows.html` | 圆角与阴影。四档圆角、三种浮层阴影、已删除的源取值、动效预算 |
| 06 | `preview/components-buttons.html` | 组件。按钮变体与状态、胶囊、表单与校验、提示条、表格四态 |
| 07 | `preview/brand-assets.html` | 品牌资产。文字标识、6 个提取图标、领域术语语气表。**再看这张。** |

`preview/index.html` 是索引页，列出同样的顺序。

## Review Workflow

审阅这套系统：先 open `preview/applied-surfaces.html`，它内嵌加载 `ui_kits/app/` 的真实屏幕与 `examples/templates/` 的源模板，可以并列 inspect 收敛前后的差异；再看 `preview/brand-assets.html` 核对 `assets/` 里保留的图标；然后按 Preview Manifest 的 02–06 逐张 review 令牌层。

复用这套系统：先读 `DESIGN.md`（权威规范）与 `context/source-domain-language.md`（术语硬约束），再按顺序引入三个 CSS，最后从 `ui_kits/app/` copy 最接近的屏幕改内容，不要从零写 CSS。

```html
<link rel="stylesheet" href="colors_and_type.css">   <!-- 颜色 + 排版，Light / Dark -->
<link rel="stylesheet" href="tokens.css">             <!-- 间距 / 圆角 / 阴影 / 动效 / 密度 -->
<link rel="stylesheet" href="ui_kits/app/components.css">  <!-- 组件层（@import components/ 全部文件） -->
```

只需要单类组件时，跳过聚合入口直接引用 `ui_kits/app/components/data-table.css` 这样的单个文件。

源项目无构建链，模板是单文件内联 CSS。要落到那种环境，把文件内容按同样顺序粘进第一个 `<style>` 即可，不需要任何工具链。

主题与密度挂在根元素上：

```html
<html data-theme="light" data-density="comfortable">
<!-- data-theme: light | dark -->
<!-- data-density: comfortable | compact -->
```

代理接入这套系统时读 `SKILL.md`，它列了硬性规则与交付前检查清单。

## 核心约束速查

| 维度 | 规则 |
| --- | --- |
| 主题 | 只有 Light / Dark 两套（源实现有 10 套） |
| 强调色 | 一个，`--accent`。实心填充一屏内 ≤2 次；小号文字用 `--accent-text` |
| 字重 | 只允许 400 / 500 / 600 |
| 圆角 | 取值集合 ⊆ {4, 6, 8, 999}px。面板与卡片上限 8px |
| 阴影 | 只服务浮层，∈ {dropdown, modal, sticky}。页面内卡片无阴影 |
| 渐变 | 全站禁止 |
| `backdrop-filter` | 仅粘性顶栏 |
| 动效 | 只过渡颜色类属性。禁止 transform 悬浮、入场关键帧 |
| 焦点环 | 所有 button / a / nav-item / 可点卡片必须有 `:focus-visible` |
| 主操作 | 每个视图 `primary` 计数 = 1 |
| 表格窄屏 | ≤820px 转卡片列表，不用横向滚动兜底 |
| 数字 | 全部 `tabular-nums`；邮箱 / Token / 验证码用等宽字体 |

完整规则见 `DESIGN.md`。反模式清单在 §15，其中 22 条附有源实现的具体证据。

## 文案约束

界面文案是简体中文，但术语受源仓库统一业务语言约束，不能自由替换。最容易踩的一条：**验证码查询是幂等读取**，所以不能写「领取」「发放」「消费」。完整对照表见 `DESIGN.md` §14.2 与 `context/source-domain-language.md`。

日志例外：日志用英文 + 结构化键值，便于分析工具处理。

## 示例数据说明

`ui_kits/app/` 与 `preview/` 中的邮箱地址、验证码、计数、时间戳都是示例值，用于演示排版与对齐，不是源系统的真实记录。指标卡不显示变化量，因为无真实基线数据。这是有意为之：本系统不编造数据。
