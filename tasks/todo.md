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
