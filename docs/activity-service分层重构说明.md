# activity-service 服务级编排与 DDD 分层

日期：2026-11-05。

本次重构按“完整业务用例由上层编排，内部模块公开业务能力，共用一个本地事务”实施。四个微服务、业务玩法、HTTP 路径、JSON 字段、事件契约及数据库表结构保持兼容，不新增中间件。

## 代码入口与依赖

```text
services/activity-service/internal/
├─ interfaces/
│  ├─ http/                     # Gin、请求绑定和响应错误映射
│  ├─ messaging/                # 集成事件解码、永久失败隔离适配
│  └─ worker/                   # 定时触发与生命周期
├─ application/                 # 完整业务用例、事务/锁/幂等/结算编排
├─ assets/
│  ├─ domain/                   # 钱包、商品费用、初始化策略、奖励选款
│  ├─ application/              # 资产变更、奖励与选择能力、仓储端口
│  └─ infrastructure/mysql/     # 钱包、分类流水、奖励、购买持久化
├─ lottery/
│  ├─ domain/                   # 权重、点亮演算、固定计划、预览状态
│  ├─ application/              # 演算/幂等恢复能力及仓储端口
│  └─ infrastructure/mysql/     # 批次、逐抽、点亮流水、预览、幂等
├─ participation/
│  ├─ domain/                   # 期次、轮进度、阶段、宝箱状态约束
│  ├─ application/              # 进度能力及仓储端口
│  └─ infrastructure/mysql/     # 门禁、参与、轮次、阶段、宝箱
├─ infrastructure/
│  ├─ mysql/                    # 事务装配、Outbox、查询投影、任务租约
│  ├─ sqlkit/                   # 仅基础设施使用的 SQL/批量辅助
│  ├─ redis/                    # 用户操作锁
│  ├─ ranking/                  # 榜单结算客户端
│  └─ effects/                  # 提交后有界通知队列和发布
└─ fault/                       # 不携带 HTTP 状态码的业务错误
```

依赖方向：`interfaces → 顶层 application → 上下文 application/domain`；基础设施实现应用层声明的端口，`cmd/server` 完成组装。上下文应用包不互相调用，不引用顶层 application，也不导入 Gin、SQL、Redis、MQ 或 `pkg/platform`。组合规则 JSON 是顶层应用的数据类型，数据库格式不变。

- [完整抽奖编排](../services/activity-service/internal/application/draw.go)：资产扣费/发奖、抽奖演算和参与进度在这里组合。
- [事务会话端口](../services/activity-service/internal/application/ports.go)：只组合仓储能力及 Outbox；`Scope` 只包含用户、期次与业务 ID。
- [MySQL 事务装配](../services/activity-service/internal/infrastructure/mysql/uow.go)：一个连接、一个事务，绑定三个上下文仓储及 Outbox。
- [统一 HTTP 入口](../services/activity-service/internal/interfaces/http/routes.go)：按用例接口调用，不依赖具体 Service；绑定标签只在请求 DTO 中。

原 `appstate.State` 已删除。顶层私有 `work` 仅保存一次用例所需的独立领域对象、能力引用和提交后效果，不包含 SQL，不作为模块间的公共输入。各模块接受本模块的类型；抽奖奖品转换为资产奖励时显式复制候选，避免选款更改原始抽奖快照。

## 抽奖与事务

```text
HTTP / 消息 / 后台入口
  → 顶层用例校验、读取幂等结果、获取用户锁
  → UnitOfWork.Within
      → 期次共享门禁 → 钱包 → 参与 → 当前轮 → 具体单据
      → 检查申请抽数所需的全额花瓣
      → lottery.Generate 生成固定计划
      → assets.Apply 分类扣费、中奖与退款
      → assets.SaveGrants 保存每抽奖励
      → participation.ApplyDraw 更新轮进度和积分
      → lottery.SaveDraw 保存批次、逐抽、独立点亮和幂等
      → Outbox.Append
  → 统一提交
  → 入队允许来源的实时通知
```

模块仓储不持有提交权限。基础设施内部的 `sqlkit.Tx` 不提供 Begin/Commit/Rollback，也不进入业务包。`BindRound` 只绑定后续仓储方法的轮号和预览起点版本，不传递公共业务快照。

榜单服务通过 Outbox 投递的正式抽奖事件更新积分和 Redis 排名。缓存写入与恢复见 [榜单 Redis 缓存与故障恢复](榜单Redis增量实现.md)。

所有 SQL 错误向上传播，业务失败回滚整笔事务。提交返回错误仍按结果未知处理，禁止发提交后通知；普通抽奖允许以同一操作标识恢复。MySQL 行锁、条件更新、自然唯一键继续作为最终保护，Redis 锁保持快速冲突拒绝策略。

分表范围仍为 draw_order、draw_detail、flower_lit_log、wallet_ledger 四类，固定按 user_id % 4 同库路由。reward_grant 不分表。快照、每抽奖励和独立点亮事实均保留；最多 100 行一批的 SQL 拆块不拆事务。查询投影保留在服务级 MySQL 查询适配器，允许组合本服务数据，不为历史列表强行构造领域聚合。

## 接口与行为兼容

- 顶层提供 DrawCommand、PurchaseCommand、SelectRewardCommand 等明确输入，原 Action 命令已删除。
- 抽奖返回明确 DrawResult；既有列表和部分操作响应继续保持原 JSON 投影形状，避免前端升级。
- 业务错误在 HTTP 层映射状态码；消息适配器将永久业务错误映射为现有消费者能识别的隔离错误，数据库或网络失败仍可重试。
- 账号初始化和排名奖励事件解码位于 messaging；账号余额初始值不变，Inbox、资产流水与初始化共同提交。
- 用户选款现在传递已认证昵称，修正旧入口缺失昵称而抑制中奖通知的问题；重复选款不重复通知，阶段和宝箱仍不播报。
- Bootstrap 由顶层应用编排，查询适配器只返回待初始化的用户/期次组合，不持有应用回调。

## 验证与边界

PowerShell 中使用项目指定 SDK：

```powershell
& 'D:\Program Files\GO\go1.25.1\bin\go.exe' test ./... ./services/identity-service/... ./services/activity-service/... ./services/ranking-service/... ./services/notification-service/...
& 'D:\Program Files\GO\go1.25.1\bin\go.exe' vet ./services/activity-service/...
```

新增测试覆盖：

- 扣费、中奖入账、奖励、进度、抽奖事实和 Outbox 任一步失败的整体回滚及通知抑制。
- 正式抽奖重放、异参数冲突、24 小时过期拒绝、提交结果未知后的同标识恢复。
- 满轮停止及分类退款、申请全额余额检查、预览按固定结果兑现和补差共同回滚。
- 180 抽时的批量边界、奖励/逐抽行数、点亮事实、四分表路由、同一个事务连接，以及每条 SQL 写入失败时回滚。
- 轮次/阶段/宝箱/选款/预览领域状态约束，以及 HTTP、消息错误契约与依赖边界。

本轮测试和 vet 通过。SQL 录制驱动与内存仓储只验证编排和事务装配，不能代替真实 MySQL 的 SQL 执行与锁竞争验证。Docker 引擎未运行，真实依赖联调和端到端用例尚未重跑；未执行建库、建表、迁移或修改部署数据。
