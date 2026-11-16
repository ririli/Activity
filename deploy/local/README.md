# 本地部署与联调

## 仅启动依赖

在仓库根目录运行 `docker compose up -d` 启动三个 MySQL 实例、Redis 和 RabbitMQ。用 `docker compose ps` 查看状态，用 `docker compose logs mysql-identity mysql-activity mysql-ranking` 检查初始化。新卷会初始化各实例对应的库、迁移和应用账号。`docker compose stop` 停止容器并保留数据。

| MySQL 服务 | 逻辑库 | 宿主机端口 |
|---|---|---:|
| `mysql-identity` | `activity_identity` | `13307` |
| `mysql-activity` | `activity_core` | `13306` |
| `mysql-ranking` | `activity_ranking` | `13308` |

宿主机使用 `127.0.0.1`，容器使用对应服务名和端口 `3306`。可在首次初始化前用 `IDENTITY_MYSQL_ROOT_PASSWORD`、`ACTIVITY_MYSQL_ROOT_PASSWORD`、`RANKING_MYSQL_ROOT_PASSWORD` 分别设置管理密码，缺省时读取 `MYSQL_ROOT_PASSWORD`，本地默认值为 `activity-local-root`。Redis 仍用 `16379`，RabbitMQ 使用 `15673`，管理界面使用 `15672`。

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

## 启动完整服务

在仓库根目录使用 PowerShell 运行。Go 版本见根目录 `go.work`。

```powershell
go run ./scripts/devkeys
$env:INTERNAL_TOKEN = '<替换为至少 32 字节的随机字符串>'
$env:ALLOWED_ORIGIN = 'http://localhost:8080'
npm --prefix web/h5 ci
npm --prefix web/h5 run build
docker compose -f compose.yaml -f compose.services.yaml up -d --build
```

浏览器访问 `http://localhost:8080`。Nginx 同源提供 H5、API 与 SSE。开发密钥保存在 Git 忽略的 `.local/keys/`。使用已有 MySQL 卷时，先核对下文的迁移清单和应用账号授权。前端热更新见 [H5 开发说明](../../web/h5/README.md)。

MySQL 就绪后，用活动服务的种子程序创建一个本地期次。`EDITION_ID` 每次创建时取新的正整数，`NODE_ID` 在同时运行的进程间保持唯一，Compose 已使用 1～4。

```powershell
$env:MYSQL_DSN = 'core_app:dev-local-core@tcp(127.0.0.1:13306)/activity_core'
$env:EDITION_ID = [DateTimeOffset]::UtcNow.ToString('yyyyMMddHHmmss')
$env:EDITION_START = [DateTimeOffset]::UtcNow.AddMinutes(-1).ToString('yyyy-MM-ddTHH:mm:sszzz')
$env:EDITION_END = [DateTimeOffset]::UtcNow.AddDays(7).ToString('yyyy-MM-ddTHH:mm:sszzz')
$env:EDITION_HOURLY_RANKING = 'true'
$env:NODE_ID = '100'
go run ./services/activity-service/cmd/seed
```

## 数据库迁移

完整 Compose 在新卷上按以下顺序初始化 MySQL 8.4。应用账号分别拥有各自逻辑库的 DML 权限。

|顺序|脚本|
|---|---|
|000|各实例通过 Compose 的 `MYSQL_DATABASE` 创建对应逻辑库|
|010|[账号库初始迁移](../../services/identity-service/migrations/001_initial.sql)|
|011|[账号库雪花发号](../../services/identity-service/migrations/002_snowflake.sql)|
|020|[活动核心库初始迁移](../../services/activity-service/migrations/001_initial.sql)|
|021|[活动记录四分表](../../services/activity-service/migrations/002_records.sql)|
|022|[活动榜单投影](../../services/activity-service/migrations/003_ranking_projection.sql)|
|023|[活动结算调度](../../services/activity-service/migrations/004_settlement_scheduling.sql)|
|024|[活动整点窗口与入账](../../services/activity-service/migrations/005_hourly_ranking.sql)|
|025|[活动库雪花发号](../../services/activity-service/migrations/006_snowflake.sql)|
|030|[榜单库初始迁移](../../services/ranking-service/migrations/001_initial.sql)|
|031|[榜单结算续跑](../../services/ranking-service/migrations/002_settlement_resume.sql)|
|032|[榜单整点任务与名单](../../services/ranking-service/migrations/003_hourly_ranking.sql)|
|033|[榜单库雪花发号](../../services/ranking-service/migrations/004_snowflake.sql)|
|090|本地应用账号与授权 [账号](mysql/identity_user.sql) · [活动](mysql/activity_user.sql) · [榜单](mysql/ranking_user.sql)|

已有卷不会自动执行新迁移。备份并核对实际结构后，只按顺序补执行缺失的迁移。DDL 会隐式提交，初始化中途失败时先检查日志和已创建的对象。用 [只读核验 SQL](mysql/verify_schema.sql) 检查基础表数量及分表约束。仓库根目录的 `./scripts/Check-Migrations.ps1` 用于静态核验初始 DDL 和分表模板。

首次启用持久化发号时，先停止使用同一 `NODE_ID` 的旧发号进程，确认服务器时间已越过该节点旧版最后发号的毫秒，再启动新进程。若旧节点发生过时钟异常且无法确认已发时间，分配从未使用的节点号。账号、活动、榜单服务需显式配置 `NODE_ID`，不同实例分配不同节点号。`SNOWFLAKE_CLOCK_WAIT` 默认 `10ms`。超过时限时写请求可收到 `503 ID_UNAVAILABLE`，抽奖可以沿用原幂等键重试。每次向所属 MySQL 预留最多 250 毫秒的发号区间，数据库迁移或恢复时需保护 `snowflake_reservation` 的高水位。恢复旧备份时，沿用原节点号前需确认其时间已越过恢复点之后曾预留的上界。

### 从旧共享 MySQL 迁移数据

新实例使用三个新卷。要沿用旧环境的数据，先停止业务服务并备份旧 MySQL 的三个库，确认源库与当前业务表结构兼容。分别执行 `mysqldump --single-transaction --no-create-info --complete-insert --no-tablespaces <逻辑库>` 导出数据。停止旧 `mysql` 容器，释放活动库的 `13306` 端口，再启动三个新 MySQL 服务。将每份数据导入对应实例上刚初始化的空业务库，检查表行数、活动期次及钱包余额后再启动业务服务。旧 `mysql_data` 卷留作回退备份。

## 联调

`EDITION_HOURLY_RANKING=true` 在创建新期次时预建整点窗口。请在第一笔参与或抽奖前完成启用。独立直连 Redis 和 Sentinel 模式的配置见 [榜单 Redis 故障恢复](../../docs/榜单Redis增量实现.md)。整点结算流程见 [整点榜单结算](../../docs/整点榜单结算.md)。

使用独立 Compose 项目名和数据卷。追加 `-f compose.verify.yaml` 后，活动、账号、榜单 MySQL 分别映射到 `23306`、`23307`、`23308`，Redis 和 RabbitMQ 分别映射到 `26379`、`25673`。设置独立的 `HTTP_PORT`、匹配的 `ALLOWED_ORIGIN` 及空闲的 `PROXY_SUBNET` 和 `PROXY_IP`。种子程序和 `E2E_MYSQL_DSN` 使用测试活动库。

在运行 `go test -count=1 -v ./tests/e2e` 前，设置 `E2E_BASE_URL`、`E2E_MYSQL_DSN`、`E2E_REDIS_ADDR` 和 `E2E_EDITION_ID`。用例会注册测试用户并修改该用户的测试轮进度。测试服务可使用 `RATE_LIMIT_MODE=observe`，入口和注册限额仍由各自配置控制。限流复测见 [入口与业务限流](../../docs/入口与业务限流.md)。
