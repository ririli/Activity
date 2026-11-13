# notification-service 分片广播实现与验收

日期：2026-11-05。实现基于当前工作区源码；测试环境为 Windows / Go 1.25.1 / Intel i7-10750H，以及 Docker 中的真实 Redis、Nginx、identity、activity、ranking。未修改数据库结构。

## 1. 已实现行为

一个 Hub 管理多个固定分片，每片有独立连接 map、锁、1024 条广播队列和一个常驻 Worker。Redis 接收后校验并调用 HTTP 编码函数一次，把不可变 Frame 非阻塞投递到所有分片。8 个 Worker 并发遍历各自连接，再非阻塞投递到每连接的 32 条队列，Handler 独立写入 SSE。

连接通过 `(next.Add(1)-1)%shardCount` 轮询分配并保存所属分片，不在运行时迁移。轮询均衡新连接，不能保证断开后的存量完全均衡。每实例和每 sid 配额通过独立配额锁、连接计数及 sessionCounts 统一维护，不扫描全体连接。注册与注销锁顺序统一为配额锁、分片锁；Worker 只获取分片锁。

同一分片串行处理入口消息，同一连接保持实际收到消息的顺序。跨片可以不同步，也可以独立丢弃。满分片队列只丢该片的新消息，满连接队列只丢该连接的新消息；Worker 和 Handler 都检查过期。更快的生产速度可能更早填满慢连接队列，减少延迟与丢弃更多展示消息可能同时发生，不将低延迟解读为必达能力。

`Connection.Messages` 为只读通道，Frame 的字符串按值共享。`Unsubscribe` 幂等并清理队列，不关闭通道。`Close` 幂等，撤销就绪、取消 context、等待 Worker、清理队列与连接计数；Handler 通过 `Done` 退出，已有网络写入最多受原有 5 秒写截止时间约束。父 context 取消也会启动清理。Redis 订阅故障撤销就绪；健康 PING 最多每秒一次，实际订阅读错误也触发重连。

HTTP 地址、SSE 字段、ready/auth_required/unavailable 事件、心跳、周期鉴权和 Redis 频道保持兼容。application 只依赖标准 context、领域值对象及编码函数，HTTP 层将配额错误映射为原有 429/503。

## 2. 配置与观测

|环境变量|默认值|含义|
|---|---:|---|
|NOTIFICATION_HUB_SHARDS|8|分片数，启动后不调整|
|NOTIFICATION_MAX_CONNECTIONS|10000|实例总连接准入上限|
|NOTIFICATION_SESSION_LIMIT|5|同一 sid 跨分片连接上限|
|NOTIFICATION_SHARD_QUEUE|1024|每片广播队列深度|
|NOTIFICATION_CONNECTION_QUEUE|32|每连接发送队列深度|

未设置时使用默认值；显式设置为空、非整数、零或负数均启动失败。Compose 已透传这些变量。同一 sid 限额仍为每实例限制。

`Hub.Stats()` 提供当前连接数、各分片队列深度、非法/编码失败计数、分片队列丢弃、连接队列丢弃、Worker 过期丢弃计数。主进程每 10 秒输出一次结构化汇总，无逐消息日志。分片丢弃单位为“消息×分片”，连接丢弃单位为“消息×连接”，过期计数单位为“分片帧”；这些指标都不是浏览器播放成功数。

## 3. 已执行验证

- notification 单元测试及 race 检查通过：默认 10000 准入上限、8 片轮询、跨片 sid 限额、并发配额竞争、广播顺序、仅编码一次、慢分片隔离、两级队列丢弃、非法/过期消息、重复注销、并发关闭与父 context 取消。
- 真实 HTTP 流测试通过：展示帧兼容、429/503 及 Retry-After、请求断开、Hub 关闭、订阅不可用后的 SSE 退出与配额清理。
- 真实 Redis 订阅测试通过：20 条消息按序到达，不再每条 PING；Redis 客户端故障撤销就绪、拒绝新连接并能取消订阅循环。
- 现有 `TestProjectionAndSSE` 使用真实四服务、Redis 和 Nginx，通过零积分榜单投影及 SSE 广播测试。测试启动临时 notification 容器运行本次构建产物，不替换原服务镜像或旧容器二进制。
- 持续内存测试覆盖 1000/10000 连接、1/8 分片、每秒 1/10/100 条消息，分别在 GOMAXPROCS=1/4/8 下运行；每种场景追加 64 条瞬时突发。延迟统计覆盖实际持续消费的模拟读者，而非仅统计入口入队。

本机 50 次广播完成基准的一个快照（10000 模拟连接，GOMAXPROCS=8）：

|流程|平均完成时间|每广播分配量|
|---|---:|---:|
|原算法，逐连接 JSON 编码|14.08 ms|约 11.44 MB|
|共享编码，1 分片|8.43 ms|约 1.68 KB|
|共享编码，8 分片|8.41 ms|约 1.68 KB|

本机这组数据主要证明共享编码减少分配；8 片相对 1 片没有稳定的完成时间优势。读者调度和共享汇总确认通道也在计时路径内，不能从此宣称 8 倍加速。另已覆盖所有连接集中单片的不均衡情况。

单独区分持续阶段和突发阶段的复测：10000 连接、每秒 100 条、GOMAXPROCS=8，1 片和 8 片持续阶段都完成 201 条广播、2010000 次模拟接收，没有队列丢弃。追加 64 条突发后，8 片发生 287675 次连接队列丢弃，仍保持有界队列；1 片在该次运行未丢弃。结果受调度影响，仅是短时测试，不代表长期稳定性或消息必达。

真实网络联调仅使用 **2 条 SSE 连接**，未放宽日常 Nginx 限额：

|发布负载|持续时间|实际/预期接收|P99 延迟上界|
|---|---:|---:|---:|
|1 条/秒|5 秒|10/10|4 ms|
|10 条/秒|35 秒|700/700|3 ms|
|100 条/秒，加 64 条突发|5 秒|1128/1128|2 ms|

上述网络测试发布失败和流异常均为零；35 秒运行跨过心跳及周期鉴权。小规模测试后容器进程 VmRSS/VmHWM 为 14712 KiB，此值不能用于推算万连接内存。内存测试另外记录 Go Heap/Sys、累计分配、GC 次数与暂停、注册/注销耗时；Go Sys 不是 RSS。

另对万连接、100 条/秒的 1/8 分片场景进行进程采样：两场景合计墙钟 4.66 秒、进程 CPU 11.36 秒，50 毫秒间隔采样的 RSS 峰值约 126.9 MiB。8 分片场景注册/注销全部模拟连接分别约 11.67/2.36 毫秒。该进程包含模拟读者、直方图与测试框架，RSS 不是 notification 网络进程的独立开销，CPU 秒也不等于单核墙钟时间。

本地原始日志位于 `.local/notification-shards/`：benchmark.log、sustained.log、steady-burst.log、resource-load.log、resource-process.json、e2e.log、sse-rate*.json、hub.log、process-memory.log。该目录不进入 Git；其中 tokens.json 是仅供本地负载工具使用的凭据文件，不能提交或分享。

## 4. 复测与万连接网络验收

在仓库根目录用 PowerShell 执行。单元测试和静态检查：

```powershell
& 'D:\Program Files\GO\go1.25.1\bin\go.exe' test -race ./services/notification-service/...
& 'D:\Program Files\GO\go1.25.1\bin\go.exe' vet ./services/notification-service/...
```

完成时间基准包含原算法、1/8 分片和偏斜分布；每条广播等待全部模拟读者消费完成。固定次数避免消息超过 30 秒展示期限：

```powershell
& 'D:\Program Files\GO\go1.25.1\bin\go.exe' test ./services/notification-service/internal/notification/application -run '^$' -bench BenchmarkBroadcastCompletion -benchtime=50x '-cpu=1,4,8' -benchmem
$env:NOTIFICATION_HUB_LOAD='1'
& 'D:\Program Files\GO\go1.25.1\bin\go.exe' test ./services/notification-service/internal/notification/application -run '^TestHubSustainedLoad$' -v '-cpu=1,4,8' -timeout=4m
Remove-Item Env:NOTIFICATION_HUB_LOAD
```

真实 Redis 集成测试需要单独提供测试 Redis 地址，不清空数据库，只使用唯一测试频道：

```powershell
$env:NOTIFICATION_TEST_REDIS_ADDR='127.0.0.1:26379'
& 'D:\Program Files\GO\go1.25.1\bin\go.exe' test -race ./services/notification-service/internal/notification/infrastructure -run TestRedisBroadcastSubscription -v
```

万连接完整网络压测尚未执行。已提供独立代理覆盖配置及 `cmd/sseload`；应在隔离项目中准备至少 2000 个不同登录会话的有效 Access token，JSON 文件格式为字符串数组，每 sid 默认最多复用 5 条连接。压测准备、建连及运行期间 token 都应有效。不要通过放宽日常鉴权或账号配额制造连接。

先运行 `scripts/Prepare-SSELoad.ps1`，从当前日常配置生成 `.local/sse-load/default.conf`；配置发生漂移会失败并要求复核。然后在隔离 Compose 项目的 Compose 文件参数最后追加 `-f compose.sse-load.yaml`，再接 `up` 等子命令。覆盖配置调整专用压测代理的请求/连接限额、worker_connections、文件描述符容量，并将该隔离 notification 的 SSE 建连速率桶容量由 3 调到 5，以允许同一 sid 在建连阶段连续打开 5 条连接；sid 资源上限仍为 5。不改写日常配置，不调整鉴权、注册和登录限制。应避免与日常项目共用端口、子网或数据卷。

```powershell
& ./scripts/Prepare-SSELoad.ps1
# 在隔离环境启动后：准备好 tokens.json，指定同一环境的 Redis 和频道前缀。
$env:SSE_LOAD_REDIS_ADDR='127.0.0.1:26379'
$env:SSE_LOAD_APP_ENV='local'
& 'D:\Program Files\GO\go1.25.1\bin\go.exe' run ./services/notification-service/cmd/sseload -url http://localhost:28080/api/notification/v1/barrage/stream -tokens .local/sse-load/tokens.json -connections 10000 -connect-rate 100 -rate 100 -burst 64 -duration 1m
```

工具通过真实 Bearer 鉴权并等待所有流 ready 后才发布测试弹幕，不自动重连、不补发、不修改业务数据。输出发布数、实际/预期投递数、流/发布失败和按毫秒直方图计算的延迟分位数；超过 60 秒延迟落入最后一桶。丢失展示消息不使进程报错，但输出的实际/预期数量必须纳入验收。工具的接收计数不是 H5 播放计数。

完整验收应分别以 1/10/100 条/秒运行，覆盖稳定期、突发、慢读者、断开重连与鉴权失效；同步采集 notification、identity、Redis、代理的 CPU、RSS、出口带宽、Hub 队列丢弃与连接清理。负载客户端也可能成为瓶颈，应部署在独立机器。未完成这些验证前，仅确认已实现万连接准入和分片广播能力，不承诺生产容量。
