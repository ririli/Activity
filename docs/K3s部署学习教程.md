# Activity 的 Kubernetes / k3s 部署学习教程

适用环境：Windows、PowerShell 7、Docker Desktop 的 Linux 容器模式。项目：`D:\gopath\src\Activity`。部署日期：2026-11-06。

本文配合已经实际部署的 Activity 项目学习。先从浏览、查询和日志开始，再读清单、修改配置、观察恢复，最后理解它与企业生产环境的差距。

## 1. 先打开已经部署好的项目

访问 **https://localhost:18443**。

本机使用为 localhost 单独生成的自签名 TLS 证书，浏览器首次访问会提示证书不受信任。确认地址确实是这个 localhost 学习站点后，可以通过浏览器的“高级 / 继续访问”进入。脚本没有把证书加入 Windows 根证书库。正式环境应使用匹配真实域名的可信证书。

可以自行注册账号体验抽奖。验收会留下昵称为“K3s学习验收”的真实测试账号和业务记录；榜单上看到它们属于正常现象。注册保留每 IP 的频率限制，短时间反复注册可能需要等待一分钟。没有开放真实支付。

当前演示期次：`2026110601`，开始时间 `2026-11-01 00:00:00 +08:00`，结束时间 `2026-12-01 00:00:00 +08:00`。超过结束时间后，活动操作被拒绝是业务规则，不能通过重建数据库解决；应显式初始化新期次。

明文地址 `http://localhost:18080` 不提供业务入口，当前返回 404。请使用上面的 HTTPS 地址，否则 Secure Cookie 的行为也不同。

## 2. k8s、k3s、k3d 分别是什么

简单理解：**Kubernetes 管容器，k3s 是一个轻量的 Kubernetes 发行版，k3d 帮你把 k3s 节点运行在 Docker 容器里。**

这里的 Kubernetes API、Deployment、Service、Secret、PVC、Ingress 等概念可以迁移到其他 Kubernetes 发行版。k3d 的节点容器是本机学习的便利方式；生产部署不会因为创建了三个节点容器，就获得三台独立服务器的故障隔离。

本次版本固定为 k3d `v5.9.0`、k3s `v1.35.5+k3s1`，本机 kubectl 为 `v1.36.1`。安装脚本下载官方 k3d Windows 二进制，并核验官方 SHA256 清单。

集群名是 `activity-learn`，包含：

|节点|角色|本次用途|
|---|---|---|
|k3d-activity-learn-server-0|控制面，同时允许调度工作负载|API、调度、控制器及部分工作负载|
|k3d-activity-learn-agent-0|工作节点|运行分配到此节点的 Pod|
|k3d-activity-learn-agent-1|工作节点|运行分配到此节点的 Pod|

为利用学习机资源，server 也允许运行业务。企业生产集群通常隔离控制面，并使用多个独立故障域。当前三个节点共享同一台 Docker Desktop 主机，控制面只有一个。

官方说明：[k3d 服务暴露](https://k3d.io/stable/usage/exposing_services/)、[k3s 网络组件](https://docs.k3s.io/networking/networking-services)。

## 3. “一个 Pod 放一个服务”在这里如何落地

先区分三个容易混淆的词：

- **业务服务**：例如 identity-service，是你的 Go 程序与业务边界。
- **Pod**：Kubernetes 调度和重建的基本单元。本项目每个常驻 Pod 只有一个主要应用容器。
- **Kubernetes Service**：为一组 Pod 提供稳定访问地址、DNS 和流量转发，不是 Go 程序本身。

本项目四个 Go 服务分别拥有自己的 Deployment、Pod 与 Service。没有把四个服务塞进一个容器或一个 Pod。其他项目若有明确需求，Pod 内可以存在 sidecar 或 init container；“一个 Pod 只能有一个容器”不是 Kubernetes 的硬规则。[Kubernetes Pod 概念](https://kubernetes.io/docs/concepts/workloads/pods/)

|工作负载|控制器|副本|作用|
|---|---|---:|---|
|identity|Deployment|1|账号、JWT、会话与资料 RPC|
|activity|Deployment|1|资产、抽奖、奖励、活动进度与结算|
|ranking|Deployment|1|榜单事实、查询与排名结算|
|notification|Deployment|1|Redis 订阅与 SSE 实时通知|
|web|Deployment|1|H5 静态页面及已有 Nginx API 入口|
|mysql|StatefulSet|1|三个隔离的逻辑库，持久保存业务事实|
|redis|StatefulSet|1|限流、缓存与实时广播，启用 AOF|
|rabbitmq|StatefulSet|1|可靠事件传递，持久化 broker 数据|

因此正常状态是 **8 个常驻 Running Pod**。建表、初始化和网络验证的 Job 也会创建 Pod；它们完成后显示 `Completed`，不是故障。

MySQL 的三个逻辑库仍在一个实例内，分别由不同账号访问。`user_id % 4` 是活动核心库内的分表规则，与三个 Kubernetes 节点没有对应关系。

请求链路：

```text
浏览器 https://localhost:18443
  → k3d 的本机端口映射
  → k3s 自带 Traefik Ingress（终止 TLS）
  → web Service → Nginx Pod
      ├─ H5 页面
      ├─ /api/identity/     → identity Service → identity Pod
      ├─ /api/activity/     → activity Service → activity Pod
      ├─ /api/ranking/      → ranking Service → ranking Pod
      └─ /api/notification/ → notification Service → SSE

identity / activity / ranking → 各自拥有的 MySQL 逻辑库
业务服务 → Redis、RabbitMQ（仅按实际调用关系放行）
```

数据库、Redis、RabbitMQ 没有映射到 Windows 公网或局域网端口。对外入口只绑定 `127.0.0.1`。

## 4. 先认识这些 Kubernetes 对象

|对象|直观作用|本项目示例|
|---|---|---|
|Namespace|给一组资源划分管理范围|activity|
|Deployment|声明应用应运行几个副本，Pod 消失后负责补建|identity、activity 等|
|ReplicaSet|Deployment 管理具体副本时使用的控制器|通常由 Deployment 自动生成|
|StatefulSet|提供稳定编号、持久卷关联与有序管理|mysql-0、redis-0、rabbitmq-0|
|Service|Pod 改名、换 IP 后仍保持稳定访问入口|identity:8080、mysql:3306|
|Ingress|声明外部 HTTP/HTTPS 请求如何进入集群|localhost → web Service|
|ConfigMap|保存普通配置|服务 URL、限流模式、Nginx 配置|
|Secret|保存口令、令牌与私钥等敏感配置|数据库密码、JWT 私钥、TLS 私钥|
|PVC|申请持久存储并关联到 Pod|data-mysql-0 等|
|Job|执行有结束点的一次性任务|建表、活动初始化|
|NetworkPolicy|限制哪些 Pod 可以与哪些目标通信|默认拒绝后按业务放行|
|Role / RoleBinding|声明某个身份可对哪些资源执行什么操作|activity-reader 只读身份|

不要直接创建并长期维护一个孤立业务 Pod。你声明的是 Deployment/StatefulSet 的期望状态，控制器负责让实际状态接近期望状态。

## 5. 在 PowerShell 中连接这个集群

以下命令都在 **PowerShell 7** 执行，不是在 MySQL、Redis 或容器内的交互提示符中执行。

```powershell
Set-Location D:\gopath\src\Activity
$env:KUBECONFIG = 'D:\gopath\src\Activity\.local\k3s\kubeconfig.yaml'
kubectl config current-context
kubectl get nodes -o wide
```

第一行切换项目目录。第二行只为当前 PowerShell 会话选择 kubeconfig；不覆盖默认的 `C:\Users\20240\.kube\config`。后两行确认当前上下文和节点状态，应看到 `k3d-activity-learn` 与三个 `Ready` 节点。

```powershell
kubectl -n activity get deployments,statefulsets
kubectl -n activity get pods -o wide
kubectl -n activity get services,ingresses,pvc,jobs
```

`-n activity` 指定业务命名空间。第一条看期望副本与就绪副本；第二条看 Pod 的状态、IP 和所在节点；第三条看访问入口、存储和一次性任务。

重点读这些输出：

- `READY 1/1`：一个容器中一个已经就绪。
- `Running`：进程正在运行，还要结合 READY 判断能否接流量。
- `Completed`：一次性任务成功结束。
- `PVC Bound`：申请的持久卷已经绑定。
- Service 的 ClusterIP 是集群内地址，通常不能直接从 Windows 浏览器访问。

## 6. 看日志和事件，理解系统在做什么

```powershell
kubectl -n activity logs deployment/activity --tail=50
kubectl -n activity logs deployment/identity --tail=50
kubectl -n activity logs job/activity-schema-v1
kubectl -n activity logs job/activity-seed-v1
```

前两条查看业务日志。后两条确认数据库迁移和活动初始化执行情况。schema Job 日志应列出按编号执行的 SQL 文件；它不会在每次应用重启时重新建表。

```powershell
$podName = kubectl -n activity get pods -l app.kubernetes.io/name=activity -o jsonpath='{.items[0].metadata.name}'
kubectl -n activity describe pod $podName
kubectl -n activity get events --sort-by=.lastTimestamp
```

第一行通过标签取得当前 Pod 名，避免把教程里的旧随机后缀写死。`describe` 包含调度、镜像、探针、资源与事件；最后一条按时间查看命名空间事件。

当容器重启次数大于 0 时，可以进一步看上一次容器日志：

```powershell
kubectl -n activity logs $podName --previous --tail=100
```

`--previous` 需要该 Pod 曾经有已退出的容器。没有上一次实例时，命令报找不到日志是正常的。

## 7. 读懂一个 Deployment

完整文件在 `deploy\k8s\identity.yaml`。下面摘录关键字段用于阅读；实际部署使用仓库完整清单与 Kustomize overlay。

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: identity
  namespace: activity
spec:
  replicas: 1
  strategy:
    type: Recreate
  selector:
    matchLabels:
      app.kubernetes.io/name: identity
  template:
    metadata:
      labels:
        app.kubernetes.io/name: identity
    spec:
      automountServiceAccountToken: false
      containers:
        - name: identity
          image: activity-k8s/identity:20261106-1
          resources:
            requests:
              cpu: 50m
              memory: 96Mi
            limits:
              cpu: "1"
              memory: 256Mi
```

`selector` 把控制器与 Pod 标签关联起来。Service 也用标签选择目标 Pod。`template` 是补建新 Pod 时使用的模板，直接修改一个 Pod 不会自动改变这个模板。

资源参数的含义：

- `requests.cpu: 50m`：调度时申请约 0.05 个 CPU 核心的资源份额，不表示永远只能使用这么多。
- `limits.cpu: "1"`：CPU 使用上限；超出通常会被限速。
- `requests.memory: 96Mi`：调度依据之一。
- `limits.memory: 256Mi`：容器内存限制；超过可能被 OOMKilled。

这些是针对本机学习部署的起始值，不是未经压测的生产容量承诺。[Kubernetes 资源管理](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)

### 为什么现在只部署单副本，并用 Recreate

当前 Go 服务生成 ID 时使用固定的 `NODE_ID`。四个服务分别使用 1～4，初始化进程使用 100。直接把 activity 的 `replicas` 改成 2，会让两个进程使用同一个节点号；重叠滚动更新也会短暂产生同样的问题。

所以现在使用单副本 `Recreate`：先停止旧 Pod，再创建新 Pod。这会有短暂服务中断，但遵守当前代码的约束。多副本上线前，需要完成节点号的唯一分配、复用保护及并发验收；notification 的实例内连接额度也要明确集群语义。

企业部署不是把 `replicas: 3` 写进每个文件。副本数与发布方式必须和应用的并发、安全及状态管理能力一致。当前也没有为单副本写一个看似高可用、实际会阻止正常节点维护的 PDB。

## 8. 三种探针分别负责什么

|探针|判断问题|失败后发生什么|
|---|---|---|
|startupProbe|应用是否已经完成启动|启动失败达到阈值后重启；成功前不会让其他探针过早干扰|
|readinessProbe|当前是否适合接收业务流量|从 Service 的可用端点中移除，通常不重启容器|
|livenessProbe|进程是否已经失去正常运行能力|达到阈值后重启容器|

四个 Go 服务的存活检查使用 `/health/live`。identity/activity/ranking 就绪检查使用 `/health/ready`，notification 使用 `/health/subscription` 检查订阅状态。

数据库短暂不可用时，服务应先失去就绪状态。若把外部依赖故障直接当成所有 Pod 的存活故障，容易造成集体重启。

Kubelet 直接检查 Pod，不走对外 Nginx。外部访问 `/health/` 被入口拒绝，并不代表探针失败。

```powershell
kubectl -n activity get endpointslices -l kubernetes.io/service-name=activity
```

这条命令用于查看 activity Service 的端点信息。Pod 名称和 IP 可以变化，业务服务仍通过 Service 访问。

## 9. ConfigMap、Secret 与服务隔离

配置分成两类：

- `app-config` 等 ConfigMap 保存非敏感设置，例如服务地址、Cookie 是否 Secure、限流模式。
- Secret 保存随机数据库口令、内部令牌、JWT 私钥和 TLS 私钥。只有 identity 挂载 JWT 私钥；其余服务只读取 Access 公钥。

```powershell
kubectl -n activity get configmaps
kubectl -n activity get secrets
kubectl -n activity get configmap app-config
```

这里只列名称和普通配置，不需要把 Secret 解码打印到终端或文档中。Secret 的 Base64 编码本身不是加密；本集群另外启用了 k3s 的静态加密。备份控制面时还需要保护对应密钥。[k3s Secret 静态加密](https://docs.k3s.io/security/secrets-encryption)

本机敏感文件位于 Git 已忽略的 `.local\k3s\`：

- `credentials.json`：本次部署生成的随机口令，限制 Windows 文件 ACL。
- `keys\`：独立的 Access/Refresh Ed25519 密钥。
- `localhost.key`：本地 TLS 私钥。
- `kubeconfig.yaml`：具有集群管理能力的连接凭据。

保留这些文件，不要提交、发群或当成普通临时缓存清理。已有数据库时丢失本地口令文件，部署脚本会要求先恢复凭据，避免自动改密码导致服务连接失败。

数据库账号分别只授权 `activity_identity`、`activity_core`、`activity_ranking` 的 DML 权限。建表和初始化使用独立管理员 Secret，不把管理员口令分发给业务服务。

普通配置以环境变量注入后，修改 ConfigMap 不会改变已经运行的进程环境。修改后应显式重启对应 Deployment：

```powershell
kubectl -n activity rollout restart deployment/activity
kubectl -n activity rollout status deployment/activity --timeout=180s
```

第一条触发重建，第二条等待就绪。本项目的 Recreate 发布会短暂中断该服务，应在学习环境操作。

## 10. NetworkPolicy 和 RBAC 解决不同的问题

**NetworkPolicy 管网络访问，RBAC 管 Kubernetes API 权限。**

本集群在 activity 命名空间默认拒绝进出流量，再放行 DNS、Traefik → web、web → API、必要的服务调用及基础设施访问。业务容器不需要访问 Kubernetes API，因此关闭了 ServiceAccount token 自动挂载。

```powershell
kubectl -n activity get networkpolicies
kubectl -n activity describe networkpolicy mysql-clients
```

这两条用于观察哪些 Pod 可以访问 MySQL。策略通过标签匹配，不依赖会变化的 Pod IP。网络策略还需要网络插件实际执行；本次额外创建了携带正确 Redis 密码的无授权 Pod，确认连接仍被阻断，而不是只看 YAML 文件存在。

```powershell
kubectl -n activity auth can-i get pods --as=system:serviceaccount:activity:activity-reader
kubectl -n activity auth can-i get secrets --as=system:serviceaccount:activity:activity-reader
```

第一条应返回 `yes`，第二条应返回 `no`。本机管理员 kubeconfig 可以模拟这个身份来验收权限；这里没有给其他用户分发管理员权限。[Kubernetes NetworkPolicy](https://kubernetes.io/docs/concepts/services-networking/network-policies/)

## 11. 持久卷：Pod 消失为什么数据还在

```powershell
kubectl -n activity get pvc
kubectl get pv
```

本次三个 PVC：MySQL 5Gi，Redis 1Gi，RabbitMQ 1Gi。数字是声明的容量，不代表立即写满相应磁盘空间。

StatefulSet 会让 `mysql-0` 重建后继续挂载 `data-mysql-0`。验收确实删除并重建了 MySQL Pod，随后核对同一钱包的金币、花瓣和流水序号完全一致。

但是当前 StorageClass 是 k3s 自带的 `local-path`：

1. 它适合本机学习，数据绑定到一个节点的本地路径。
2. Pod 重建不等于节点损坏；这次验证不证明节点故障后可自动迁移数据。
3. 删除 PVC、节点容器或整个集群，都可能造成数据丢失。
4. 持久化不等于备份；生产需要独立备份、恢复演练及适合实际故障域的存储方案。

不要通过“删卷重建”处理迁移或应用错误。

## 12. 从零重现本次部署

先保证 Docker Desktop 已启动，并处于 Linux 容器模式；本机 Docker Linux 虚拟机分配约 8GB 内存。本例需要镜像和二进制下载能力。

```powershell
Set-Location D:\gopath\src\Activity
docker version
```

应同时看到 Client 和 Server 信息；只有 Client 或显示无法连接 daemon 时，先启动 Docker Desktop。

### 第一步：安装并创建集群

```powershell
.\scripts\Install-K3sLearning.ps1
```

脚本核验 k3d、创建或启动固定学习集群、开启 Secret 静态加密、导出独立 kubeconfig，并等待三个节点 Ready。k3d 安装在项目的 `.local\k3s\tools\k3d.exe`，没有要求你修改系统 PATH。

### 第二步：构建镜像并部署

```powershell
.\scripts\Deploy-K3sLearning.ps1
```

脚本完成这些动作：

1. 生成并保存本地随机凭据、独立 JWT 密钥和 localhost TLS 证书。
2. 创建命名空间、Secret、普通配置和迁移文件 ConfigMap。
3. 构建四个 Go 镜像与前端镜像。Go 使用项目的 1.25.1 SDK；前端按锁文件安装并构建。
4. 导出 Linux amd64 镜像归档，复制到三个节点的 containerd，并核对镜像实际存在。
5. 应用 Kustomize 清单，等待三个基础设施工作负载就绪。
6. 首次执行 schema Job，再执行 seed Job；完成过的 Job 不重复运行。
7. 等待五个 Deployment 就绪，保存本次镜像 tag。

镜像 tag 默认取执行时间，每次构建使用新 tag。已经部署的本次镜像 tag 是 `20261106-1`。不要反复修改同一个 tag 的内容。

Windows 下实测发现 k3d 的镜像导入可能显示成功却未导入，因此脚本使用单平台导出和逐节点 containerd 导入，并独立核对镜像清单。源码已处理这个兼容问题。

### 第三步：运行验收

```powershell
.\scripts\Verify-K3sLearning.ps1
```

它会注册真实测试账号，并进行业务、权限、网络和故障恢复检查。默认会重建 identity 与 MySQL Pod，因此只针对这个专用学习集群执行。业务限流保持 `enforce`，脚本通过适当间隔遵守限流。

成功后读取结果：

```powershell
Get-Content .\.local\k3s\verification.json
```

验证结果文件不保存访问令牌和密码。

## 13. 修改镜像、观察发布与回滚

代码或镜像构建内容改变后，使用新的 tag：

```powershell
.\scripts\Deploy-K3sLearning.ps1 -ImageTag 20261106-2
```

`-ImageTag` 表示本次发布版本。脚本构建、导入镜像，再用 `.local\k3s\overlay` 更新版本。基础清单中保留镜像名，实际 tag 由 overlay 提供。

同一版镜像已经构建并导入时，可以只重新应用配置：

```powershell
.\scripts\Deploy-K3sLearning.ps1 -ImageTag 20261106-1 -SkipBuild
```

这会复用现有镜像，保留数据库、Secret 与已完成 Job。新的 SQL 迁移应编写新的版本化迁移 Job；不要修改已完成 Job 的模板来假装执行了升级。

查看 Deployment 发布历史：

```powershell
kubectl -n activity rollout history deployment/activity
```

发生应用版本问题、并且确实存在上一个 revision 时，可以回退：

```powershell
kubectl -n activity rollout undo deployment/activity
kubectl -n activity rollout status deployment/activity --timeout=180s
```

回滚 Deployment 模板不会自动回滚数据库，也不会回滚外部 Secret/ConfigMap 的内容。执行前要确认应用和数据库版本兼容。

## 14. 建议按这个顺序学习

第一轮只运行第 5、6 节的查询和日志命令，找到一个 Deployment、一个 Pod、一个 Service、一个 PVC。

第二轮打开 `deploy\k8s\identity.yaml`，对照第 7、8 节找出 replicas、selector、resources、三种 probe、securityContext 和 Secret 引用。

第三轮观察自动恢复。先记录 Pod UID，再删除一个身份服务 Pod：

```powershell
$podName = kubectl -n activity get pods -l app.kubernetes.io/name=identity -o jsonpath='{.items[0].metadata.name}'
kubectl -n activity get pod $podName -o jsonpath='{.metadata.uid}'
kubectl -n activity delete pod $podName
kubectl -n activity get pods -w
```

最后一条持续观察，看到新 identity Pod Ready 后按 `Ctrl+C` 退出观察；它不会停止服务。新 Pod 的名称或 UID 变化，登录数据仍在 MySQL 中。

第四轮运行权限检查和 NetworkPolicy 查询，分别解释“没有网络访问权限”与“没有 Kubernetes API 访问权限”的区别。

这些练习已经足以建立完整部署的基本认识。业务服务的多副本扩容应等唯一 NODE_ID 分配等问题解决后再做。

## 15. 常见问题排查顺序

|现象|先查什么|不要直接做什么|
|---|---|---|
|Pending|describe 中的资源、PVC、节点及调度事件|删除 PVC|
|ImagePullBackOff|镜像 tag、节点 containerd 镜像清单、拉取策略|认为 Docker 本机有镜像就等于节点也有|
|CrashLoopBackOff|当前/previous 日志、写目录权限、配置、OOMKilled|不停盲目重启|
|Running 但 0/1|readinessProbe 和依赖状态|只看 Running 就认定部署成功|
|API 502|Nginx 日志、Service DNS、EndpointSlice、NetworkPolicy|把应用故障一概归因于网络代理|
|SSE ready 但无消息|订阅健康、Redis 广播、消息有效期及来源|假造业务中奖结果|
|初始化 Job 失败|Job 日志和数据库实际状态|未核对部分执行结果就重跑建表|
|注册返回 429|注册 IP 窗口及入口限流|关闭所有限流来掩盖问题|
|刷新失败|访问地址、Secure Cookie、Origin、会话状态|在 HTTP 地址期待 HTTPS Cookie 行为|

本例 Nginx 使用 `identity.activity.svc.cluster.local` 等完整服务域名。Nginx 的运行时 resolver 不等同于系统 resolver，不能假设它会自动给 `identity` 补齐 Kubernetes 的 search domain。

RabbitMQ 的 Erlang cookie 必须仅拥有者可读。清单使用非 root 用户、必要的持久写目录、标准输出日志，并处理 PVC 重新挂载时 cookie 权限的要求。

## 16. 停止、恢复与数据保护

学习完成后释放运行资源：

```powershell
& .\.local\k3s\tools\k3d.exe cluster stop activity-learn
```

这会停止节点容器与本集群工作负载。之后恢复：

```powershell
& .\.local\k3s\tools\k3d.exe cluster start activity-learn
kubectl get nodes
kubectl -n activity get pods
```

停止和启动保留节点及存储。不要把 `cluster delete`、`delete namespace` 或 `delete pvc` 当作普通停止命令；这些是不同级别的数据操作。

当前为方便继续学习，集群保持运行。原先的 Compose 项目和数据卷没有迁移或删除。

## 17. 本次部署的验收结果与生产边界

本次验收覆盖 HTTPS 页面、注册与 Secure/HttpOnly Cookie、账号事件初始化钱包、正式抽奖与幂等、阶段/购买、RabbitMQ 榜单投影与 gRPC 昵称、独立点亮事实、SSE、身份 Pod 自恢复、MySQL Pod 重建后的钱包持久化、RBAC、实际网络隔离、令牌续期/撤销、单容器工作负载和 Secret 静态加密。

2026-11-06 实测：**17 项检查全部通过**，三个节点 Ready，8 个常驻 Pod 均为 1/1 Running，3 个 PVC 均为 Bound，两个初始化 Job 均已完成。另有短期网络验收 Job，会自动清理。机器可读证据保存在 `.local\k3s\verification.json`，详细输出保存在 `.local\k3s\verify.log`。

本次采用了企业常见规范：声明式资源、独立服务边界、非 root 与只读根文件系统、关闭权限提升、资源约束、健康探针、Secret/配置分离、最小权限、网络隔离、版本化镜像、独立迁移和故障恢复验证。

生产上线还必须结合实际需求完成：

- 独立主机/故障域与多控制面，业务多副本前的唯一 ID 和并发设计。
- 高可用数据层、合适的存储、备份与恢复演练。
- 可信域名证书、镜像仓库与摘要固定、漏洞扫描及准入策略。
- 发布审批、数据库兼容性、完整指标/告警与容量压测。

本次前端构建的 npm 审计输出包含 24 项依赖提示（10 moderate、11 high、3 critical），未在部署任务中自动升级依赖。这个计数来自构建依赖树，需要另行评估与测试，不能直接等同线上可利用漏洞数，也不能忽略后宣称已经达到生产安全验收。

学习集群验证了部署机制与实际业务链路；它没有被包装成已经通过生产高可用或安全认证的集群。
