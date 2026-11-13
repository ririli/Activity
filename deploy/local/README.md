# 本地依赖

在仓库根目录运行 `docker compose up -d` 启动 MySQL、Redis 和 RabbitMQ；用 `docker compose ps` 查看状态，`docker compose logs mysql` 检查 MySQL 初始化。`docker compose stop` 停止容器并保留数据，之后用 `docker compose up -d` 恢复。

本机连接地址分别是 `127.0.0.1:13306`、`127.0.0.1:16379`、`127.0.0.1:15673`。RabbitMQ 管理界面为 `http://127.0.0.1:15672`。默认 MySQL root 密码为 `activity-local-root`，RabbitMQ 用户和密码分别为 `activity`、`activity-local`。可通过项目根目录 `.env` 的 `MYSQL_ROOT_PASSWORD`、`RABBITMQ_USER` 和 `RABBITMQ_PASSWORD` 在首次启动前覆盖；这些默认值仅供本地开发，不用于生产。

MySQL 仅在数据卷首次初始化时按顺序执行现有建库和初始迁移；已有卷不会自动执行新的迁移。初始化失败时检查日志和现有表，不能直接重跑全部 SQL。停止容器不会删除数据；不要使用 `docker compose down -v`，除非明确决定删除本地数据库及消息数据。当前 Compose 只包含依赖，四个业务服务尚未实现或接入。

## 整点榜单结算

|顺序|脚本|
|---|---|
|024|[活动整点窗口与入账](../../services/activity-service/migrations/005_hourly_ranking.sql)|
|032|[榜单整点任务与名单](../../services/ranking-service/migrations/003_hourly_ranking.sql)|

`EDITION_HOURLY_RANKING=true` 在创建新期次时预建整点窗口。请在第一笔参与或抽奖前完成启用。榜单默认在本地 Redis 上使用四个逻辑分片。为四片提供独立实例时，在 ranking 容器设置 `RANKING_REDIS_SHARD_ADDRS=host0:6379,host1:6379,host2:6379,host3:6379`，顺序对应 `user_id % 4`，各实例使用同一 `REDIS_PASSWORD`。结算流程和验证方法见 [整点榜单结算](../../docs/整点榜单结算.md)。
