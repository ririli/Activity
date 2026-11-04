# Activity

活动后端项目，采用 DDD 分层与微服务架构。目前仅建立目录骨架，尚未初始化 Go 模块、引入依赖或编写业务代码。

## 目录结构

```text
Activity/
├── services/
│   ├── identity-service/       # 账号、认证、会话
│   ├── activity-service/       # 活动参与、抽奖、资产与奖励
│   ├── ranking-service/        # 积分榜单、最终排名、排名奖励清单
│   └── notification-service/   # 实时中奖通知
├── contracts/
│   ├── http/                   # 对外 HTTP 接口契约
│   ├── rpc/                    # 服务间同步接口契约
│   └── events/                 # 跨服务集成事件契约
├── pkg/platform/               # 可复用的技术基础能力，不存放领域模型
├── deploy/local/               # 本地依赖与服务部署资源
├── scripts/                    # 开发、构建与运维辅助脚本
├── tests/e2e/                  # 跨服务端到端测试
├── docs/                       # 需求与开发规划
└── web/                        # 已有 H5 资源
```

## 服务内部结构

每个服务使用以下结构，目录中的 `.gitkeep` 仅用于保留空目录：

```text
<service>/
├── cmd/server/                # 服务启动与依赖组装
├── configs/                   # 配置模板，不存放真实密钥
├── migrations/                # 本服务拥有的数据库迁移
├── internal/
│   └── <context>/
│       ├── domain/            # 聚合、值对象、领域服务、仓储接口
│       ├── application/       # 用例编排、事务、幂等与应用接口
│       ├── infrastructure/    # 仓储实现、缓存、消息与外部适配
│       └── interfaces/        # HTTP/RPC 处理器、消息消费者等入口
└── test/integration/          # 数据库、消息等集成测试
```

各服务预建的上下文：

|服务|上下文目录|职责|
|---|---|---|
|identity-service|identity|账号、认证与会话|
|activity-service|participation|活动期次与规则、用户轮回、花朵、阶段和宝箱进度|
|activity-service|lottery|正式抽奖、先抽后付预览与兑现|
|activity-service|assets|钱包、购买兑换、奖励记录及选择权益|
|ranking-service|ranking|积分投影、排名查询与最终排名结算|
|notification-service|notification|中奖消息映射与实时推送|

活动核心服务额外提供 `seeds/`，预留活动规则和奖池初始化数据。各模块按职责归属实现，跨模块事务由应用层编排；钱包、抽奖结果和参与进度初期共用本服务的本地事务。

## 开发约定

- 领域层不依赖数据库、HTTP 或消息队列框架。
- 服务之间通过契约通信，不直接访问其他服务的数据库。
- 公共目录仅共享技术能力和通信契约，不共享聚合、领域实体或数据库模型。
- 后续建项时再确定 Go 模块路径、工作区配置、框架与消息队列，不在目录准备阶段固定这些选择。
- 本阶段未修改 `web/`，也未创建可运行服务或部署配置。

详细业务范围与开发顺序见 [项目开发规划](docs/Activity项目开发规划（DDD微服务）.md)，业务规则以 [需求文档](docs/Lottery业务需求文档（不含后台）.md) 为准。当前持久层设计见 [持久层设计方案 V2](docs/持久层设计方案V2.md)，包含活动核心库记录表的同库4分表方案。

## 数据库脚本

账号服务技术选型、双 JWT、会话撤销、负缓存、IP 注册限流与实施步骤见 [identity-service 实现方案](docs/identity-service实现方案.md)。该方案为待实施设计，尚未编写认证业务代码。

已提供 MySQL 8.4 初始建库建表脚本，共3个逻辑库、42张物理表；执行顺序、字段约定与检查方式见 [数据库脚本使用说明](docs/数据库脚本使用说明.md)。脚本尚未连接数据库执行，不包含业务种子数据。

## 后端实现进度（2026-11-04）

已新增四个可编译服务与容器配置；此前的目录骨架说明已过时。启动方式、API和未完成项以 [后端实现与验收](docs/后端实现与验收.md) 为准。当前仍有未完成设计项，不能视为完整生产交付。
