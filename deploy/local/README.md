# 本地依赖

在仓库根目录运行 `docker compose up -d` 启动 MySQL、Redis 和 RabbitMQ；用 `docker compose ps` 查看状态，`docker compose logs mysql` 检查 MySQL 初始化。`docker compose stop` 停止容器并保留数据，之后用 `docker compose up -d` 恢复。

本机连接地址分别是 `127.0.0.1:13306`、`127.0.0.1:16379`、`127.0.0.1:15673`。RabbitMQ 管理界面为 `http://127.0.0.1:15672`。默认 MySQL root 密码为 `activity-local-root`，RabbitMQ 用户和密码分别为 `activity`、`activity-local`。可通过项目根目录 `.env` 的 `MYSQL_ROOT_PASSWORD`、`RABBITMQ_USER` 和 `RABBITMQ_PASSWORD` 在首次启动前覆盖；这些默认值仅供本地开发，不用于生产。

MySQL 仅在数据卷首次初始化时按顺序执行现有建库和初始迁移；已有卷不会自动执行新的迁移。初始化失败时检查日志和现有表，不能直接重跑全部 SQL。停止容器不会删除数据；不要使用 `docker compose down -v`，除非明确决定删除本地数据库及消息数据。当前 Compose 只包含依赖，四个业务服务尚未实现或接入。
