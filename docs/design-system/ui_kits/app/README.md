# 应用界面套件 / Applied UI Kit

iCloud Privacy Mail 设计系统的已应用界面层。三个屏幕不是静态截图式 mock，而是可交互的真实实现，直接消费包根目录的令牌文件。

## Structure / 文件结构

```
ui_kits/app/
  index.html                  套件入口：组合示例 + 屏幕索引 + 组件文件清单
  components.css              聚合入口，@import components/ 下全部文件
  components/
    app-shell.css             AppShell / Sidebar / 侧栏导航 / 粘性顶栏
    buttons.css               Button 五变体 × 三尺寸
    cards-panels.css          Card / MetricCard / Panel
    data-table.css            DataTable / 状态胶囊 / 空态 / 骨架 / 分页
    forms.css                 Field / Input / Switch / Alert / Tabs
    overlays.css              DangerZone / Dropdown / Modal / Log
  workbench.html              工作台：概览 / 登录态 / 创建邮箱 / 邮箱池
  account-manage.html         账号数据管理 + 运行配置弹窗
  login.html                  登录页
```

`index.html` 是入口页：它逐个 `<link>` 引入 `components/` 下六个文件，并用它们组合出一块真实运行的界面（侧栏 + 粘性顶栏 + 指标卡 + 数据表格 + 提示条），页内可切换主题与密度、可点击表格行切换选中态。

### Components / 组件角色

| 组件 | 文件 | 职责 |
| --- | --- | --- |
| AppShell | `components/app-shell.css` | 240px 侧栏 + 内容区两栏栅格，≤1120px 折叠为顶部横向导航 |
| Sidebar | `components/app-shell.css` | 品牌区、`.side-nav` 导航项、当前项左侧指示条、`.count-dot` 计数 |
| Button | `components/buttons.css` | `.primary` `.secondary` `.ghost` `.danger` `.danger-solid` × `.sm` `.lg` |
| MetricCard | `components/cards-panels.css` | 概览页四张指标卡，caption + metric 两行，无编造的变化量 |
| Panel | `components/cards-panels.css` | `.panel-head`（标题 + 描述 + 单个操作）+ `.panel-body` |
| DataTable | `components/data-table.css` | 粘性表头、中性悬浮、强调色选中、`data-label` 驱动的窄屏卡片化 |
| Field | `components/forms.css` | 标签 / 输入 / 字段级错误三段结构，错误行预留高度 |
| Composer | `components/forms.css` | 创建邮箱与登录表单的输入组合（`.field-grid` + `.form-actions`） |
| DangerZone | `components/overlays.css` | 不可逆操作隔离区，配两步确认 |
| Modal | `components/overlays.css` | 运行配置弹窗，Esc 与点击遮罩关闭 |

## Usage / 如何复用

需要一个完整屏幕时，从本目录复制最接近的那个再替换内容，不要从零写 CSS：

| 需求 | 起点 |
| --- | --- |
| 带侧栏的多视图工作台 | `workbench.html` |
| 管理后台 + 弹窗 + 危险操作 | `account-manage.html` |
| 登录 / 认证页 | `login.html` |
| 多屏索引 / 组合示例 | `index.html` |

整包引入（顺序有意义，组件层依赖前两个令牌文件）：

```html
<link rel="stylesheet" href="../../colors_and_type.css">
<link rel="stylesheet" href="../../tokens.css">
<link rel="stylesheet" href="components.css">
```

只需要其中一类组件时，跳过聚合入口，直接引用单个文件：

```html
<link rel="stylesheet" href="../../colors_and_type.css">
<link rel="stylesheet" href="../../tokens.css">
<link rel="stylesheet" href="components/data-table.css">
```

源项目无构建链（Go 服务端渲染单文件模板），要落到那种环境就按同样顺序把文件内容依次粘进第一个 `<style>`，不需要任何工具链。

图标来自 `../../assets/icons/`（6 个 24×24 描边 SVG，从源实现提取）。三个屏幕内联使用相同的路径数据，因此在 `file://` 下也能正常渲染。

## 可交互的部分

不是装饰性 mock，下列行为真实可用：

**index.html**
- 组合示例内切换 Light / Dark 主题与舒展 / 紧凑密度
- 点击示例表格行切换选中态（`aria-selected`）

**workbench.html**
- 侧栏四项切换视图，标题同步更新，当前视图写入 `localStorage`（`icpm.view`）
- 用户下拉菜单开合，点击外部区域关闭
- 菜单内切换主题与密度，观察同一组件在四种组合下的表现
- 邮箱池点击验证码复制到剪贴板，原位显示「已复制」2 秒后消失
- 点击表格行切换选中态，可对比中性悬浮与强调色选中
- 「旧接口」卡片展示字段级错误态（`aria-invalid` + `.field-error`）

**account-manage.html**
- 默认紧凑密度，可与工作台的舒展密度直接对比
- 「运行配置」打开模态，Esc 或点击遮罩关闭
- 危险操作区两步确认：第一步只展开确认按钮，不执行任何操作；可取消
- 「登录账号数据」面板展示骨架加载态

**login.html**
- 登录 / 注册标签页切换，提交按钮文案与 `autocomplete` 同步变化
- 提交空表单触发字段级校验，焦点跳到第一个错误字段
- 校验通过后进入加载态（`aria-busy`，宽度不跳变），随后跳转工作台

## Design Notes / 设计约束与源实现基础

本套件的 layout、colors、typography 全部来自包根目录的 tokens，组件文件内不定义任何字面色值。三个屏幕在结构上对应 `../../examples/templates/` 中逐字保留的源模板（based on 真实 Go 模板实现），但按 `../../DESIGN.md` §12 落地清单做了以下收敛：

| 源实现 | 本套件 |
| --- | --- |
| 顶栏 9 个交互元素挤在一行 | 「刷新状态」+ 用户下拉，低频操作收进菜单 |
| 登录态视图一条 toolbar 放 6 个按钮 | 拆成新接口 / 旧接口两张分组卡，各一个主操作 |
| 「一键删除码」与导出按钮并排 | 隔离进 `.danger-zone`，两步确认 |
| 概览页 4 张快捷卡（3 张与侧栏重复） | 删除，只保留指标卡 + 最近活动 |
| 全局日志 `min-height: 260px` 常驻 | 收为底部固定高度日志条 |
| 表格 `min-width: 940px` 横向滚动 | ≤820px 转卡片列表（`data-label` 驱动） |
| 粘性表头因容器无高度而失效 | `.table-scroll` 加 `max-height` |
| 按钮与导航项无焦点环 | 全部补 `:focus-visible` |
| emoji 与几何字符当图标 | 内联 SVG 描边图标 + `currentColor` |
| 10 套主题 | Light / Dark 两套 |

术语受源仓库统一业务语言约束：验证码查询是幂等读取，界面不写「领取」「发放」「消费」。对照表见 `../../DESIGN.md` §14.2 与 `../../context/source-domain-language.md`。

## 示例数据

邮箱地址、验证码、计数、时间戳均为示例值，用于演示排版与对齐，不是源系统的真实记录。指标卡不显示变化量，因为无真实基线数据可用。

## 相关

- `../../DESIGN.md` — 规范全文
- `../../SKILL.md` — 硬性规则与交付前检查清单
- `../../preview/applied-surfaces.html` — 内嵌加载本套件三个屏幕，可与源模板并列对比
- `../../examples/templates/` — 三个源模板逐字保留
