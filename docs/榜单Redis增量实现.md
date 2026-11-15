# 榜单 Redis 先写与故障恢复

本页说明榜单服务的消息消费、Redis 查询与缓存恢复。实现入口为 [缓存维护与消费](../services/ranking-service/internal/ranking/infrastructure/mysql/cache.go)。

## 四片部署

- K3s 默认部署四组榜单 Redis，每组一主一副。三个 Sentinel 同时监控 `ranking-shard-0` 至 `ranking-shard-3`，每组需要两个 Sentinel 达成仲裁。榜单服务使用 `RANKING_REDIS_SENTINEL_ADDRS` 指定三个 Sentinel 地址，通过 `RANKING_REDIS_SENTINEL_PASSWORD` 和 `RANKING_REDIS_PASSWORD` 分别认证。每片连接池只连接该组当前主节点。 [K3s 部署](../deploy/k8s/README.md)
- Compose 按需启用相同拓扑。也可通过 `RANKING_REDIS_SHARD_ADDRS` 指定四个固定的数据节点地址，与分片 `0` 至 `3` 顺序一致，认证使用 `REDIS_PASSWORD`。两种地址配置互斥。两种地址均为空时，用 `REDIS_ADDR` 在共享 Redis 上划分四个逻辑分片。 [本地部署](../deploy/local/README.md)
- 新注册用户 ID 的低两位依次轮换 `0` 至 `3`，避免每毫秒只注册一人的场景中分片长期集中。线上容量按实际入榜人数和抽奖量的最热分片计算。

## 消费与查询

- 活动服务在正式抽奖事务内保存事实和 Outbox。榜单服务通过已有队列消费，正常情况下先更新 Redis，再提交榜单 MySQL。MySQL 提交后确认消息。
- Redis 每期按 `user_id % 4` 划分实时榜。每片使用 ZSET 排名、Hash 保存积分和达分秒、Set 保存已处理的 drawID。Lua 合并积分时取最大业务秒。重复 drawID 跳过缓存更新，仍尝试提交 MySQL。MySQL 用 `score_fact.draw_id` 主键去重并验证事实，按连续抽奖序号推进。Redis 可先显示已到事件的合计。
- 实时榜从四片归并 Top 20，并用所在片的排名与其他三片排在该用户前的人数计算本人名次。期次缓存异常时整份查询回退 MySQL，单实例最多允许 8 个并发降级查询，查询超时为 2 秒。活动结束榜读取 MySQL 的 `ranking_snapshot`。整点用独立 Redis 窗口，见 [整点榜单结算](整点榜单结算.md)。

## 故障与重建

- Redis 写入失败时，消费者继续保存 MySQL，并暂停读取该期缓存。Redis 写成功、MySQL 临时失败时，未确认事件会重投。同一代缓存内的 Set 防止重复加分。
- Redis 脚本检查节点主从角色、进程 `run_id`、代次标识和各集合计数。Sentinel 提升副本后，新主的 `run_id` 使复制过来的旧代次失效。服务从 MySQL 的一致性快照读取 `score_fact` 和零分参与记录，验证四片的新代次后重新开放 Redis 查询。重建期间暂缓该期的榜单消息处理。所有已落库事实均进入积分和去重 Set，包括跨序号缺口的事实。
- 榜单服务按单活部署。Kubernetes 使用 Recreate，Compose 保持一个 ranking 实例。重建时整期缓存先下线，继续对用户提供 MySQL 降级。结算核对独立写 MySQL，最终快照和奖励清单生成后整组退役 Redis 键。
- 保持 `noeviction`。学习配置单个数据进程的 `maxmemory` 为 96 MB，容器上限 256 MiB。容量评估需计入榜单、用户 Hash、随抽奖批次增长的去重 Set、整点窗口，以及复制和 AOF 重写的内存峰值。用实际峰值请求验证队列积压、MySQL 降级能力，以及 30 秒重建预算是否足够。

## 集成测试

设置 `RANKING_TEST_REDIS_ADDR` 和 `RANKING_TEST_MYSQL_DSN` 指向隔离的测试实例。空库设置 `RANKING_TEST_ALLOW_SCHEMA=1`，由实时榜测试应用基础榜单迁移。测试整点结算时预先应用三份榜单迁移，设置 `RANKING_TEST_PREPARED_SCHEMA=1`，MySQL DSN 带 `parseTime=true`。设置 `RANKING_TEST_REDIS_SHARD_ADDRS` 和 `RANKING_TEST_REDIS_PASSWORD` 后，分片及整点恢复测试使用四个指定主节点。保持运行中的榜单应用与集成测试数据库隔离，避免两者认领同一条结算任务。

```powershell
& 'D:\Program Files\GO\go1.25.1\bin\go.exe' test -count=1 -v ./services/ranking-service/...
```
