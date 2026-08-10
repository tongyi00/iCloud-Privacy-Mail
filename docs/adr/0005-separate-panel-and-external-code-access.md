# 分离面板取码与外部验证码 API

面板内取码始终调用同源接口，并使用 Web Session 与邮箱归属授权；`public_base_url` 和邮箱独立 token 只用于生成外部验证码 API 地址。该边界避免面板受 CORS、HTTPS 混合内容和反代 Host 影响，也避免为了内部交互扩大公开 token URL 的浏览器使用范围。
