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
