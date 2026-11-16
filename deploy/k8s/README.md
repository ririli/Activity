# Activity Kubernetes 学习部署

Windows / PowerShell 7 / Docker Desktop Linux engine。使用 k3d 5.9.0 承载 k3s 1.35.5+k3s1，集群名 `activity-learn`，1 个 server、2 个 agent。Kubeconfig 独立保存在 `.local/k3s/kubeconfig.yaml`。

```powershell
# 在仓库根目录执行。
.\scripts\Install-K3sLearning.ps1
.\scripts\Deploy-K3sLearning.ps1
.\scripts\Verify-K3sLearning.ps1
```

入口：`https://localhost:18443`。浏览器首次访问会提示本地自签名证书；只对当前 localhost 学习站点继续访问。脚本不向 Windows 根证书库添加信任。

## 文件职责

- `identity/activity/ranking/notification.yaml`：四个独立 Deployment 与 ClusterIP Service，每 Pod 一个业务容器。
- `mysql/`：账号、活动和榜单各一个 MySQL StatefulSet、Service、PVC 和管理员 Secret。活动的四组分表仍位于 `mysql-activity`。
- `redis.yaml`、`rabbitmq.yaml`：共享 Redis 与 RabbitMQ 的 StatefulSet、Service 和 PVC。
- `ranking-redis.yaml`、`ranking-redis-network.yaml`：四组榜单 Redis 主从、三个共享 Sentinel、固定 Pod DNS、独立 PVC 和网络访问规则。
- `web.yaml`：静态前端与已有 Nginx API 入口，Traefik Ingress 终止 TLS。
- `config.yaml`、两个 `nginx*.conf`：公开运行配置；口令与私钥由脚本创建为 Secret。
- `network-policies.yaml`：默认拒绝，单独放行 DNS、入口与有业务依据的服务调用。
- `rbac.yaml`：只读运维身份示例，可读 Pod/日志，不可读 Secret。
- `namespace.yaml`：命名空间、Pod Security 基线、资源配额和默认资源限制。
- `schema-job.yaml`：新数据库一次性迁移 Job；管理员 Secret 仅供迁移/初始化使用。
- `seed-job.yaml`：演示期次 `2026110601` 初始化，2026-11-01 至 2026-12-01。过期后应显式建立新期次。
- 两个 Dockerfile：构建 Go 服务（活动镜像含 `/seed`）与前端；web 专属 dockerignore 保证根目录排除规则不会遗漏前端源码。

## 安全与数据

应用使用非 root 用户、只读根文件系统、关闭权限提升、丢弃 capabilities、RuntimeDefault seccomp、不挂载 ServiceAccount token。只读运维 Role 不含 Secret 或写权限。K3s 启用 Secret 静态加密；随机密码、JWT 私钥、TLS 私钥和管理员 kubeconfig 仍属于敏感文件，保存在 Git 忽略的 `.local/k3s/`。

三个应用分别连接 `mysql-identity:3306/activity_identity`、`mysql-activity:3306/activity_core` 和 `mysql-ranking:3306/activity_ranking`，应用账号仅拥有本库 DML 权限。迁移 Job 按实例运行，使用各自管理员账号。迁移与 seed Job 保留完成状态，重复部署不会重新初始化业务数据。若 Job 失败或凭据文件丢失，先检查现有资源和数据。

### 已有共享 MySQL 的学习集群

1. 运行 `./scripts/Deploy-K3sLearning.ps1 -PrepareDatabasesOnly`，创建三个新实例并应用各自迁移。旧服务继续使用原来的 `mysql` 和 PVC。
2. 停止旧集群写入和消费者。使用旧 `mysql-0` 导出 `activity_identity`、`activity_core` 和 `activity_ranking`。源库结构与新迁移一致时，使用 `mysqldump --single-transaction --no-create-info --complete-insert --no-tablespaces <逻辑库>`，分别导入 `mysql-identity-0`、`mysql-activity-0`、`mysql-ranking-0` 对应的空库。核对表行数、活动期次和钱包余额。
3. 运行 `./scripts/Deploy-K3sLearning.ps1 -ConfirmMySQLCutover`，将应用切到新连接。已有默认期次时跳过 Seed。验证完成后，按保留策略处理旧 `mysql` 工作负载，保留旧 PVC 作为回退数据。

## 部署约束

账号、活动和榜单服务使用固定 `NODE_ID`，Deployment 采用单副本 `Recreate`。扩副本时需为发号进程分配独立节点号。`mysql-*-schema-v2` Job 在新应用启动前创建发号预留表。首次切换时确保旧发号进程已退出且系统时间已越过旧版最后发号的毫秒；无法确认时使用从未分配的节点号。通知服务还有实例内连接额度等扩容语义需要验证。 [发号配置与恢复](../local/README.md#数据库迁移)

当前属于采用企业常见交付规范的学习集群。三个节点共享同一 Docker Desktop 主机，单控制面与单数据库不具备生产高可用；local-path PVC 证明 Pod 重建后数据保留，不提供节点故障迁移、跨机副本或备份。生产还需要独立故障域、高可用数据库、可恢复备份、可信 TLS、镜像仓库/摘要与扫描、环境审批及监控告警等。

榜单 Redis 成对跨节点调度，三个 Sentinel 分布在三个节点。部署脚本等待四组复制链路在线，并核对三个 Sentinel 的仲裁和主节点地址。共享 `redis` Service 继续提供应用限流、会话撤销及通知。榜单应用通过 Sentinel 提供的 Pod DNS 连接各片主节点。

## 更新与停止

```powershell
# 每次构建使用一个新 tag；脚本创建本地 Kustomize overlay 并更新 Deployment。
.\scripts\Deploy-K3sLearning.ps1 -ImageTag 20261106-2

# 当前镜像已构建时可重新应用同一版。
.\scripts\Deploy-K3sLearning.ps1 -ImageTag 20261106-1 -SkipBuild

# 仅当前 PowerShell 会话选择此集群，不改默认 kubeconfig。
$env:KUBECONFIG = (Resolve-Path '.local/k3s/kubeconfig.yaml').Path
kubectl -n activity get pods,svc,ingress,pvc
kubectl -n activity logs deployment/activity --tail=50

# 学习结束后停止整个集群，之后 start 可恢复。
& .\.local\k3s\tools\k3d.exe cluster stop activity-learn
& .\.local\k3s\tools\k3d.exe cluster start activity-learn
```

配置以环境变量注入时，修改 ConfigMap/Secret 不会改变既有进程环境，需要显式重启相应 Deployment。SQL 迁移 Job 使用一次性版本名，不通过修改旧 Job 或删除 PVC 升级。不要用删除集群、命名空间或 PVC 来代替排障，这可能丢失学习数据。

## 验收记录

`Verify-K3sLearning.ps1` 检查 HTTPS、资产初始化、抽奖与幂等、阶段与购买、榜单/RPC、SSE、Pod 自动恢复、MySQL Pod 重建后钱包数据不变、网络隔离、RBAC 和刷新/撤销。默认启用真实业务限流，操作间隔用于遵守阈值。该脚本会在学习集群注册测试账号，默认还会重建 identity 与 MySQL Pod。

在隔离的学习集群中，使用活跃测试期次运行四分片故障验收。脚本使用 k3d 节点的 containerd 暂停 Redis/Sentinel 进程，检查 MySQL 降级、Redis 切主、主副本重启后的角色以及失去仲裁时的恢复。

```powershell
.\scripts\Verify-RankingRedisHA.ps1 -Environment K3s -EditionId 2026110601 -BaseUrl https://localhost:18443
```

成功运行后，验证结果位于 `.local/k3s/verification.json`。
