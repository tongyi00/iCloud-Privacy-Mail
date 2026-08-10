# iCloud Privacy Mail — 设计系统规范

> 目标风格：Linear + Vercel + Notion 的现代 SaaS 后台。克制、专业、有高级感。
> 适用范围：`internal/app/templates/` 下的 `index.html`（工作台）、`manage.html`（账号数据管理）、`login.html`（登录）。

---

## 1. 现状分析

分析对象为项目现有三个模板文件（共 4340 行，全部为内联 CSS + 内联 JS）：

| 文件 | 行数 | 体积 | 角色 |
| --- | --- | --- | --- |
| `index.html` | 3356 | 155 KB | 主工作台（概览 / 登录态 / 创建邮箱 / 邮箱池 / 日志） |
| `manage.html` | 884 | 38 KB | 账号数据管理（仪表盘 / 用户 / 邮箱 / 日志） |
| `login.html` | 100 | 5 KB | 登录与注册 |

### 1.1 三套互不兼容的设计令牌

三个文件各自定义了一份 `:root`，同名变量含义不同，视觉无法对齐：

- 圆角：`login.html` 用 24px / 12px，`manage.html` 用 20px / 16px / 12px / 10px / 9px，`index.html` 用 8px / 999px / 4px。同一个产品的三个页面像三个产品。
- 阴影：`index.html` 是 `0 14px 36px rgba(16,24,40,.06)`，`manage.html` 是 `0 28px 90px rgba(17,70,101,.12)`，量级差 2.5 倍。
- `manage.html` 中还存在未纳入令牌的硬编码色值：`.email { color:#0f766e }`、`th { background:#f7fbff }`、`.pill.gray/red/amber` 全套写死十六进制。

### 1.2 主题数量失控，且变量命名与语义脱节

`index.html` 定义了 10 套主题（mint / sky / graphite / sunset / violet / rose / forest / midnight / obsidian / aurora），每套重写约 30 个变量，叠加 `data-density` 的舒展 / 紧凑两档，等于 20 种组合需要回归验证——实际不可能验证完整。

更严重的是命名问题：品牌强调色变量叫 `--green`、`--green-2`，但在 sky 主题里它是蓝色 `#3b82f6`，在 sunset 里是橙色 `#fb7338`，在 violet 里是紫色 `#7c6cf0`。变量名在说谎，任何后续维护者读代码都会被误导。

### 1.3 廉价渐变与"玻璃拟态"堆叠

这是当前 UI 最直接的廉价感来源。渐变出现在几乎每一个层级：

```css
/* body 三层渐变叠加 */
background: linear-gradient(135deg, …), linear-gradient(180deg, var(--bg-soft) 0, var(--bg) 420px, …);
/* 每一张卡片都有玻璃叠层 */
.card { background: linear-gradient(145deg, rgb(var(--surface-rgb)/.94), rgb(var(--surface-rgb)/.62)); }
/* 所有激活态、所有主按钮、所有标签页 */
.nav-item.active, button.primary, .segmented button.active,
.log-tabs button.active, .account-tabs button.active { background: linear-gradient(135deg, var(--green), var(--green-2)); }
/* 卡片顶部装饰条 */
.card::before { background: linear-gradient(90deg, var(--green), …); }
/* 假高光 */
box-shadow: inset 0 1px 0 rgba(255,255,255,.35);
```

同时定义了 8 个半透明玻璃变量（`--soft-glass`、`--glass`、`--glass-strong`、`--glass-soft`、`--glass-hi`、`--glass-line`、`--sidebar-bg`、`--head-bg`），并在侧栏与顶栏施加 `backdrop-filter: blur(24px) saturate(165%)`、每个 `.panel` 施加 `blur(20px) saturate(150%)`。结果是：页面上没有一块纯净的实色表面，文字始终浮在噪声上，且大面积 `backdrop-filter` 在长列表滚动时有明显性能代价。

### 1.4 字重体系崩塌

代码中出现的字重：`700 / 800 / 850 / 900 / 950`。表头 `th` 是 900，表单 `label` 是 700，状态 `.pill` 是 900，`.account-name` 是 950，导航项是 800，连 12px 的 `.tool-panel-label` 都是 900。

当所有元素都加粗时，没有任何元素被强调——层级信息完全丢失，只剩下"整页都很吵"。

### 1.5 动效超出企业后台的合理范围

```css
button:hover  { transform: translateY(-2px); }   /* 每个按钮都在跳 */
button:active { transform: translateY(0) scale(.96); }
.card:hover       { transform: translateY(-3px); }
.quick-card:hover { transform: translateY(-4px); }
.nav-item:hover   { transform: translateX(3px); }
.nav-item:hover .nav-ico { transform: scale(1.1) rotate(-4deg); }
.quick-card::after { /* 斜向扫光，0.85s 位移动画 */ }
```

外加 `fadeUp` / `viewIn` / `popIn` / `navPop` / `modalIn` 五组关键帧和 `nth-child` 逐项延迟入场。一个需要长时间盯着表格取验证码的运维工具，不需要卡片扫光和图标旋转。`prefers-reduced-motion` 已正确处理，这是现有实现里做对的部分。

### 1.6 操作密度与主次关系失控

- 顶栏 `.content-head` 同时容纳：主题下拉、密度分段控件、运行配置按钮、刷新状态（primary）、版本徽章、更新按钮、用户徽章、账号数据按钮、退出按钮 —— 9 个交互元素挤在一行。
- 「登录态」视图的一条 toolbar 里有 6 个按钮（新接口登录 / 提交新接口验证码 / 旧接口登录 / 提交旧接口验证码 / 保存配置 / 检测登录态），其中"登录"是 primary，但四个按钮属于两条互相排斥的流程，视觉上无法区分。
- 「运行配置」弹窗一行放了 5 个按钮 + 2 个下拉（导出数据 primary、导出邮箱API、只导出邮箱、一键删除码 danger、关闭），且 `一键删除码` 这种不可逆的远端删除操作与普通导出并排。
- 概览页的 4 张 `quick-card` 中有 3 张跳转到侧栏已有的同名视图，属于重复导航。
- 全局日志面板在 `view-section` 之外，`min-height: 260px`，因此在每一个视图底部长期占据大块空间。

### 1.7 表格与表单的具体缺陷

- `th { position: sticky; top: 0 }` 写在 `.table-wrap { overflow:auto }` 内，但容器没有限定高度，粘性表头实际不生效。
- 表格 `min-width: 940px`（`manage.html` 为 900px），在 1120px 断点以下必然横向滚动，且没有为窄屏设计替代呈现。
- 行悬浮用 `color-mix(in srgb, var(--green) 6%, var(--card))`——把品牌色当成中性反馈色，10 套主题下会得到 10 种不同色相的行悬浮。
- 数字列没有 `tabular-nums`（只有 `.card strong` 有），验证码 / 计数在列中无法对齐。
- 表单标签是 `12px / weight 700 / color: var(--muted)`——尺寸偏小且用弱化色，长期填写体验差。
- 没有定义字段级错误态，所有反馈都退化成面板顶部一个 `.notice` 块；也没有必填标记规范。
- 焦点态只为 `input / textarea / select` 定义，**所有按钮和导航项都没有 `:focus-visible` 焦点环**，键盘操作不可用。
- `button:hover` 统一把背景改成 `color-mix(green 8%)`，导致次级按钮悬浮时"变成品牌色"，与主按钮抢注意力。
- `manage.html` 用几何字符与 emoji 充当图标（`✉ ⌂ ▣ ◉ ◎ ☰`），跨平台渲染不一致，也不可着色。

### 1.8 排版细节

- `body` 字体栈首位曾是 `Inter`，后续的"polish 层"又用 `-apple-system` 覆盖了一次——同一文件里两条 `body { font-family }` 规则，前一条完全无效。
- `.eyebrow` 对中文施加 `text-transform: uppercase` + `letter-spacing: .08em`，对中文无效果且拉散字距。
- `body { letter-spacing: .004em }` 对中文正文属于无意义的全局微调。

---

## 2. 新设计语言

### 2.1 三条原则

**结构优先于装饰。** 层级由留白、发丝边框和字号建立，不由渐变、阴影和玻璃建立。表面一律实色。

**颜色是信号，不是氛围。** 界面主体是中性灰阶；蓝色强调色只用于"当前所在"和"主操作"；绿 / 琥珀 / 红只用于表达邮箱与登录态的真实状态。一屏之内强调色出现不超过两次。

**为长时间注视而设计。** 这是一个每天要盯着邮箱池和验证码列表的运维工具。密度高于呼吸感，稳定高于活泼：不做入场动画，不做悬浮位移，只做 120ms 的颜色与边框过渡。

### 2.2 视觉基调

- 背景近白（`oklch(99% 0.002 240)`），卡片纯白，靠 1px 发丝边框区分层次。
- 圆角克制：容器 6px，控件 6px，输入 6px，状态点用胶囊形。**不做大圆角卡片墙。**
- 阴影只服务于浮层（下拉、弹窗、Toast）。页面内的卡片、面板、表格一律 0 阴影。
- 字重只用 400 / 500 / 600。600 是本设计系统的最粗字重，仅用于标题与关键数字。
- 全部数字使用 `tabular-nums`，邮箱 / Token / 验证码使用等宽字体。

### 2.3 主题策略

**10 套主题削减为 2 套：Light 与 Dark。** 保留 `data-density` 的舒展 / 紧凑两档（对企业后台是真实需求）。变量按语义命名（`--accent` 而非 `--green`），主题切换只覆盖语义层，组件层不感知主题。

---

## 3. 色彩系统

### 3.1 Light（默认）

```css
:root {
  color-scheme: light;

  /* 中性层 */
  --bg:              oklch(99% 0.002 240);   /* 页面底色 */
  --surface:         oklch(100% 0 0);        /* 卡片 / 面板 / 表格 */
  --surface-sunken:  oklch(97.5% 0.003 250); /* 表头 / 日志 / 只读区 */
  --surface-hover:   oklch(97% 0.004 250);   /* 行与列表项悬浮 */
  --fg:              oklch(18% 0.012 250);   /* 主文本 */
  --fg-secondary:    oklch(38% 0.012 250);   /* 次要文本 */
  --muted:           oklch(54% 0.012 250);   /* 标签 / 辅助说明 */
  --border:          oklch(92% 0.005 250);   /* 发丝边框 */
  --border-strong:   oklch(86% 0.006 250);   /* 输入框 / 分组边框 */

  /* 强调色（唯一品牌色） */
  --accent:          oklch(58% 0.18 255);    /* 实心填充底色 */
  --accent-fg:       oklch(100% 0 0);        /* 填充上的文字 */
  --accent-hover:    oklch(52% 0.18 255);    /* 填充悬浮 */
  --accent-text:     oklch(48% 0.19 255);    /* 白底上的强调色文字 / 链接 */
  --accent-subtle:   oklch(96% 0.02 255);    /* 选中行 / 弱强调背景 */
  --accent-border:   oklch(82% 0.08 255);

  /* 状态色：前景 / 背景 / 边框 三件套 */
  --success:         oklch(48% 0.13 155);
  --success-bg:      oklch(96.5% 0.03 155);
  --success-border:  oklch(85% 0.07 155);
  --warning:         oklch(52% 0.12 75);
  --warning-bg:      oklch(96.5% 0.04 85);
  --warning-border:  oklch(85% 0.08 80);
  --danger:          oklch(52% 0.19 25);
  --danger-bg:       oklch(96.5% 0.03 25);
  --danger-border:   oklch(86% 0.08 25);
  --neutral-badge:   oklch(45% 0.01 250);
  --neutral-bg:      oklch(96% 0.003 250);

  --focus-ring:      oklch(58% 0.18 255);
}
```

### 3.2 Dark

```css
[data-theme="dark"] {
  color-scheme: dark;

  --bg:              oklch(16% 0.008 255);
  --surface:         oklch(19.5% 0.009 255);
  --surface-sunken:  oklch(17.5% 0.008 255);
  --surface-hover:   oklch(23% 0.010 255);
  --fg:              oklch(96% 0.004 250);
  --fg-secondary:    oklch(80% 0.008 250);
  --muted:           oklch(65% 0.010 250);
  --border:          oklch(28% 0.010 255);
  --border-strong:   oklch(36% 0.012 255);

  --accent:          oklch(62% 0.17 255);
  --accent-fg:       oklch(16% 0.01 255);
  --accent-hover:    oklch(68% 0.16 255);
  --accent-text:     oklch(76% 0.13 255);
  --accent-subtle:   oklch(26% 0.05 255);
  --accent-border:   oklch(40% 0.10 255);

  --success:         oklch(76% 0.15 158);
  --success-bg:      oklch(26% 0.05 158);
  --success-border:  oklch(38% 0.08 158);
  --warning:         oklch(80% 0.14 82);
  --warning-bg:      oklch(28% 0.06 82);
  --warning-border:  oklch(40% 0.09 82);
  --danger:          oklch(72% 0.16 25);
  --danger-bg:       oklch(28% 0.07 25);
  --danger-border:   oklch(42% 0.11 25);
  --neutral-badge:   oklch(72% 0.008 250);
  --neutral-bg:      oklch(25% 0.008 250);

  --focus-ring:      oklch(70% 0.15 255);
}
```

### 3.3 使用规则

| 场景 | 用色 |
| --- | --- |
| 当前导航项 | `--accent-subtle` 背景 + `--accent-text` 文字 + 2px 左侧 `--accent` 指示条 |
| 主操作按钮 | `--accent` 填充 + `--accent-fg` 文字 |
| 白底上的链接 / 可点文本 | `--accent-text`（**不要用 `--accent`**，58% 明度对小字号对比不足） |
| 邮箱可用 / 登录态有效 | `--success` 三件套 |
| 待处理 / 需 2FA / 版本可更新 | `--warning` 三件套 |
| 失败 / 禁用 / 不可逆操作 | `--danger` 三件套 |
| 表格行悬浮 | `--surface-hover`（**中性色，不用品牌色染色**） |
| 表格行选中 | `--accent-subtle` |

对比度约束：
- `--fg` on `--surface` ≈ 15:1；`--muted` on `--surface` ≈ 5.1:1（正文与标签均达 4.5:1）。
- `--accent-fg` on `--accent` ≈ 4.6:1，仅用于 ≥14px 且 500 字重的按钮文字。
- 强调色作为小号文本时必须切到 `--accent-text`（on `--surface` ≈ 7.4:1）。
- 一屏之内 `--accent` 实心填充**最多出现 2 次**。

禁止事项：任何 `linear-gradient` / `radial-gradient` 作为按钮、导航、标签页、卡片、页面背景；`inset 0 1px 0 rgba(255,255,255,…)` 假高光；`backdrop-filter` 除粘性顶栏外一律移除。

---

## 4. 字体

```css
:root {
  --font-display: -apple-system, BlinkMacSystemFont, "SF Pro Display",
                  "PingFang SC", "HarmonyOS Sans SC", "MiSans",
                  "Microsoft YaHei UI", system-ui, sans-serif;
  --font-body:    -apple-system, BlinkMacSystemFont, "SF Pro Text",
                  "PingFang SC", "HarmonyOS Sans SC", "MiSans",
                  "Microsoft YaHei UI", system-ui, sans-serif;
  --font-mono:    ui-monospace, SFMono-Regular, "SF Mono",
                  "JetBrains Mono", Consolas, "Liberation Mono", monospace;
}
```

标题用 `--font-display`，正文与控件用 `--font-body`，邮箱地址 / API Token / 验证码 / 时间戳 / 日志用 `--font-mono`。

### 4.1 排版阶梯

| Token | 字号 / 行高 | 字重 | 字距 | 用途 |
| --- | --- | --- | --- | --- |
| `display` | 28 / 34 | 600 | -0.02em | 登录页主标题 |
| `h1` | 20 / 28 | 600 | -0.015em | 视图标题 |
| `h2` | 16 / 24 | 600 | -0.01em | 面板标题 |
| `metric` | 30 / 34 | 600 | -0.02em | 概览大数字（`tabular-nums`） |
| `body` | 14 / 22 | 400 | 0 | 正文、表格单元格、按钮 |
| `body-strong` | 14 / 22 | 500 | 0 | 强调正文、邮箱标签 |
| `label` | 13 / 18 | 500 | 0 | 表单标签（用 `--fg-secondary`，不用 `--muted`） |
| `caption` | 12 / 18 | 400 | 0 | 辅助说明、时间、页码 |
| `table-head` | 12 / 16 | 500 | 0.02em | 表头（仅拉丁字母可加字距） |
| `mono-sm` | 13 / 20 | 400 | 0 | 邮箱、Token、日志 |
| `code` | 16 / 20 | 500 | 0.02em | 验证码数值 |

规则：
- 字重只允许 400 / 500 / 600。删除全部 700 / 800 / 850 / 900 / 950。
- 不对中文使用 `text-transform: uppercase` 或 `letter-spacing`。`.eyebrow` 这一层级直接删除。
- 移除 `body { letter-spacing: .004em }` 全局字距。
- 所有数字容器加 `font-variant-numeric: tabular-nums`。
- 正文最小 13px，表格单元格 14px，`caption` 不低于 12px。

---

## 5. 间距规则

4px 基准栅格，只用以下刻度：

```css
--space-1: 4px;    --space-2: 8px;    --space-3: 12px;   --space-4: 16px;
--space-5: 20px;   --space-6: 24px;   --space-8: 32px;   --space-10: 40px;
--space-12: 48px;  --space-16: 64px;
```

| 场景 | 舒展（默认） | 紧凑 |
| --- | --- | --- |
| 页面左右边距 | 24px | 16px |
| 面板内边距 | 20px 24px | 14px 16px |
| 面板标题区 | 16px 24px | 12px 16px |
| 面板之间 | 16px | 12px |
| 卡片网格间隙 | 12px | 8px |
| 表格单元格 | 12px 16px | 8px 12px |
| 表单字段纵向间隙 | 16px | 12px |
| 标签到输入框 | 6px | 6px |
| 按钮组间隙 | 8px | 8px |
| 图标到文字 | 8px | 8px |

两档密度只改 padding 与 gap，**不改字号、不改圆角、不改边框宽度**。

侧栏宽度固定 240px（现为 286px，偏宽），窄于 1120px 时收为顶部横向导航。内容区最大宽度 1440px 居中。

---

## 6. 圆角规则

```css
--radius-sm:   4px;   /* 复选框、状态点、小徽章 */
--radius-md:   6px;   /* 按钮、输入框、下拉、分段控件项 */
--radius-lg:   8px;   /* 面板、卡片、弹窗、模态 */
--radius-full: 999px; /* 仅状态胶囊与计数点 */
```

规则：
- 面板与卡片一律 8px 上限。**不使用 12/16/20/24px 的大圆角卡片**，删除 `login.html` 的 24px 与 `manage.html` 的 20px / 16px。
- 表格本体 0 圆角；只在包裹面板上做 8px 并 `overflow: hidden`。
- 嵌套时内层圆角 = 外层 − 2px，不允许内层大于外层。
- 胶囊形只留给状态标签（`.pill`）和导航计数点，不用于按钮和标签页。

---

## 7. 阴影规则

页面内的静态元素**不使用阴影**，层级完全靠 `--border` 表达。阴影只用于真实浮起的浮层：

```css
--shadow-dropdown: 0 4px 12px oklch(20% 0.02 250 / 0.08),
                   0 1px 2px  oklch(20% 0.02 250 / 0.06);
--shadow-modal:    0 16px 40px oklch(20% 0.02 250 / 0.14),
                   0 2px 6px   oklch(20% 0.02 250 / 0.08);
--shadow-sticky:   0 1px 0     var(--border);  /* 粘性表头/顶栏的分隔线，非投影 */
```

Dark 模式下改用更强的边框而非更黑的阴影：`--shadow-dropdown: 0 4px 12px oklch(0% 0 0 / 0.4)`，并给浮层加 `border: 1px solid var(--border-strong)`。

必须删除：`--shadow: 0 14px 36px`、`--shadow-strong: 0 28px 72px`、`0 28px 90px`、主按钮的 `0 12px 26px` 光晕与悬浮 `0 18px 34px`、导航激活态 `0 12px 28px` 光晕、全部 `inset 0 1px 0 rgba(255,255,255,…)` 假高光。

### 动效预算

```css
--dur-fast: 100ms;  --dur:  140ms;
--ease:     cubic-bezier(0.2, 0, 0.2, 1);
```

只允许过渡 `background-color`、`border-color`、`color`、`opacity`、`box-shadow`（浮层）。
**禁止** `transform` 悬浮位移、缩放、旋转、扫光，以及入场关键帧与 `nth-child` 延迟。保留 `prefers-reduced-motion` 兜底。

---

## 8. Button 规范

### 8.1 尺寸

| 尺寸 | 高度 | 内边距 | 字号 | 用途 |
| --- | --- | --- | --- | --- |
| `sm` | 28px | 0 10px | 13px | 表格行内操作、分页 |
| `md` | 32px | 0 12px | 14px | 默认（工具栏、表单、弹窗） |
| `lg` | 36px | 0 16px | 14px | 登录页提交、空态主操作 |

圆角统一 `--radius-md`（6px），字重 500，`white-space: nowrap`，图标 16px、与文字间距 8px。
移动端（≤680px）触达目标不小于 44px：用 `min-height: 44px` 而非放大字号。

### 8.2 变体

| 变体 | 默认 | Hover | Active | 场景 |
| --- | --- | --- | --- | --- |
| `primary` | `--accent` 填充 / `--accent-fg` 文字 / 无边框 | 背景 `--accent-hover` | 背景再降 4% 明度 | 每个视图**唯一**主操作 |
| `secondary` | `--surface` / `--fg` / 1px `--border-strong` | 背景 `--surface-hover`，边框 `--muted` | 背景 `--surface-sunken` | 默认按钮，绝大多数场景 |
| `ghost` | 透明 / `--fg-secondary` / 无边框 | 背景 `--surface-hover`，文字 `--fg` | 背景 `--surface-sunken` | 关闭、取消、低频操作 |
| `danger` | `--surface` / `--danger` / 1px `--danger-border` | 背景 `--danger-bg` | 背景 `--danger-bg` 加深 | 删除、清理远端数据 |
| `danger-solid` | `--danger` 填充 / 白字 | 明度 −6% | 明度 −10% | 仅二次确认弹窗内的最终确认 |

关键修正：**次级按钮悬浮不得变成品牌色。** 删除现有 `button:hover { background: color-mix(green 8%) }`，改为中性的 `--surface-hover`。悬浮时前景色只能变深、不能变浅，`--muted` 不允许作为任何 hover 文字色。

### 8.3 状态

```css
.btn:focus-visible {
  outline: 2px solid var(--focus-ring);
  outline-offset: 2px;
}
.btn:disabled {
  background: var(--surface-sunken);
  color: var(--muted);
  border-color: var(--border);
  cursor: not-allowed;
}
```

`:focus-visible` 是必需项——现有实现所有按钮和导航项都缺失焦点环，键盘不可用。
加载态：按钮内替换为 14px 旋转指示器 + 文案不变 + `aria-busy="true"`，宽度不得跳变。

### 8.4 操作经济

- 一个视图内 `primary` 只出现一次。长页面底部可重复一次，但同一视口内不得出现第二个。
- 相邻按钮组最多一个实心按钮，其余为 `secondary` / `ghost`。
- 针对现有问题的具体处置：
  - 顶栏保留「刷新状态」（`secondary`）+ 用户菜单（收进下拉：账号数据 / 主题 / 密度 / 版本 / 退出），去掉常驻的主题下拉与密度分段控件。
  - 「登录态」视图的 6 按钮按流程拆成两张分组卡片（新接口 / 旧接口），每张卡各一个 `primary` 登录按钮 + 一个 `secondary` 提交验证码按钮。
  - 「运行配置」弹窗把导出类操作归为一组（一个 `primary` 导出 + 格式/范围下拉），`一键删除码` 移到弹窗底部独立的危险操作区，并要求二次确认。

---

## 9. Card 规范

卡片是**信息容器，不是装饰物**。

```css
.card {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);   /* 8px */
  padding: var(--space-5);
  box-shadow: none;                   /* 页面内卡片永不投影 */
}
```

- 无渐变背景、无 `::before` 顶部彩条、无玻璃叠层、无 `backdrop-filter`。
- 静态卡片无 hover 效果。仅当整卡可点击时：`border-color: --border-strong` + `background: --surface-hover`，**不位移、不投影**。
- 可点击卡片必须是 `<button>` 或 `<a>`，并带 `:focus-visible` 焦点环。

### 9.1 指标卡（概览）

```
┌─────────────────────────┐
│ 隐私邮箱            ← caption 12px --muted
│ 1,284               ← metric 30px/600 tabular-nums --fg
│ 近 24 小时 +32      ← caption 12px --success（无变化时省略）
└─────────────────────────┘
```
高度 96px，四列网格 gap 12px，≤1120px 两列，≤680px 一列。数字下方的变化量只在有真实数据时显示——**没有数据就不显示，不编造**。

### 9.2 面板（Panel）

表格与表单的容器，取代现有 `.panel`：

```css
.panel { background: var(--surface); border: 1px solid var(--border);
         border-radius: var(--radius-lg); overflow: hidden; }
.panel-head { display: flex; justify-content: space-between; align-items: center;
              gap: var(--space-4); padding: var(--space-4) var(--space-6);
              border-bottom: 1px solid var(--border); background: var(--surface); }
```

标题区背景与主体同为 `--surface`（去掉现有的渐变头部），仅用下边框分隔。标题 `h2` 16px/600，说明文字 13px `--muted`，工具按钮右对齐。

### 9.3 需要删除的卡片

概览页的 4 张 `quick-card` 与侧栏导航重复，且带扫光动效——删除。概览页只保留指标卡 + 最近活动列表。

---

## 10. Table 规范

表格是本产品的核心界面（邮箱池、用户列表、邮件列表），优先级最高。

### 10.1 结构与尺寸

| 元素 | 规格 |
| --- | --- |
| 行高 | 舒展 48px / 紧凑 40px |
| 单元格内边距 | 舒展 12px 16px / 紧凑 8px 12px |
| 表头 | 高 40px，背景 `--surface-sunken`，文字 12px/500 `--muted` |
| 分隔线 | 仅行间 `border-bottom: 1px solid var(--border)`；**无竖向网格线** |
| 斑马纹 | 不使用（靠行悬浮定位） |
| 行悬浮 | `background: var(--surface-hover)`，中性色 |
| 行选中 | `background: var(--accent-subtle)` + 左侧 2px `--accent` |
| 首列 | 左内边距与面板对齐（16px / 24px） |

### 10.2 粘性表头（修正现有缺陷）

现有 `th { position: sticky; top: 0 }` 因容器无高度而失效。正确写法：

```css
.table-scroll {
  max-height: calc(100vh - 280px);   /* 必须有高度约束，sticky 才生效 */
  overflow: auto;
  overscroll-behavior: contain;
}
.table-scroll thead th {
  position: sticky; top: 0; z-index: 2;
  background: var(--surface-sunken);
  box-shadow: var(--shadow-sticky);  /* 1px 分隔线，非投影 */
}
```

### 10.3 列规范

| 内容类型 | 对齐 | 字体 | 说明 |
| --- | --- | --- | --- |
| 标签 / 名称 | 左 | `body-strong` 14/500 | 可截断，`title` 显示全文 |
| 邮箱地址 | 左 | `mono-sm` 13px `--fg` | 单行 `text-overflow: ellipsis`，行内复制按钮 |
| 状态 | 左 | 状态胶囊 | 见 10.4 |
| 数量 / 计数 | 右 | `tabular-nums` | 右对齐 |
| 时间 | 左 | `mono-sm` `--muted` | 相对时间 + `title` 绝对时间 |
| 验证码 | 左 | `code` 16px/500 `--accent-text` | 点击复制，成功后 2s 内显示"已复制" |
| 操作 | 右 | `btn sm ghost` | 最多 2 个常用 + 「⋯」下拉收纳其余 |

**长文本（邮箱、Token）用固定列宽 + 省略号 + `title`，禁止 `overflow-wrap: anywhere` 把单元格撑成多行**——现有 `.mailbox-code-cell` / `.code-inline-result` 的换行方案会让行高参差不齐。

### 10.4 状态胶囊

```css
.pill { display: inline-flex; align-items: center; gap: 6px;
        height: 22px; padding: 0 8px; border-radius: var(--radius-full);
        font-size: 12px; font-weight: 500; white-space: nowrap;
        border: 1px solid; }
```

| 状态 | 前景 / 背景 / 边框 |
| --- | --- |
| 有效 / 活跃 / 可用 | `--success` / `--success-bg` / `--success-border` |
| 待处理 / 需 2FA / 同步中 | `--warning` / `--warning-bg` / `--warning-border` |
| 失败 / 已禁用 | `--danger` / `--danger-bg` / `--danger-border` |
| 未激活 / 无数据 | `--neutral-badge` / `--neutral-bg` / `--border` |

胶囊左侧可加 6px 圆点强化状态，**不依赖颜色单独表意**——每个胶囊都带文字。

### 10.5 空态、加载与响应式

- 空态：表格区域居中显示 16px/500 标题 + 13px `--muted` 说明 + 一个 `secondary` 引导按钮，高度不少于 160px。区分"无数据"与"筛选无结果"两种文案。
- 加载：骨架行（`--surface-sunken` 矩形，高 12px，圆角 4px）保持行数与行高，避免布局跳动。不用整表遮罩。
- 响应式：≤820px 时表格转为卡片列表——每条记录一张卡，标签 + 邮箱为标题行，状态胶囊右上，其余字段为「键：值」两列，操作按钮置底。**不保留 940px 的横向滚动作为窄屏唯一方案。**
- 分页固定在面板底部，左侧"共 N 条"，右侧每页条数下拉 + 上/下页 `btn sm secondary` + 页码。

---

## 11. Form 规范

### 11.1 字段结构

```
标签 *                    ← 13px/500 --fg-secondary（非 --muted）
┌──────────────────────┐
│ 输入内容              │  ← 32px/36px 高，14px 文字
└──────────────────────┘
辅助说明文字              ← 12px --muted
```

```css
.field { display: grid; gap: 6px; }
.field-label { font-size: 13px; font-weight: 500; color: var(--fg-secondary); }
.field-label .required { color: var(--danger); margin-left: 2px; }  /* 星号 */

.input, .select, .textarea {
  width: 100%;
  min-height: 32px;                 /* 舒展 36px；移动端 44px */
  padding: 0 12px;                  /* textarea: 8px 12px */
  font: 400 14px/22px var(--font-body);
  color: var(--fg);
  background: var(--surface);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  transition: border-color var(--dur) var(--ease), box-shadow var(--dur) var(--ease);
}
.input::placeholder { color: var(--muted); }
.input:hover { border-color: var(--muted); }
.input:focus-visible {
  border-color: var(--accent);
  box-shadow: 0 0 0 3px oklch(58% 0.18 255 / 0.16);
  outline: none;
}
.input:disabled { background: var(--surface-sunken); color: var(--muted); cursor: not-allowed; }
.input[aria-invalid="true"] { border-color: var(--danger); }
.input[aria-invalid="true"]:focus-visible { box-shadow: 0 0 0 3px oklch(52% 0.19 25 / 0.16); }
```

单色实底输入框，**去掉现有的 `--input-bg` 半透明背景与 focus 时的背景切换**。

### 11.2 字段级校验（现有缺失）

现有实现所有反馈都退化成面板顶部一个 `.notice`。新规范：

```html
<div class="field">
  <label class="field-label" for="appleId">Apple ID<span class="required">*</span></label>
  <input class="input" id="appleId" aria-invalid="true" aria-describedby="appleId-err">
  <p class="field-error" id="appleId-err">请输入完整的 Apple ID 邮箱地址</p>
</div>
```

`.field-error`：12px/400 `--danger`，与输入框间距 6px，出现时不引起布局跳动（预留 18px 行高空间）。
表单级错误（接口失败、2FA 超时）才使用面板顶部提示条。

### 11.3 提示条（Alert）

```css
.alert { display: flex; gap: 10px; padding: 12px 14px;
         border: 1px solid; border-radius: var(--radius-md);
         font-size: 13px; line-height: 1.6; }
```
四种语义复用状态色三件套（info 用 `--accent-*`）。左侧 16px 图标，标题 13px/500，正文 13px/400。
**不使用"左侧彩色竖条 + 圆角卡片"的老套式样。**

### 11.4 布局与分组

- 单列表单最大宽度 480px；双列网格（`1fr 1fr`，gap 16px）用于 Apple ID / 密码这类成对字段，跨列字段用 `full`。
- 逻辑分组用 `--space-8` 纵向间距 + 13px/500 分组标题 + 1px 上边框，不用嵌套卡片。
- 提交区在表单底部，与内容间距 24px，上边框分隔：左侧 `primary` 提交，右侧或其后跟 `ghost` 取消。
- 复选框 / 单选：16px 方形（`--radius-sm`）/ 圆形，选中态 `--accent` 填充 + 白色勾，标签 14px `--fg`，整体点击区高 32px。
- 开关（Switch）：36×20px，关闭态 `--border-strong`，开启态 `--accent`，用于「开启自动检测」这类即时生效项。
- 数字输入（间隔分钟、每页条数）宽度 88px，右对齐，`tabular-nums`。

### 11.5 登录页

- 单列居中卡片，宽 400px，`--surface` + 1px `--border` + `--radius-lg` + `--shadow-modal`。
- 删除现有的双栏 hero + `radial-gradient` 背景 + 24px 圆角，页面背景为纯 `--bg`。
- 登录 / 注册用下划线式标签页（激活项 `--fg` 文字 + 2px `--accent` 下边框），不用胶囊填充按钮。
- 提交按钮 `primary lg` 满宽，是页面唯一实心按钮。

---

## 12. 落地清单

按顺序执行，每步可独立验证：

1. **抽出令牌层。** 新建 `internal/app/templates/tokens.css`（或单个 `<style>` 片段），三个模板共用；删除各文件重复的 `:root`，删除 `manage.html` 全部硬编码色值。
2. **主题削减。** 10 套主题 → `light` / `dark`；`--green*` 全量重命名为 `--accent*`；保留 `data-density`。
3. **去装饰。** 全量移除 `linear-gradient` / `radial-gradient` / `backdrop-filter`（仅保留粘性顶栏）/ `inset` 假高光 / `.card::before` 彩条 / `--shadow` 与 `--shadow-strong`。
4. **字重归一。** 700–950 全部映射到 400 / 500 / 600；删除 `.eyebrow` 层级与全局 `letter-spacing`；清理 `body { font-family }` 的重复声明。
5. **动效归一。** 删除 5 组入场关键帧、`nth-child` 延迟、全部 `transform` 悬浮效果与 `quick-card::after` 扫光。
6. **补焦点环。** 为所有 `button` / `a` / `.nav-item` / 可点卡片补 `:focus-visible`。
7. **圆角归一。** 24/20/16/12/10/9px → 4/6/8px；胶囊仅留状态标签与计数点。
8. **表格重构。** `.table-scroll` 加高度约束修复粘性表头；行悬浮改中性色；数字列右对齐 + `tabular-nums`；长文本改省略号 + `title`；≤820px 转卡片列表；补空态与骨架屏。
9. **表单重构。** 标签升到 13px/500 `--fg-secondary`；补必填星号、`aria-invalid`、`.field-error`；输入框改实底。
10. **操作经济。** 顶栏 9 个交互元素收敛为「刷新状态」+ 用户下拉；登录态视图 6 按钮拆为两张分组卡；运行配置弹窗把「一键删除码」隔离进危险区并加二次确认。
11. **删冗余。** 移除概览页 4 张 `quick-card`；全局日志面板改为可折叠或独立视图，不再常驻每个视图底部。

### 验收标准

- 全站不存在 `gradient`、`backdrop-filter`（顶栏除外）、`transform` 悬浮、`font-weight > 600`。
- 圆角取值集合 ⊆ `{4, 6, 8, 999}px`；阴影取值集合 ⊆ `{dropdown, modal, sticky}`。
- 每个视图 `primary` 按钮计数 = 1。
- 键盘可完成：登录 → 保存登录态 → 创建邮箱 → 在邮箱池取码复制，全程焦点可见。
- 360 / 390 / 768 / 1024 / 1440 / 1920px 无横向滚动（表格在 ≤820px 已转卡片列表）。
- Light / Dark × 舒展 / 紧凑 4 种组合下，正文对比度 ≥4.5:1，胶囊与图标 ≥3:1，且 hover 后对比度不下降。
