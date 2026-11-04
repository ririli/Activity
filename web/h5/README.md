# H5 前端

从 Lottery/web/h5 迁移，保留源码、静态资源、设计参考、测试和依赖锁文件。

## 本地启动

在本目录打开 PowerShell，执行：

```powershell
npm ci
npm run dev
```

- `npm ci`：根据 package-lock.json 安装锁定版本的依赖，生成 node_modules。
- `npm run dev`：启动开发服务器，默认端口5173。
- `npm run build`：类型检查并生成 dist 构建产物。
- `npm test`：运行现有前端测试。

本次未迁移 node_modules、dist 和 test-results，也未重新安装依赖。
接口路径仍为 /api/v1，开发代理在 vite.config.ts 中，当前指向 http://localhost:80。
新后端需要兼容 src/api 中现有调用，或联调时调整接口与代理配置。
移动源码不代表已经接通 Activity 后端。本次仅验证迁移前后文件内容一致，未运行构建与测试。
