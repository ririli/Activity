# Activity

活动后端项目，采用 DDD 分层与微服务架构，包含账号、活动核心、榜单和实时通知四个 Go 服务。

## 目录结构

```text
Activity/
├── services/
│   ├── identity-service/       # 账号、认证、会话
│   ├── activity-service/       # 活动参与、抽奖、资产与奖励
│   ├── ranking-service/        # 积分榜单、最终排名、排名奖励清单
│   └── notification-service/   # 实时中奖通知
├── contracts/rpc/              # 服务间 RPC 契约与生成代码
├── pkg/platform/               # 可复用的技术基础能力，不存放领域模型
├── deploy/local/               # 本地依赖与服务部署资源
├── scripts/                    # 开发、构建与运维辅助脚本
├── tests/e2e/                  # 跨服务端到端测试
├── docs/                       # 需求与开发规划
└── web/                        # 已有 H5 资源
```

## 服务内部结构

服务按实际职责组织目录；仅有占位文件的空骨架已清理，不保留未使用的目录：

```text
<service>/
├── cmd/server/                # 服务启动与依赖组装
├── migrations/                # 有数据库迁移的服务保留此目录
└── internal/
    └── <context>/
        ├── domain/            # 聚合、值对象、领域服务
        ├── application/       # 用例、业务能力与仓储端口
        ├── infrastructure/    # 仓储实现、缓存、消息与外部适配
        └── interfaces/        # HTTP/RPC 处理器、消息消费者等入口
```

各服务的上下文：

|服务|上下文目录|职责|
|---|---|---|
|identity-service|identity|账号、认证与会话|
|activity-service|participation|活动期次与规则、用户轮回、花朵、阶段和宝箱进度|
|activity-service|lottery|正式抽奖、先抽后付预览与兑现|
|activity-service|assets|钱包、购买兑换、奖励记录及选择权益|
|ranking-service|ranking|积分投影、排名查询与最终排名结算|
|notification-service|notification|中奖消息映射与实时推送|

活动核心服务通过 `cmd/seed` 提供活动规则和奖池初始化入口。各模块按职责归属实现，跨模块事务由应用层编排；钱包、抽奖结果和参与进度共用本服务的本地事务。

活动服务当前统一使用服务级 `internal/application` 和 `internal/interfaces`，三个内部上下文提供领域与应用能力；具体代码结构见 [activity-service 分层重构说明](docs/activity-service分层重构说明.md)。

## 开发约定

- 领域层不依赖数据库、HTTP 或消息队列框架。
- 服务之间通过契约通信，不直接访问其他服务的数据库。
- 公共目录仅共享技术能力和通信契约，不共享聚合、领域实体或数据库模型。
- 使用仓库现有的 Go 模块、工作区及容器配置；SDK 版本与项目 `go.mod` 保持兼容。

详细业务范围与开发顺序见 [项目开发规划](docs/Activity项目开发规划（DDD微服务）.md)，业务规则以 [需求文档](docs/Lottery业务需求文档（不含后台）.md) 为准。当前持久层设计见 [持久层设计方案 V2](docs/持久层设计方案V2.md)，包含活动核心库记录表的同库4分表方案。

## 数据库脚本

账号服务技术选型、双 JWT、会话撤销、负缓存、IP 注册限流与实施步骤见 [identity-service 实现方案](docs/identity-service实现方案.md)。该方案为待实施设计，尚未编写认证业务代码。

已提供 MySQL 8.4 初始建库建表脚本，共3个逻辑库、42张物理表；执行顺序、字段约定与检查方式见 [数据库脚本使用说明](docs/数据库脚本使用说明.md)。脚本尚未连接数据库执行，不包含业务种子数据。

## 后端实现进度（2026-11-04）

已新增四个可编译服务与容器配置；H5 已迁移到新 API（内存 Access、HttpOnly Refresh Cookie、字符串 ID、edition 自动解析、SSE 控制事件），Nginx 服务 `web/h5/dist` 并支持 SPA 回退。活动核心已完成三上下文应用层拆分，购买/抽奖/预览/阶段/宝箱/结算编排已迁出 MySQL 适配器。启动方式、API 与剩余验收项以 [后端实现与验收](docs/后端实现与验收.md) 为准；压力测试、故障注入和生产可观测性仍需完成。

入口保护与 Redis 用户令牌桶的阈值、环境变量、故障策略和验证步骤见 [入口与业务限流](docs/入口与业务限流.md)。
