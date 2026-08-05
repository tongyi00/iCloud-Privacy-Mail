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
