# 榜单 Redis 增量更新

本次按确认后的简化方案实现：不新增 Redis 版本号，不实现重建或代际切换。MySQL 的原始事实、连续序号投影、截止核对与排名发奖保持原有职责。

## 更新

activity 本地事务提交后，用100ms独立超时调用 ranking 内部 `/internal/ranking/v1/projection`；活动 Outbox Worker 发布相关MQ事件前也尽力调用一次。失败不影响已提交抽奖，也不阻止MQ发布。ranking 消费MQ时先提交MySQL事实，再执行同一缓存更新，成功后ACK；失败可重投。

同步与MQ传递同一份 `FormalDrawCommitted.v1` 增量：user_id、edition_id、draw_id、consumed_petals、occurred_at_sec。draw_seq只作为既有业务事实及重复载荷核对字段，不用来决定Redis更新先后。

Lua在一次执行中：

1. 按 draw_id 查同一期去重Hash；同载荷重复跳过，异载荷返回冲突。
2. 累加原始积分。首次正分使用本次业务时间；已有正分取最大业务时间。迟到的0分初始化不覆盖正分。
3. 计算 `packed = score * (endSec-minSec+1) + endSec-reachedSec`。
4. 写ZSET、原始积分/时间和去重记录。ID始终为字符串，反向补齐20位成员保证同分同秒时小ID先。

上下界来自核心期次的开始/结束秒，按期次固定。当前参与初始化发生在活动开始之后。Lua只对不超过2^53-1的精确整数执行计算；超界或期间修改编码边界会禁用该期缓存，继续使用MySQL。

去重Hash不设短期TTL，否则延迟MQ重投可能重复加分。其空间与该期抽奖批次数相关；目前不自动清理，不能在仍可能重放时单独删除它。

## 查询及当前范围

榜单优先从一个只读Lua脚本取得Top20、本人、原始积分和名次。Redis不可用、缺少必要数据、已有MySQL参与者尚未进入缓存或精度越界时，整个响应回退MySQL。保留一次MySQL参与人数/本人存在性查询用于识别未初始化的旧期次，不宣称榜单查询完全不访问数据库。

若某个已有正分的MySQL用户在Redis中缺少历史数据，禁用该期缓存，避免把新抽奖增量当成其全部积分。现有老期次可能因此继续走MySQL；这次不回填历史或重建。新期次从0分初始化和可靠事件开始累计。

本期不处理Redis部分键丢失后的自动修复，也不提供重建切换。Redis应使用持久化且不单独淘汰榜单/去重键。缓存有异步延迟，MySQL仍是事实及结算依据。

## 验证

真实Redis测试覆盖并发重复投递、先后颠倒、迟到初始化、同分时间排序、大于2^53的字符串用户ID、同分同秒ID排序、重复ID异载荷、编码越界和非法时间不写入。

```powershell
$env:RANKING_TEST_REDIS_ADDR = '127.0.0.1:<测试Redis端口>'
& 'D:\Program Files\GO\go1.25.1\bin\go.exe' test -v -count=1 ./services/ranking-service/internal/ranking/infrastructure/redis
```

2026-11-05验证：真实Redis并发脚本测试及真实MySQL/Redis组合测试通过；包括同步增量先到、MQ乱序/重复补偿、SQL提交后Redis连接失败再重投，以及旧SQL积分缺少缓存时拒绝返回残缺缓存。四服务编译/测试及受影响模块go vet通过。测试仅使用本轮临时容器，结束后已移除，未启动或改动原Compose项目。
