# H5 前端

Vue 3 + Vite + TypeScript H5，已迁移到 Activity 新后端契约。

## 本地启动

```powershell
npm ci
npm run dev     # http://localhost:5173
npm test
npm run build   # 输出 dist/
```

开发服务器把 `/api` 代理到 `http://localhost:8080`（Compose 的 Nginx 入口），并把代理请求的 `Origin` 改写为 `http://localhost:8080`，以匹配 identity 的 `ALLOWED_ORIGIN` 和 CSRF 校验。若直接访问 Nginx 的 8080 同源入口，不需要 Vite。

## 认证与接口

- 登录/注册使用 `account, password, nickname?`；Access 只保存在内存，页面刷新后通过 HttpOnly Refresh Cookie 调用 `/api/identity/v1/auth/refresh` 恢复。
- Refresh/Logout 由浏览器自动发送 Cookie，并带 `X-Activity-CSRF: 1`；前端不再读写 `refresh_token`。旧的 `lottery_access_token`/`lottery_refresh_token` 会在启动时清理。
- 所有业务 ID 以十进制字符串处理；只有已知安全的小型道具 ID 在 UI 目录查表时转成数字。
- 活动/榜单接口由 `src/api/activitySession.ts` 自动解析当前 `edition_id`；活动期首次进入会调用幂等的 `join` 初始化参与状态。
- 记录接口使用 `before` 游标（每页 20 条），适配层在 `src/api/records.ts` 中转换为现有页面需要的分页结构。
- SSE 使用 `/api/notification/v1/barrage/stream`，处理 `ready`、`barrage`、`auth_required`、`unavailable` 控制事件；Access 过期时刷新后重连。

## Nginx / Compose

`compose.services.yaml` 的 proxy 挂载 `./web/h5/dist:/usr/share/nginx/html:ro` 并启用 SPA `try_files` 回退。**启动 Compose 前先执行 `npm run build`**，否则 Nginx 没有静态资源。

```powershell
npm run build
docker compose -f compose.yaml -f compose.services.yaml up -d --build
```

生产环境必须使用 HTTPS、设置准确的 `ALLOWED_ORIGIN`，并保持 H5 与 API 同源。
