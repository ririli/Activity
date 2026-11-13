# 榜单 Redis 先写与故障恢复

本页记录当前实现。早期 [ranking-service 设计方案](ranking-service设计方案.md) 中的同步刷榜、Redis Hash 抽奖指纹及缓存写入入口已由本方案替代。数据库结构沿用现有迁移。

## 消费与查询

- 活动服务在正式抽奖事务内保存事实和 Outbox。榜单服务通过已有队列消费，正常情况下先更新 Redis，再提交榜单 MySQL。MySQL 提交后确认消息。
- Redis 每期按 `user_id % 4` 划分实时榜。每片使用 ZSET 排名、Hash 保存积分和达分秒、Set 保存已处理的 drawID。Lua 合并积分时取最大业务秒。重复 drawID 跳过缓存更新，仍尝试提交 MySQL。MySQL 用 `score_fact.draw_id` 主键去重并验证事实，按连续抽奖序号推进。Redis 可先显示已到事件的合计。
- 实时榜从四片归并 Top 20，并用所在片的排名与其他三片排在该用户前的人数计算本人名次。期次缓存异常时整份查询回退 MySQL，单实例最多允许 8 个并发降级查询，查询超时为 2 秒。活动结束榜读取 MySQL 的 `ranking_snapshot`。整点用独立 Redis 窗口，见 [整点榜单结算](整点榜单结算.md)。

## 故障与重建

- Redis 写入失败时，消费者继续保存 MySQL，并暂停读取该期缓存。Redis 写成功、MySQL 临时失败时，未确认事件会重投。同一代缓存内的 Set 防止重复加分。
- 服务启动后从 MySQL 找到待重建期次。Redis 脚本检查进程 run_id、代次标识和各集合计数。重建暂缓该期榜单消息处理，从一致性快照流式读取 `score_fact` 和 0 分参与记录，构建并验证新代后再开放。所有已落库事实均进入积分和去重 Set，包括跨序号缺口的事实。
- 榜单服务按单活部署。Kubernetes 使用 Recreate，Compose 保持一个 ranking 实例。重建时整期缓存先下线，继续对用户提供 MySQL 降级。结算核对独立写 MySQL，最终快照和奖励清单生成后整组退役 Redis 键。
- Set 空间随当期正式抽奖批次增长。上线前用预估参与人数和抽奖批次验证 Redis 峰值内存，以及当前 30 秒重建时限内能否完成恢复。

## 集成测试

设置 `RANKING_TEST_REDIS_ADDR` 和 `RANKING_TEST_MYSQL_DSN` 指向隔离的测试实例。空库设置 `RANKING_TEST_ALLOW_SCHEMA=1`，由现有实时榜测试应用基础榜单迁移。测试整点结算时预先应用三份榜单迁移并设置 `RANKING_TEST_PREPARED_SCHEMA=1`。

```powershell
& 'D:\Program Files\GO\go1.25.1\bin\go.exe' test -count=1 -v ./services/ranking-service/...
```
