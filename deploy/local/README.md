# 本地依赖

在仓库根目录运行 `docker compose up -d` 启动 MySQL、Redis 和 RabbitMQ；用 `docker compose ps` 查看状态，`docker compose logs mysql` 检查 MySQL 初始化。`docker compose stop` 停止容器并保留数据，之后用 `docker compose up -d` 恢复。

本机连接地址分别是 `127.0.0.1:13306`、`127.0.0.1:16379`、`127.0.0.1:15673`。RabbitMQ 管理界面为 `http://127.0.0.1:15672`。默认 MySQL root 密码为 `activity-local-root`，RabbitMQ 用户和密码分别为 `activity`、`activity-local`。可通过项目根目录 `.env` 的 `MYSQL_ROOT_PASSWORD`、`RABBITMQ_USER` 和 `RABBITMQ_PASSWORD` 在首次启动前覆盖；这些默认值仅供本地开发，不用于生产。

MySQL 仅在数据卷首次初始化时按顺序执行现有建库和初始迁移。已有卷需要按当前 schema 补齐缺少的迁移。`compose.yaml` 包含依赖，叠加 `compose.services.yaml` 启动四个业务服务和 Web 入口。

## 榜单 Redis 四分片

日常开发可用共享 Redis。要验证实际四片分流和自动切主，叠加 `compose.ranking-ha.yaml`。该配置启动四组榜单 Redis 主从和三个 Sentinel。榜单 Redis 端口位于内部网络，可通过 `docker compose exec` 查看。

```powershell
$env:INTERNAL_TOKEN = [Convert]::ToHexString([Security.Cryptography.RandomNumberGenerator]::GetBytes(32))
$env:RANKING_REDIS_PASSWORD = [Convert]::ToHexString([Security.Cryptography.RandomNumberGenerator]::GetBytes(32))
$env:RANKING_REDIS_SENTINEL_PASSWORD = [Convert]::ToHexString([Security.Cryptography.RandomNumberGenerator]::GetBytes(32))
& 'D:\Program Files\GO\go1.25.1\bin\go.exe' run ./scripts/devkeys
docker compose -f compose.yaml -f compose.services.yaml -f compose.ranking-ha.yaml up -d --build --wait
```

`devkeys` 在 `.local/keys` 准备应用签名密钥。Web 入口使用 `web/h5/dist`，需要页面时按前端 README 构建。把榜单和 Sentinel 口令保存在本地 `.env` 或使用稳定的环境变量值。配置随 Redis 数据卷持久化，重新启动时使用同一组口令。

有活跃测试期次后，用故障验收脚本暂停并恢复当前主实例。可以用下列命令在默认本地库创建一个启用整点榜的期次。重复创建时使用新的期次 ID。`-VerifyHourly` 需要距下一整点至少两分钟。

```powershell
$env:MYSQL_DSN = 'core_app:dev-local-core@tcp(127.0.0.1:13306)/activity_core'
$env:EDITION_ID = '2026111501'
$env:EDITION_START = (Get-Date).ToUniversalTime().AddMinutes(-10).ToString('yyyy-MM-ddTHH:mm:ssZ')
$env:EDITION_END = (Get-Date).ToUniversalTime().AddHours(5).ToString('yyyy-MM-ddTHH:mm:ssZ')
$env:EDITION_HOURLY_RANKING = 'true'
& 'D:\Program Files\GO\go1.25.1\bin\go.exe' run ./services/activity-service/cmd/seed
.\scripts\Verify-RankingRedisHA.ps1 -Environment Compose -EditionId $env:EDITION_ID -BaseUrl http://localhost:8080 -VerifyHourly
```

## 整点榜单结算

|顺序|脚本|
|---|---|
|024|[活动整点窗口与入账](../../services/activity-service/migrations/005_hourly_ranking.sql)|
|032|[榜单整点任务与名单](../../services/ranking-service/migrations/003_hourly_ranking.sql)|

`EDITION_HOURLY_RANKING=true` 在创建新期次时预建整点窗口。请在第一笔参与或抽奖前完成启用。独立直连 Redis 和 Sentinel 模式的配置见 [榜单 Redis 故障恢复](../../docs/榜单Redis增量实现.md)。整点结算流程见 [整点榜单结算](../../docs/整点榜单结算.md)。
