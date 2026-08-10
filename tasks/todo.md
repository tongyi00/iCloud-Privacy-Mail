# 多邮箱验证码错配排查

- [x] 目标与验收：复现“两个不同邮箱的最新验证码均显示为 202123”，并定位负责错配的共享状态或取值路径；给出可验证的结论，不修改生产行为。
- [x] 建立最小复现：HTML CSS `#202123` 在日文验证码正文前时，`extractOTP` 错误返回该色值
- [x] 追踪邮件同步、存储、验证码展示到前端的完整数据流
- [x] 验证根因并记录最小修复建议
- [x] 记录排查结果与执行的检查

## 结果

- 根因不是跨邮箱共享状态：HTML 邮件正文保留后，`extractOTP` 会把样式中的六位色值 `#202123` 当作验证码；日文“認証コード”等未命中上下文关键词时尤其稳定。
- 最小修复建议：在 `extractOTP` 匹配前移除 HTML 的 `style`/`script` 内容和标签，仅对邮件可见文本匹配；该共享函数会同时覆盖同步和读取路径。
- 未修改生产代码；临时复现测试已删除。

## 验证

- `go test ./internal/app -run '^TestExtractOTPProbeSkipsHTMLColorValues$' -count=1`：失败，实际返回 `202123`，期望正文验证码 `719348`。
- 两个公网取码 API 的只读 `peek=1` 检查当前均返回 `no_code`，因此无法以当前收件箱状态完成线上复测。

# HTML 验证码提取修复

- [x] 目标与验收：HTML 邮件中的 `style`/`script` 代码不会被当作验证码；可见的日文、中文和英文验证码仍可正确提取。
- [x] 在共享提取函数前移除 HTML 非可见内容
- [x] 将最小复现固化为回归测试
- [x] 运行格式化、测试、静态检查与构建
- [x] 检查最小差异并提交、推送当前分支

## 结果

- `extractOTP` 先移除 HTML 的 `style`/`script` 内容和标签，再匹配验证码；邮件预览与入库正文保持不变。
- 新增日文 HTML 回归用例，确保 CSS 色值与脚本中的六码不会抢先被返回。

## 验证

- `go test ./... -count=1`
- `go vet ./...`
- `go build -trimpath -o .\bin\icloud-privacy-mail.exe .\cmd\panel`
- `git diff --check` 与所有改动文件 UTF-8 无 BOM 检查

# 邮箱池邮件查看

- [x] 目标与验收：在邮箱池每个账号旁提供“显示邮件”按钮；点击后显示该账号的全部可读取邮件，并保留现有功能。
- [x] 定位邮箱池、账号与邮件读取的现有实现
- [x] 采用现有接口完成最小 UI 与数据串接
- [x] 添加或调整最小回归检查
- [x] 运行格式化、测试与构建验证
- [x] 记录结果与验证方式

## 结果

- 邮箱池每行新增“显示邮件”按钮，弹窗展示所选邮箱的全部已同步邮件（主题、发件人、时间、正文）。
- 复用现有鉴权邮件接口，未新增路由或依赖；正文经 HTML 转义后渲染。
- 约束：当前同步机制只缓存验证码类邮件，完整 iCloud 收件箱不属于本次改动范围。

## 验证

- `go test ./internal/app -run '^TestListMailboxMessagesReturnsAllMailboxMessages$' -count=1`
- `go test ./...`
- `go vet ./...`
- `go build -trimpath -o .\bin\icloud-privacy-mail.exe .\cmd\panel`
- Node 脚本语法编译与 `git diff --check`

# 邮件 HTML 预览修正

- [x] 目标与验收：HTML 邮件按原格式预览，不把样式源码作为正文显示；预览禁止邮件脚本访问页面。
- [x] 构造可复现当前纯文本渲染问题的前端检查
- [x] 确认正文解析和前端渲染边界
- [x] 用最小安全隔离方式渲染 HTML，并保留纯文本兜底
- [x] 添加回归检查并运行格式化、测试、构建
- [x] 记录结果、约束与经验

## 结果

- IMAP 的 multipart 邮件优先保存 HTML 分支，纯文本邮件保持纯文本。
- 邮件弹窗在无脚本、无同源权限的 iframe 中渲染 HTML；CSS、远程图片与字体不会请求网络。
- 历史缓存若未保存 HTML 则按纯文本展示；之后同步的邮件保留 HTML MIME 正文。

## 验证

- `go test ./internal/app -run '^(TestParseICloudIMAPMessagePrefersHTMLPart|TestMailboxHTMLPreviewUsesSandboxedIFrame|TestMailboxMessageViewerDoesNotSynchronize)$' -count=1`
- `go test ./...`
- `go vet ./...`
- `go build -trimpath -o .\bin\icloud-privacy-mail.exe .\cmd\panel`
- Node 前端 HTML 预览回归检查、脚本语法编译与 `git diff --check`

# 邮件查看即时展示

- [x] 目标与验收：点击“显示邮件”只读取本地缓存并立即展示 HTML，不发起 IMAP 同步。
- [x] 固化查看动作不等待同步的回归检查
- [x] 移除查看动作的自动回补调用
- [x] 运行测试、静态检查与构建
- [x] 记录结果与验证方式

## 结果

- 查看邮件只请求本地消息接口，随后在沙箱 iframe 中展示已保存的 HTML 正文。
- 已移除仅由查看动作使用的强制 IMAP 回补分支，避免与后台同步争用锁而持续加载。

## 验证

- `go test ./internal/app -run 'TestMailbox' -count=1`
- `go test ./... -count=1`
- `go vet ./...`
- `go build -trimpath -o .\bin\icloud-privacy-mail.exe .\cmd\panel`
- Node 内联脚本语法检查、UTF-8 无 BOM 检查和 `git diff --check`

# 系统代码健康与缺陷审计

- [x] 重述目标与验收：检查当前系统的可复现缺陷与高风险代码路径；结论需有代码或命令证据
- [x] 定位现有实现、调用链和测试覆盖
- [x] 建立最小反馈环并运行测试、静态检查与构建
- [x] 复核鉴权、存储、并发、外部请求和前端交互边界
- [x] 为确认的问题建立最小复现并给出修复建议（不修改生产行为）
- [x] 运行最终验证并复查差异
- [x] 记录结果、剩余风险与验证故事

## 结果

- 确认高风险缺陷：IMAP 积压截断后游标越过未读取邮件；邮件全部头字段参与别名子串匹配；Apple 2FA pending 未绑定面板用户；公开登录/注册在全局 Store 锁内执行 PBKDF2 且无节流。
- 确认中风险缺陷：Store 保存失败不回滚内存；删除末页数据后邮箱列表停留在空页；严格 JSON 解码接受尾随第二个值；前端登录失败无提示且可重复提交。
- 条件性风险：首次部署的第一个匿名注册者自动成为管理员；高难度 Apple Hashcash 存在随机失败；公开地址/反代协议配置不匹配时前端取码会受 CORS 或混合内容阻止。
- 当前 `state.json` 未发现重复 ID、重复邮箱、孤儿引用或非法邮箱状态；配置端口 `50001` 当前没有监听进程，全局 API Key 未配置。
- 未修改生产代码；4 组临时回归探针均已删除。

## 验证

- `go test ./... -count=1`、`go vet ./...`、`go build ./...`：通过。
- `go test ./internal/app -cover -count=1`：通过，语句覆盖率 59.1%。
- 临时探针各连续运行 3 次，稳定复现 Store 回滚、尾随 JSON、越界分页、跨用户 2FA、收件人错配和重复发码。
- 三个 HTML 模板的内联 JavaScript 语法检查：通过。
- `go test -race` 未运行：当前 `CGO_ENABLED=0`；`staticcheck`、`govulncheck`、`gosec` 未安装。

# 缺陷修复方案文档

- [x] 目标与验收：基于已确认缺陷编写可直接实施的详细修复方案，统一领域术语并记录必要决策
- [x] 确认文档覆盖范围与交付分期：覆盖全部已确认问题，按 P0/P1/P2 分期
- [x] 建立领域词汇表与关键不变量：验证码查询为幂等读取，不记录消费状态
- [x] 为每项缺陷记录根因、修复点、回归测试和验收标准
- [x] 记录跨缺陷的实施顺序、迁移与回滚策略
- [x] 复核源码行号、文档链接、UTF-8 编码和工作树差异
- [x] 记录结果与验证故事

## 结果

- 新增 `docs/系统缺陷修复方案.md`，覆盖全部确认问题，按 P0/P1/P2 描述根因、最小修复、回归测试、验收、迁移、发布和回滚。
- 新增根 `CONTEXT.md`，统一隐私邮箱、有效验证码、验证码查询、邮箱归属、同步游标、面板用户与 Apple 登录流程等术语。
- 新增 7 个 ADR，记录重复取码、管理员初始化、JSON 事务提交、IMAP 连续游标、面板/外部取码边界、用户关闭和更新校验决策。
- 根据用户纠正，验证码查询明确为幂等读取；现有“已返回后跳过”行为列为待移除缺陷，不再把并发重复返回视为问题。

## 验证

- 所有新增文档均可按严格 UTF-8 解码，无 BOM、无替换字符。
- 文档中的源码证据位置、现有测试名称和配置字段已与当前代码复核。
- 相对文档链接、Markdown 结构、`git diff --check` 和工作树范围已检查。

# P0-01 至 P0-07 实施

- [x] 明确 P0-01 至 P0-07 验收标准并定位现有实现
- [x] 为每个缺陷先添加失败回归测试
- [x] 实施 P0-01 事务式 Store 提交
- [x] 实施 P0-02 bootstrap、关闭公开注册和鉴权限流
- [x] 实施 P0-03 Apple 2FA pending 归属与状态保护
- [x] 实施 P0-04 结构化收件人精确匹配
- [x] 实施 P0-05 IMAP 最旧批次与连续游标
- [x] 实施 P0-06 验证码幂等读取
- [x] 实施 P0-07 严格 JSON 请求边界
- [x] 运行 gofmt、go test ./...、go vet ./...、go build ./...、git diff --check
- [x] 复核差异、记录结果与未解决风险

## 验收标准

- P0-01 至 P0-07 的回归测试覆盖文档列出的失败模式，并先红后绿。
- 不引入依赖、不迁移 SQLite、不实现 P1/P2。
- 有效验证码在新鲜窗口内可被所有授权请求重复读取。

## 结果

- Store 写入失败会回滚内存候选状态，使用唯一 `0600` 临时文件完成同步、关闭和原子替换；邮件与游标按同步批次一次提交。
- 新增一次性 bootstrap、管理员用户创建、登录限流和锁外密码校验；公开注册永久返回 `registration_closed`。
- Apple 2FA pending 绑定 owner 并采用 waiting/submitting/completed 状态；持久化失败后重试不再调用 Apple。
- IMAP/WebService 只按 To/Cc/Bcc 精确归属；IMAP 从最旧未检查 UID 开始并按连续完成位置推进。
- 验证码查询不再写消费状态，普通、peek、cache 和并发授权读取均返回当前五分钟内最新验证码。
- JSON 请求拒绝第二个值、尾随垃圾和超过 1 MiB 的请求体。

## 验证

- `gofmt`：完成。
- `go test ./... -count=1`：通过。
- `go vet ./...`：通过。
- `go build ./...`：通过。
- `git diff --check`：通过。
- `go test -race`：未运行；当前 `CGO_ENABLED=0` 且未安装 gcc。

## 未解决风险

- 未执行 race detector；2FA 单提交者与验证码并发行为已由普通并发回归测试覆盖，但仍需在具备 C 工具链的环境补跑 `go test -race ./internal/app -count=1`。
- 按范围未处理 P1 的 UIDVALIDITY、后台任务停止/等待、在线更新校验等问题，也未处理 P2 前端与反代行为。

# P1-01 至 P1-08 实施

- [x] 明确 P1-01 至 P1-08 验收标准并定位现有实现
- [x] 为每个缺陷先添加失败回归测试
- [x] 实施 P1-01 UIDVALIDITY 与世代回看
- [x] 实施 P1-02 登录态定向更新
- [x] 实施 P1-03 IMAP 凭据变更重启 watcher
- [x] 实施 P1-04 owner drain 与后台任务关闭
- [x] 实施 P1-05 在线更新 SHA-256 完整性
- [x] 实施 P1-06 Hashcash Context 取消
- [x] 实施 P1-07 WebSession 过期清理
- [x] 实施 P1-08 并行 IMAP 拨号连接回收
- [x] 运行 gofmt、go test ./...、go vet ./...、go build ./...、git diff --check
- [x] 复核差异、记录结果与未解决风险

## 验收标准

- P1-01 至 P1-08 的回归测试覆盖文档列出的失败模式，并先红后绿。
- 不引入依赖、不迁移 SQLite、不实现 P2。
- 验证码查询保持幂等读取，P1 修改不恢复消费状态。

## 结果

- IMAP 同步持久化 UIDVALIDITY，世代变化或旧状态缺失世代时清空游标并按配置回看；首次 watcher 不再通过 UIDNEXT 跳过历史邮件。
- Apple keepalive 改为按登录态 Kind 定向更新；App 专用密码变更会唤醒 watcher，以密码摘要识别并等待旧 worker 后重连。
- 用户删除采用 owner drain，scheduler、同步、创建和保活受 owner/root context 控制；进程关闭会等待后台任务并记录超时。
- 在线更新仅在目标资产具有合法 SHA-256 时允许应用；GitHub sidecar/统一 checksum、下载摘要常量时间比较及原文件保留均有回归覆盖。
- Hashcash 支持 Context 取消且不再受平均工作量上限限制；启动和会话创建会清理过期 WebSession；并行 IMAP 拨号会关闭所有未采用连接。
- 验证码查询仍为幂等读取；未增加依赖、未迁移 SQLite、未实施 P2。

## 验证

- `gofmt`：完成。
- `go test ./... -count=1`：通过。
- `go vet ./...`：通过。
- `go build ./...`：通过。
- `git diff --check`、严格 UTF-8 无 BOM 检查：通过。

## 未解决风险

- 当前环境 `CGO_ENABLED=0` 且无 gcc，未执行 race detector；owner drain、worker 替换和并行拨号由普通并发回归测试覆盖。
- 进程关闭总超时仍为 ADR 约定的 10 秒，可能短于单 owner drain 的 30 秒；超时会明确记录，但操作系统随后退出时仍可能强制终止未完成外部请求。
- 按范围保留 P2 的前端分页、异步错误展示和部署/反代问题。

# P2-01 至 P2-03 实施

- [x] 明确 P2-01 至 P2-03 验收标准并定位现有实现与测试缝
- [x] P2-01：先添加越界分页失败回归，再实现后端页码钳制与前端状态收敛
- [x] P2-02：先添加前端失败/防重复回归，再修复登录、2FA、scheduler、删除和状态操作
- [x] P2-03：先添加面板同源取码与公开 URL 回归，再修复 Web Session 鉴权和配置校验
- [x] 保持验证码查询为幂等读取，不增加依赖、不迁移 SQLite
- [x] 运行 gofmt、go test ./...、go vet ./...、go build ./...、模板脚本检查和 git diff --check
- [x] 自审差异、记录结果与未解决风险并提交

## 验收标准

- 删除末页唯一记录后，后端与前端都收敛到实际最后一页。
- 指定异步操作失败可见、按钮恢复，快速重复触发只发送一次请求。
- 面板取码只走同源相对路径和 Web Session owner 鉴权；外部 token/API Key 行为保持兼容。
- 未配置 `public_base_url` 时返回相对公开 URL，配置时严格校验 scheme/host 并生成绝对 URL。

## 结果

- 邮箱列表在过滤后计算总页数并钳制请求页；删除末页数据后，响应和前端状态都会收敛到实际最后一页。
- 新旧 Apple 登录、两类 2FA、scheduler 启停、邮箱状态/停用/删除均呈现失败并阻止重复 Promise；邮箱列表重绘期间继续保持变更操作忙状态。
- 面板验证码查询使用 `/api/mailboxes/{id}/code` 同源路径和 Web Session owner 鉴权；外部 token、全局 API Key 与幂等读取行为保持不变。
- 未配置 `public_base_url` 时 API URL 为相对路径；配置时只接受带 host 的绝对 HTTP/HTTPS URL，并覆盖空路径、缺失配置文件和成功读取文件三条启动路径。
- 未增加依赖、未迁移 SQLite。

## 验证

- `gofmt`：完成。
- `go test ./... -count=1`：通过，包含 Node VM 异步行为测试和三个模板的内联脚本语法检查。
- `go vet ./...`：通过。
- `go build ./...`：通过。
- `git diff --check`、严格 UTF-8 无 BOM 检查：通过。

## 未解决风险

- Node VM 覆盖代表性的旧接口登录拒绝/双击路径；其余指定操作复用同一显式模式并通过脚本编译，但未运行真实浏览器点击回归。
- `public_base_url` 校验允许合法 URL 路径前缀；部署者仍需确保反向代理实际暴露该前缀。
- 当前环境未执行 race detector；P2 新增状态仅在浏览器单线程脚本中使用，Go 侧改动没有新增后台并发状态。

# P0-P2 完成后代码审查

- [x] 目标与验收：审查 `6e44df6..HEAD` 的 P0/P1/P2 实现，优先发现可复现缺陷、回归与测试缺口，不修改生产代码
- [x] 复核三个提交的差异、调用链和领域不变量
- [x] 运行完整测试、静态检查、构建和针对性故障探针
- [x] 检查鉴权/持久化、IMAP/后台并发、前端/部署边界
- [x] 按严重度记录发现、源码证据、复现和剩余风险
- [x] 复查工作树与临时探针，记录验证故事

## 结果

- 确认 P2 配置边界缺陷：`public_base_url` 接受 query 和 fragment，随后以字符串拼接 API 路径，生成语义错误的公开取码 URL。
- 确认 P0 资源生命周期缺陷：认证限流器不会清理过期用户名、IP 和操作键，长期运行或大量不同来源失败请求会让内存表无界增长。
- 关机永久驻留候选已否定：Go `http.Server.Shutdown` 在等待连接前会先关闭监听器，即使 context 已过期，`ListenAndServe` 仍可退出。
- 未发现 Store 事务、验证码幂等读取、2FA owner 隔离、IMAP 连续游标、后台 owner drain、更新摘要校验或指定前端异步操作的新回归。
- 未修改生产代码；一次性故障探针均已删除。

## 验证

- `go test ./... -count=1`、乱序 `go test ./internal/app -shuffle=on -count=1`、`go vet ./...`、`go build ./...`：通过。
- P0/P1/P2 针对性回归测试：通过。
- URL 探针稳定接受 `https://public.example/base?x=1` 和 `https://public.example/base#fragment`；限流探针在 100 个过期键后仍保留 101 项。
- `git diff --check 6e44df6..HEAD`：通过；`gofmt -l cmd internal` 列出的 5 个文件均不在 P0-P2 差异中，属于审查基线遗留。
- `go test -race` 未运行：当前 `CGO_ENABLED=0` 且未安装 gcc。

# P0-P2 遗留问题修复实施文档

- [x] 目标与验收：为审查确认的 URL 校验缺口和限流表无界增长编写可在新会话直接实施的完整文档
- [x] 核对现有领域术语、ADR、修复方案和生产调用链
- [x] 明确最小修复、非目标、兼容性和回滚边界
- [x] 定义失败回归测试、实施顺序、验收标准和验证命令
- [x] 复核源码引用、Markdown、UTF-8 无 BOM 和工作树范围
- [x] 记录文档结果与验证故事

## 结果

- 新增 `docs/P0-P2遗留问题修复实施方案.md`，完整覆盖 F-01 `public_base_url` 严格校验和 F-02 认证限流项机会式清理。
- 文档固定了允许/拒绝输入矩阵、最小生产改动、显式时钟与锁内清理规则、红绿测试、实施顺序、兼容性、发布回滚和完成定义。
- 文档附带新会话 `/implement` 提示，可直接交给另一个会话执行。
- 未修改 `CONTEXT.md`：没有新增领域术语；未新增 ADR：两项局部修复可逆且没有架构级取舍。
- 未修改生产代码。

## 验证

- 文档 430 行、30 个标题、14 组闭合代码围栏；引用的 6 个源码/领域文档路径均存在。
- `git diff --check`：通过。
- 新文档、`tasks/todo.md`、`CONTEXT.md` 和关联 ADR 均为严格 UTF-8、无 BOM、无替换字符。

# P0-P2 遗留修复完成后复审

- [x] 目标与验收：审查当前未提交的 F-01/F-02 修复及其对 P0-P2 核心语义的影响；只报告可复现问题，不修改生产代码
- [x] 复核 `config.go`、`server.go` 与新增回归测试差异
- [x] 运行定向测试、全量测试、乱序测试、静态检查和构建
- [x] 检查 URL 边界、限流生命周期、并发与时钟边界
- [x] 清理临时探针并记录发现、验证和剩余风险

## 结果

- 未发现新的可复现缺陷；F-01、F-02 实现符合 `docs/P0-P2遗留问题修复实施方案.md`。
- `public_base_url` 统一拒绝用户信息、query、空 query 分隔符、fragment 和空 fragment 分隔符，保留合法路径前缀与大小写无关 HTTP(S) scheme。
- 认证限流项具有明确到期时间并在锁内按分钟机会式清理；活跃封禁、登录失败阈值、操作阈值和时钟回拨行为均有回归覆盖。
- 验证码查询仍为幂等读取；P0-P2 其他关键调用链没有因本轮改动发生变化。
- 未修改生产代码，也未创建临时探针。

## 验证

- F-01/F-02 及现有限流定向测试：通过。
- `go test ./... -count=1`：通过。
- `go test ./internal/app -shuffle=on -count=1`：通过。
- `go vet ./...`、`go build ./...`、四个改动文件 `gofmt -l`、`git diff --check`：通过。
- `go test -race` 未运行：当前 `CGO_ENABLED=0` 且没有 gcc。
- 生产与回归测试改动仍未提交：`config.go`、`server.go`、`p0_regression_test.go`、`p2_regression_test.go`。

# P0-P2 遗留修复 F-01/F-02

- [x] F-01：添加 `public_base_url` 非法/合法输入回归并确认红灯
- [x] F-01：严格拒绝用户信息、query、fragment，保留合法路径前缀
- [x] F-02：添加认证限流项生命周期回归并确认红灯
- [x] F-02：实现锁内机会式过期清理且保持现有阈值
- [x] 运行 gofmt、定向/全量测试、乱序测试、go vet、go build 和差异检查
- [x] 记录结果、验证与未解决风险

## 验收标准

- `public_base_url` 仅接受无用户信息、query、fragment 的绝对 HTTP(S) URL 或空值。
- 过期认证限流项会被后续认证操作回收，活跃封禁和既有阈值不变。
- 验证码查询继续幂等读取；不新增依赖、后台 goroutine、配置项或 SQLite 迁移。

## 结果

- `public_base_url` 在统一加载出口拒绝用户信息、query 和 fragment（含空分隔符），继续接受空值、合法路径前缀、编码路径字符和大小写不敏感的 HTTP(S) scheme。
- 认证限流项记录明确到期时间，四个现有入口在同一 mutex 内最多每分钟机会式清理一次；时钟回退仍触发清理，活跃封禁和原阈值不变。
- 验证码查询代码、面板同源取码、外部 token/API Key、SQLite 和依赖均未修改。

## 验证

- 新增 F-01 回归先稳定失败 5 个非法 URL 用例；修复后定向测试通过。
- 新增 F-02 回归先稳定失败过期键清理与时钟回退用例；修复后 `TestAuthRateLimiter*` 全部通过。
- `gofmt`、`go test ./... -count=1`、`go test ./internal/app -shuffle=on -count=1`、`go vet ./...`、`go build ./...`：通过。
- `git diff --check`、严格 UTF-8 无 BOM 检查：通过。

## 未解决风险

- 当前 `CGO_ENABLED=0` 且未安装 gcc，未运行 `go test -race ./internal/app -count=1`；限流状态始终在既有 mutex 内访问，并由普通回归测试覆盖。
- 机会式清理只解决过期项永久驻留；一分钟内的高基数来源洪泛仍需由反向代理或边缘网关限流。

# 前端设计系统重构（refactor/frontend-design-system）

依据：`docs/design-system/DESIGN.md`（由 Open Design 产出，已核对诊断准确）
范围：`internal/app/templates/` 下 index.html(3356) / manage.html(884) / login.html(100)

## 前置约束（已勘查确认）

- 模板为**纯静态 HTML**，`{{ }}` 出现 0 次；数据全由 JS 调 API 后 `innerHTML` 渲染
- `writeTemplate` 仅做 `webFS.ReadFile` + 原样 `Write`，无模板解析 → 改 HTML 不影响 Go 侧
- JS 的 DOM 契约面（**重构中必须逐一保持**）：
  - index.html：`$('id')` 封装引用 56 个唯一 id；页面 `id=` 属性 70 个；`innerHTML` 渲染点 10 处
  - `querySelectorAll` 依赖的选择器：`.mailbox-message-html`、`.manage-refresh-clock`、
    `.nav-item[data-view]`、`.view-section[data-view]`、`[data-density-option]`、`[data-log-category]`
  - manage.html：`.nav button[data-view]`、`.view-section[data-view]`、`.account-card`
  - `classList` 操作 14 处（active / hidden 等状态类不得改名）

## 分阶段任务

- [x] 阶段1 令牌层与主题（DESIGN.md 清单 1-2）
      10 套主题 → light/dark；`--green*` → `--accent*`；清 manage.html 硬编码色值；保留 data-density
- [x] 阶段2 去装饰与归一（清单 3-7）
      移除 27 处 gradient、8 处 backdrop-filter、假高光、.card::before 彩条、大阴影；
      41 处 font-weight 700-950 → 400/500/600；删 5 组入场动画与 10 处 translateY；
      圆角 → {4,6,8,999}；补全 focus-visible（当前 0 覆盖）
- [x] 阶段3 表格与表单（清单 8-9）
      修粘性表头（容器缺高度约束致失效）；行悬浮改中性色；数字列 tabular-nums 右对齐；
      长文本改省略号+title；≤820px 转卡片列表；补空态与骨架屏；
      表单标签 13px/500、必填星号、aria-invalid、.field-error
- [x] 阶段4 信息架构（清单 10-11）**已获用户确认执行**
      顶栏 9 元素 → 「刷新状态」+ 用户下拉；登录态 6 按钮 → 两张分组卡；
      「一键删除码」隔离危险区 + 二次确认；删 4 张 quick-card；日志面板改可折叠

## 验收标准（每阶段后校验）

- `rtk go build ./...` 通过；`rtk go test ./internal/... -short` 保持 232 passed
- 静态校验：gradient=0、backdrop-filter≤1(顶栏)、font-weight>600 计数=0、
  圆角取值 ⊆ {4,6,8,999}px、translateY 悬浮=0
- DOM 契约校验：56 个 `$('id')` 引用全部命中页面实存 id；6 个 querySelectorAll 选择器仍有匹配
- 键盘可达：登录 → 保存登录态 → 创建邮箱 → 邮箱池取码复制，全程焦点可见
- 断点 360/390/768/1024/1440/1920px 无横向滚动

## Results

### 阶段1 令牌层与主题（commit 见 refactor/frontend-design-system）
- `index.html` / `manage.html` / `login.html` 三份模板统一换成 oklch 语义令牌，
  主题从 10 套收敛为 `light` / `dark`，旧主题名经 `LEGACY_THEME_MAP` 归类迁移，
  无存储值时跟随 `prefers-color-scheme`。
- 保留兼容别名层（`--panel`、`--line`、`--text`、`--green` 等映射到语义令牌），
  未改写的组件规则仍可工作。
- `login.html` 整体重写：单张 `min(400px,100%)` 居中卡、下划线式标签页、唯一实心按钮。

### 阶段2 去装饰与归一
- 清掉 27 处 gradient、假高光、`.card::before` 彩条、大阴影与 10 处 translateY 悬浮，
  字重全部收敛到 400/500/600，圆角收敛到 {4,6,8,999}px，补齐 `:focus-visible` 焦点环。
- 顺带修掉两个既有缺陷：粘性表头因容器无高度约束而失效；`$('serviceInfo')` 为不可达死代码。

### 阶段3 表格与表单
- 表格：`.table-wrap` 补 `max-height` 让粘性表头真正生效（manage.html 同样缺失，已一并修）；
  长文本（邮箱 22ch / ID 18ch）改省略号 + `title`；数字列与时间列用 `tabular-nums`，
  数字列右对齐且表头同步加 `.num`。
- 空态：`index.html` 邮箱池区分「筛选无结果」（给清空搜索）与「尚无数据」（给去创建），
  manage.html 四处空态补标题 + 说明文案，最小高度 160px 防面板塌陷；另加 `.skeleton-line` 骨架样式。
- 响应式：≤820px 表格转卡片列表，字段名由 `td::before { content: attr(data-label) }` 生成，
  因此表格 DOM 与渲染 JS 结构无需改动；4 个渲染器共 17 个单元格补上 `data-label`。
- 表单：必填字段加 `.req` 星号与 `required`；新增 `.field-error`（预留 18px 行高，报错不跳动）；
  登录页接入字段级校验 `setFieldError`，同步 `aria-invalid` 并聚焦首个非法字段，
  注册模式额外校验密码长度 ≥ 6；补上 `login.html` 缺失的 `--danger-glow` 令牌。

### 验证情况
- `bash tasks/check-design.sh`：全部通过（DOM 契约 + 8 项 DESIGN.md 验收，focus-visible 19 处）
- `rtk go build ./...`：Success
- `rtk go test ./internal/... -short -count=1`：232 passed
- 起服务实测 8801 端口：`/` `/login` `/manage` 均 200；三份模板 CSS 花括号配平；var() 引用无未定义令牌

### 阶段4 信息架构
- 顶栏：可见交互元素从 9 个降到 2 个（`刷新状态` + 用户菜单触发器），
  主题/密度/运行配置/账号数据/版本与更新/退出登录全部收进 `#userMenuPanel` 折叠面板。
  菜单支持点击外部关闭、Esc 关闭并回焦触发器，`aria-expanded` 与 `hidden` 同步。
- 登录态视图：原 6 按钮平铺工具栏拆成新接口/旧接口两张 `.login-group` 卡，
  每张卡只有一个 primary（对应「登录」），提交验证码归到各自接口下；
  保存配置与检测登录态移到共用次级工具栏。
- 危险操作：`一键删除码` 从运行配置工具栏移入独立 `.danger-zone`，
  补第二道确认（需手动输入「删除」，不匹配则取消并记日志）。
- 概览：删掉 4 张 quick-card（其中 3 张与侧栏导航重复），同时清理其全部死 CSS。
- 日志面板：改为可折叠，默认收起，展开状态存 `ipm_log_open`；
  标题行整行可点击作开关，箭头随 `aria-expanded` 旋转；原面板内重复的「刷新状态」按钮删除。

### 阶段4 验证情况
- `bash tasks/check-design.sh`：全部检查通过（focus-visible 18 处）
- `rtk go build ./...`：Success
- `rtk go test ./internal/... -short -count=1`：232 passed
- 起服务实测 8803 端口：`/` `/login` `/manage` 均 200
- 结构核查：三份模板花括号配平；16 个新增类名均同时存在 CSS 规则与标记使用；
  已删除的 `quick-card`/`quick-grid`/`account-dock`/`appearance-controls`/`version-dock`/`theme-select`
  在全文已无残留引用

### 浏览器实测（补齐 DESIGN.md §12 剩余两项验收）

用 Playwright 驱动真实 Chromium 完成，此前只能静态核查的两项现已确认。
测试实例用独立数据目录，通过 `IPM_BOOTSTRAP_TOKEN` 初始化管理员后登录，避免污染开发数据。

- 断点无横向滚动：`/login` `/` `/manage` × 360/390/768/1024/1440/1920px 共 18 组，
  `scrollWidth - clientWidth` 全部为 0。
  注意首轮测试踩坑：未登录时 `/` 与 `/manage` 会被前端重定向到 `/login`，
  必须先登录，否则测的其实是登录页。
- 键盘可达性：Tab 遍历出 11 个停靠点，可见元素无焦点环者 0 个，
  无停靠点落入 `[hidden]` 容器（即收起的菜单和日志面板不会截留焦点）。
- 用户菜单：打开 `aria-expanded=true` / `hidden=false`，
  Esc 后回到 `false` / `true` 且焦点归还触发器。
- 顶栏可见交互元素实测为 2 个（刷新状态、userMenuTrigger），与设计目标一致。
- 日志面板：默认收起，点击展开并写入 `ipm_log_open=1`，刷新后保持展开。
- 主题令牌随 `data-theme` 切换：`--bg` light `oklch(99% 0.002 240)` → dark `oklch(16% 0.008 255)`。
- 表格响应式：390px 下 `thead` 为 `display:none`，行变 `grid`，
  `td::before` 实际渲染出字段名（取到 `"ID"`），确认卡片列表生效且 JS 无需改动。
- 粘性表头：1440px 下 `.table-wrap` `max-height` 计算为 580px、`overflow-y:auto`，
  `th` 为 `position:sticky; top:0`，高度约束到位所以不会静默失效。
- 控制台错误 0 条。
