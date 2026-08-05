# Lessons

## 2026-08-05 — HTML 邮件预览

- 失败模式：将邮件 HTML 在入库前去标签，再按纯文本渲染，导致 CSS 源码直接展示。
- 检测信号：包含 `<style>` 的 multipart/alternative 邮件无法生成沙箱 iframe 预览。
- 预防规则：需要显示邮件正文时保留优先的 HTML MIME 分支，并仅在无脚本、无同源权限的 iframe 中渲染。
