# 证据出处

本设计系统的每条结论都可追溯到源仓库文件。本文件记录对应关系，便于后续核对与更新。

## 源

- 源项目：`前端分析与设计系统`（id `d1c88d7b-d06d-48a1-aeb6-eac01f10a5db`）
- 关联目录：`E:\Code\Ai\sub2api_git\iCloud-Privacy-Mail`
- 复制进本工作区的文件：`DESIGN.md`（32KB，源项目产出的设计分析）
- 本工作区补充读取的源仓库文件：
  - `internal/app/templates/index.html`（159,657 字节，主工作台）
  - `internal/app/templates/manage.html`（47,771 字节，账号数据管理）
  - `internal/app/templates/login.html`（12,577 字节，登录）
  - `CONTEXT.md`（2,263 字节，统一业务语言）

## 保留物与来源

| 本包路径 | 来源 | 处理方式 |
| --- | --- | --- |
| `examples/templates/workbench-index.html` | `internal/app/templates/index.html` | 逐字复制，未修改 |
| `examples/templates/account-manage.html` | `internal/app/templates/manage.html` | 逐字复制，未修改 |
| `examples/templates/login.html` | `internal/app/templates/login.html` | 逐字复制，未修改 |
| `context/source-domain-language.md` | `CONTEXT.md` | 逐字复制，未修改 |
| `assets/icons/nav-overview.svg` | `index.html` 概览导航项内联 SVG | 提取路径数据，补 `xmlns` 与尺寸属性 |
| `assets/icons/nav-session.svg` | `index.html` 登录态导航项内联 SVG | 同上 |
| `assets/icons/nav-create.svg` | `index.html` 创建邮箱导航项内联 SVG | 同上 |
| `assets/icons/nav-mailboxes.svg` | `index.html` 邮箱池导航项内联 SVG | 同上 |
| `assets/icons/caret-down.svg` | `index.html` `.user-menu-caret` | 同上 |
| `assets/icons/caret-right.svg` | `index.html` `.log-toggle-caret` | 同上 |

## 令牌出处

`colors_and_type.css` 与 `tokens.css` 的全部色值、字体栈、间距刻度、圆角、阴影、动效参数逐字取自 `index.html` 的 `:root` 块（约第 1 行起的第一个 `:root`）。三个模板已完成令牌化改造，实测状态：

| 检查项 | index.html | manage.html | login.html |
| --- | --- | --- | --- |
| `gradient` 出现次数 | 0 | 0 | 1 |
| `backdrop-filter` 出现次数 | 3（粘性顶栏） | 0 | 0 |
| `oklch` 出现次数 | 65 | 64 | 47 |
| `--accent*` 引用次数 | 36 | 24 | 15 |
| `--green*` 残留引用 | 4 | 3 | 0 |
| `focus-visible` 规则数 | 7 | 4 | 4 |
| `tabular-nums` 出现次数 | 6 | 4 | 0 |

`--green*` 的少量残留是向后兼容别名（`index.html` 第二个 `:root` 中有一段 `--panel`、`--card`、`--text` 之类的旧名映射层），不是新的品牌色定义。`login.html` 的 1 处 `gradient` 与 `index.html` 的 3 处 `backdrop-filter` 在源仓库中仍待清理，属于 DESIGN.md §12 落地清单的遗留项，不代表本设计系统允许它们。

## 文案与领域术语出处

| 本包内容 | 源证据 |
| --- | --- |
| 状态胶囊文案（可用 / 已使用 / 失败 / 停用） | `index.html` 的 `statusText()` 映射 |
| 状态类名（`status-available` 等） | `index.html` 的 `statusClass()` 实现 |
| 导航项与视图标题（概览 / 登录态 / 创建邮箱 / 邮箱池） | `index.html` `data-view` 与 `activeViewTitle` |
| 面板标题（Apple 登录 / 新接口（Apple Account）/ 旧接口（iCloud Web）/ 隐私邮箱池 / 运行配置 / 危险操作） | `index.html` 的 `<h2>` `<h3>` |
| 表头（标签 / 邮箱 / 状态 / 收件 / 验证码 / 操作） | `index.html` 邮箱池 `<th>` |
| 管理台表头（ID / 账号 / 角色 / 归属 / Apple ID / 备注 / 创建时间 / 最后登录） | `manage.html` `<th>` |
| 按钮文案（开始创建 / 同步已有邮箱 / 提交验证码 / 检测登录态 / 刷新状态 / 导出数据 / 一键删除码 / 清空搜索 / 去创建邮箱 等） | `index.html` `manage.html` 按钮文本 |
| 指标卡标签（登录账号 / 隐私邮箱 / 验证码邮件 / API 状态） | `index.html` `.card` 内 `<span>` |
| 表单标签（标签 / 备注 / 2FA 验证码 / 新接口验证码方式 / 自动检测间隔（分钟）/ 定时间隔分钟 / 每轮创建间隔秒 / 每页） | `index.html` `<label>` |
| 品牌副标题「隐私邮箱创建、同步和取码 API 工作台。」 | `index.html` `.brand .sub` |
| 登录页占位文案（例如 admin 或 user@example.com / 至少 6 位） | `login.html` `placeholder` |
| §14 术语对照表全部条目 | `CONTEXT.md` 的 Language 段 |

## 示例数据说明

`ui_kits/app/` 与 `preview/` 中的邮箱地址（`quiet-harbor-8412@icloud.com` 等）、验证码数值、计数和时间戳是**示例数据**，用于演示排版与对齐，不是从源系统导出的真实记录。源仓库的模板使用运行时注入数据，不含固定示例值。指标卡未显示任何变化量，因为无真实基线数据可用。

## 缺失的证据

- **无位图品牌资产**：源仓库不含 PNG / JPG / SVG logo、favicon、应用图标、托盘图标或头像。因此本包无 `build/` 目录。
- **无字体文件**：不含 woff / woff2 / ttf / otf。字体栈全部依赖系统字体，因此本包无 `fonts/` 目录。
- **无设计稿或参考截图**：源项目仅交接了 `DESIGN.md`，无 Figma 导出、浏览器快照或草图。
- **无既有设计系统**：源项目 `Active design system: (none)`，本包是首个成文系统。
