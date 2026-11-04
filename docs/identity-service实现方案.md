# identity-service 技术选型与实现方案

日期：2026-11-04  
状态：待实施的详细设计；本轮仅生成文档，不初始化模块、不安装依赖、不修改或执行 DDL。  
依据：[业务需求 V1.2](Lottery业务需求文档（不含后台）.md)、[持久层设计 V2](持久层设计方案V2.md)及现有 `services/identity-service/migrations/001_initial.sql`。本文补充认证实现细节，不变更已确定的业务边界。

## 1. 方案结论

采用 **Access JWT + 轮换式 Refresh JWT + MySQL 会话 + Redis 撤销黑名单 + BigCache 负缓存**。注册按真实来源 IP 限流，登录失败按账号计数。

JWT 解决令牌签名与身份声明；MySQL 保存账号状态和会话撤销事实；Redis 加速撤销拒绝与分布式限流；BigCache 只短暂缓存“不在 Redis 黑名单”的查询结果。刷新不能只验 JWT 签名，必须核对数据库中的当前 Refresh Token 哈希。

初期没有流量和延迟实测，优先选用严格撤销模式：业务服务本地验 Access JWT，再调用 identity 内部会话校验接口；identity 查询自己拥有的会话和账号。BigCache 命中只跳过 Redis，不跳过数据库校验。因此退出成功后的后续校验可以识别撤销，Redis 丢数据也不会恢复旧会话。代价是每个受保护请求仍有一次内部调用和一次数据库查询；不能宣称此组合实现了无状态鉴权或显著减少数据库访问。

若后续实测要求减少内部调用，可以另行评审有界延迟模式：本地验签 + Redis 黑名单 + 短时负缓存，但必须接受最长缓存期限内的撤销延迟，并补齐禁用传播、Redis 重启重建与拒绝服务策略。本期不默认采用这个模式。

## 2. 范围与技术选项

负责注册、登录、续期、当前会话退出、当前账号信息、账号禁用校验及账号创建事件。允许同账号多设备登录，各次登录建立独立会话，退出只撤销当前会话。密码重置、第三方登录、短信、RBAC 和后台管理均不在本期范围。

|能力|建议选择|取舍与约束|
|---|---|---|
|语言|Go 1.25.1，建项时复核依赖兼容性|本机已有匹配 SDK；当前无 go.mod，不在本轮固定模块路径|
|HTTP|标准库 net/http + Go 1.22 起的 ServeMux 路由|接口数量少，减少框架依赖；统一实现错误处理和中间件|
|内部通信|私网 HTTP JSON|复用 HTTP 栈；不为少量认证接口引入 gRPC|
|MySQL|既有 MySQL 8.4，database/sql + go-sql-driver/mysql|显式事务、行锁和唯一键处理，不引入 ORM|
|Redis|go-redis/v9，Lua 原子限流|复用现有 Redis；不以 Redis 保存唯一会话事实|
|JWT|golang-jwt/jwt/v5；Ed25519 / EdDSA|只有 identity 持私钥，业务服务仅持 Access 公钥|
|密码|golang.org/x/crypto/argon2，Argon2id|随机盐、PHC 格式存储，禁止明文及快速散列密码|
|进程缓存|allegro/bigcache/v3|只用于黑名单负缓存，允许随时淘汰和重启丢失|
|日志配置|log/slog + 显式配置结构|敏感信息脱敏；密钥以只读文件挂载|
|交付|Docker + Compose + 已规划 Nginx|不增加注册中心、专用网关或 Kubernetes|

库版本在开发建项时检查维护状态、Go 版本约束与安全公告，然后固定具体版本和 go.sum；上述主版本不是已安装依赖。密码参数初值为内存19 MiB、迭代2、并行度1、随机盐16字节、输出32字节，部署环境压测后再提高；限制并发散列任务，避免容器内存被耗尽。[OWASP 密码存储建议](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html)

## 3. 领域及目录

`Account` 管理账号名、密码摘要、昵称和 ACTIVE / DISABLED 状态；`Session` 管理用户、当前 Refresh 哈希、固定到期时间和撤销时间。一个 Session 就是一个 Refresh Token 家族，刷新不换 session_id。

```text
services/identity-service/
  cmd/server/                    启动、依赖组装、优雅关闭
  configs/                       非敏感默认参数与配置说明
  internal/identity/
    domain/                      Account、Session、规则、仓储接口
    application/                 Register、Login、Refresh、Logout、GetMe、Introspect
    infrastructure/
      mysql/                     仓储、事务、Outbox
      redis/                     限流、撤销缓存
      cache/                     BigCache 适配
      security/                  JWT、密码散列、随机ID
    interfaces/http/             对外接口、内部接口、参数与错误映射
  migrations/                    已有 DDL，本轮保持原样
  test/integration/              MySQL/Redis 并发与故障测试
contracts/http/                  后续落地外部接口契约
contracts/rpc/                   后续落地内部 HTTP 契约
contracts/events/                后续落地账号创建事件契约
```

应用层依赖 PasswordHasher、TokenIssuer、RegistrationLimiter、LoginFailureLimiter、RevocationStore、Clock、IDGenerator 等接口，领域层不导入 HTTP、SQL、Redis 或 JWT 库。共享包只提供验签、HTTP 客户端和日志等技术能力，不共享 Account / Session 实体。

用户及事件 ID 采用应用生成的正数63位时间序列 ID，写入现有 BIGINT UNSIGNED；单实例固定节点号，多实例必须分配唯一节点号，时钟回拨停止生成并告警。外部 JSON 中 user_id、event_id 均输出十进制字符串，避免 JS 精度丢失；session_id 和 jti 使用 crypto/rand 生成至少128位随机标识。

## 4. Token 与浏览器存储

|项目|Access JWT|Refresh JWT|
|---|---|---|
|有效期初值|15分钟|7天固定会话期限|
|用途|访问业务接口|仅用于 identity 刷新和退出|
|签名密钥|独立 Access 私钥|独立 Refresh 私钥|
|声明|iss、aud、sub、sid、jti、iat、nbf、exp、token_use=access|同类字段，token_use=refresh，aud=identity-refresh|
|保存位置|H5 内存|Secure、HttpOnly、SameSite=Strict Cookie|
|服务端持久化|不保存完整令牌|login_session.refresh_token_hash 保存完整 JWT 字节的 SHA-256|

Access 的 aud 固定为本项目业务资源集合标识 `activity-api`，业务服务同时要求 token_use=access。Refresh 的 exp 始终等于首次登录确定的 session.expires_at，轮换不把7天不断后移。JWT 内不放密码、余额、昵称或敏感资料。

验签固定算法、issuer、audience、token_use，并要求 exp、iat、nbf、sub、sid、jti 完整合法；限制令牌长度，时钟容差初值30秒。kid 只能命中本地受信密钥集合，不允许由令牌指定 URL 拉取密钥。Access 与 Refresh 严禁互用。[JWT 校验选项](https://golang-jwt.github.io/jwt/usage/parse/)

Refresh Cookie 建议名 `__Secure-activity_refresh`，Path=/api/identity/v1/auth，省略 Domain；清理时保持同名同路径。所有认证响应 Cache-Control: no-store，不把 Token 放入 URL 或日志。生产 HTTPS；本地开发可用本地 HTTPS，关闭 Secure 仅限显式开发配置。

同源 Nginx 部署是默认条件。refresh、logout 等 Cookie 认证接口要求 POST、精确校验 Origin 并要求自定义请求头，拒绝未知 Origin；不允许任意源携带凭据的 CORS。Access 随 Authorization: Bearer 发送；页面刷新时先调用 refresh 恢复内存令牌。WebSocket/SSE 的浏览器认证传递方式由 notification 联调单独确定，禁止为了连接方便把长期 JWT 拼入查询串。

私钥只挂载给 identity，业务服务部署 Access 公钥集合。轮换先部署新公钥再切换签名 kid；旧 Access 公钥保留到已签发 Token 到期加时钟容差，Refresh 旧公钥保留至相关会话到期。计划外密钥泄露属于安全撤销操作，不能套用普通轮换保留策略。

## 5. 关键流程及事务

### 5.1 注册并自动登录

1. 限制请求体大小，校验账号 `^[A-Za-z0-9_]{3,20}$`、密码至少6个 Unicode 字符；设计增加最大128字符及512字节的资源上限，超限明确拒绝，不截断。密码不 Trim、不大小写转换。昵称默认账号名，非空昵称最多64个字符。
2. 账号大小写敏感，不转小写，与现有 ascii_bin 唯一索引一致。来源 IP 格式化后执行注册限流；仅受信配置中的白名单绕过该项限流。
3. 计算密码摘要、生成用户ID、session_id、双 Token 及 Refresh 哈希。CPU 密集散列放事务外。
4. 同一 MySQL 事务插入 account、login_session、AccountCreated.v1 Outbox。重复账号以唯一约束为最终判据，事务全部回滚，已消耗注册次数不返还。
5. 提交成功才发送 Cookie、Access 和用户资料；响应丢失时账号已存在，用户通过登录恢复，不把重复注册解释为重新赠送资产。

identity 不创建钱包、不写活动库。事件只传 event_id、event_type、schema_version、user_id、occurred_at；初始金币和花瓣由 activity 的初始化策略决定。activity 同一事务写 Inbox 去重及按 user_id 唯一的钱包初始化，重复事件不重复赠送。

初期 Outbox Worker 通过私网 HTTP 投递到 activity 的内部账号创建事件接收接口，不新增消息中间件；接收方提交事务后才返回成功。利用现有租约字段抢占任务、失败退避重试、积压告警，投递语义为至少一次。活动侧首次发现钱包未初始化时返回可重试的 ACCOUNT_INITIALIZING（503），不把余额默认为0；H5 有界退避重试，超时提示稍后重试，Outbox 持续恢复。

### 5.2 登录

1. 参数合法后读取账号失败计数，已达5次直接429；不存在的账号同样参与计数。
2. 按大小写敏感账号查库并校验密码；不存在时使用固定的有效虚拟摘要执行同样的散列验证，避免明显时序差异。不缓存“不存在账号”，避免注册成功后短时不能登录。
3. 不存在或密码错误统一401 INVALID_CREDENTIALS，原子增加失败次数并从本次失败重新计时15分钟。只有密码校验通过后才返回账号禁用提示，避免直接暴露账号是否存在。
4. 正确且 ACTIVE 时开启短事务，锁定并复核账号状态，插入新会话；提交后清除失败计数、返回双 Token。清除计数失败返回可重试503，不误报密码错误，已创建会话不会对客户端发放 Token。

并发请求可能在第5次失败完成前已进入密码校验，本方案限制的是后续接纳，并非保证总共只执行5次密码散列。新增的并发失败仍原子计数；成功清零与失败计数以 Redis 命令执行顺序为准。若需要严格串行的账号登录顺序，应另行设计，不使用会超时的分布式锁掩盖这个边界。

### 5.3 刷新与重放

1. 校验 Refresh 签名、用途、到期和浏览器来源；由可信 sub、sid 定位会话。
2. 所有修改遵守先锁 account 再锁 login_session 的顺序；校验 ACTIVE、用户匹配、未撤销、数据库会话未到期。
3. 对完整 Refresh JWT 求 SHA-256，与当前会话哈希做恒定时间比较。
4. 匹配则生成新 jti 和新 Token 对，在同一事务替换 refresh_token_hash，保持 session_id 和 expires_at 不变；提交后才返回。
5. 对于签名有效、用户与会话匹配、仍未过期但哈希不匹配的旧 Refresh，视为重放，提交该会话 revoked_at，随后写 Redis 撤销标记，返回401并清 Cookie。不能因随机 sid 或签名错误撤销别人的会话。

采用严格单次使用，没有宽限重用窗口。两个并发刷新中一个先成功，另一个会触发整个会话撤销；刚签出的 Access 也会被后续校验拒绝。H5 同页面必须使用 singleflight，多标签页通过 Web Locks / BroadcastChannel 协调；Refresh 网络结果未知时不自动重试旧 Token，转重新登录。此选择牺牲少量弱网体验，换取无需新增历史令牌表的清晰重放检测。[RFC 9700 刷新令牌保护](https://www.rfc-editor.org/rfc/rfc9700.html#section-4.14)

### 5.4 退出及禁用

退出使用 Cookie 中有效签名的 Refresh 定位当前会话，即使 Access 已过期也能退出；不要求 Refresh 哈希仍为当前版本，撤销该 sid 本身是安全收敛操作。事务锁定并将 revoked_at 设置为当前时间，已撤销则幂等成功。提交后尽力写 Redis 并删除本机 BigCache 条目，再清 Cookie。

退出成功的权威点是 MySQL 提交，Redis 写失败记录告警但不阻止退出成功，因为严格模式每次都检查数据库。数据库不可用返回503并清本地 Cookie，但必须提示服务端退出尚未确认；不宣称远端令牌已撤销。

禁用不是本期公开接口，但内部状态规则要完整：锁账号设置 DISABLED，并撤销其未撤销会话后提交；逐会话尽力写黑名单。后续重新启用账号不得恢复旧会话。禁用、登录、刷新采用统一锁序防止并发绕过。退出只针对当前会话，不撤销其他设备。

## 6. Redis 黑名单与 BigCache 负缓存

### 6.1 数据与校验路径

|Key|值|有效期/用途|
|---|---|---|
|identity:revoke:sid:{sid}|1|到该会话 expires_at + 30秒；覆盖同会话所有 Access 和 Refresh|
|identity:limit:register:{ipDigest}|次数|每次接纳尝试重置60秒|
|identity:limit:login:{accountDigest}|失败次数|每次失败重置15分钟|

IP 和账号键使用独立密钥的 HMAC-SHA256，减少原始信息暴露；所有实例使用一致密钥，账号大小写保留。Redis 使用 ACL 和网络隔离；限流与撤销键不应因普通缓存写入而被任意淘汰，监控内存和写失败。

业务服务本地验签后将原始 Access 通过受保护内部 HTTP 传给 identity；identity 再验签，不能信任调用方直接传入的 user_id 或 sid。内部接口不对公网路由，使用 Compose 私网和独立服务凭据认证，跨主机时启用 TLS；凭据不能替代用户 Token 验证。

identity 校验顺序：

1. BigCache 发现尚未到期的“该 sid 不在 Redis 黑名单”记录时，跳过 Redis 查询。
2. 未命中则查 Redis：命中撤销直接401；明确不存在才写负缓存；超时或连接错误不能当作不存在。
3. 无论负缓存是否命中，均从 identity MySQL 主库查询 Session JOIN Account，校验用户、状态、revoked_at 和 expires_at。数据库事实决定是否放行，不从异步副本读取撤销状态。
4. 返回内部用户身份上下文；业务层只使用已验证的 user_id。业务服务不能直接读取账号库。

BigCache 初值：负缓存逻辑 TTL 2秒、LifeWindow 2秒、CleanWindow 1秒、HardMaxCacheSize 16 MiB。值内保存 expires_at，每次 Get 自行检查，不能以后台清理是否执行判断过期；键用完整 sid 防止缓存键歧义。设置硬上限，缺失或淘汰视为普通未命中。[BigCache 配置说明](https://github.com/allegro/bigcache)

本机退出主动删负缓存只改善命中路径；其他实例无需依靠广播保证正确，因为仍查数据库。不为负缓存引入 Pub/Sub 失效协议。这里“负”指未发现撤销记录，不是未发现账号，也不是密码错误。

### 6.2 故障和一致性边界

|故障|行为|
|---|---|
|Redis 不可用|注册、登录及刷新返回503，防止限流与入口保护绕过；现有 Access 校验可仅使用 MySQL；退出仍以 MySQL 提交为准|
|BigCache 不可用/容量满|跳过本机缓存，正常查 Redis / MySQL|
|MySQL 不可用|注册、登录、刷新、鉴权返回503；不能仅凭签名及缓存放行业务请求|
|Redis 重启、撤销键被淘汰|MySQL 仍拒绝撤销会话；Redis 可按查询发现的撤销会话按剩余期限回填|
|identity 不可用|业务服务鉴权失败返回503，不能误报用户密码错误或盲目循环刷新|

严格撤销指：撤销提交后启动的权威状态查询拒绝旧会话。已在撤销前完成校验、正在执行的活动事务可以完成，不宣称退出能够中断已接纳抽奖。notification 长连接在建立时验证，并每30秒重新校验一次或在下一次客户端请求时校验；退出后长连接关闭存在该周期延迟，不能把 HTTP 的撤销语义直接套用于常驻连接。

## 7. 注册 IP 限流与登录失败限制

Nginx 覆盖客户端传入的来源头；Go 仅信任配置中反向代理 CIDR。直连请求使用 RemoteAddr，不能盲取 X-Forwarded-For 第一个值。IPv4 映射 IPv6 统一还原，IPv6 以规范化完整地址为粒度；不默认扩大到 /64，以免改变 IP 业务定义。容器后端不暴露公网端口。

注册使用 Lua 一次完成读取、判定、INCR 和 EXPIRE：

```text
格式不合法：400，不访问注册计数
受信白名单：跳过本项计数
count >= 3：429，返回剩余 TTL，不增加计数、不续期
count < 3：count += 1，TTL 设为60秒，放行到后续注册逻辑
```

例：t=0、20、40秒各接纳一次，则 t=40 秒计数3、到期 t=100；t=50 的第4次被拒绝，不把到期时间推到110秒。重复账号、后续散列或数据库失败均已计数；不退款。此规则是连续尝试后的静默窗口，不是自然分钟计数，也不是“任意60秒最多3次”的标准滑动日志算法。

登录失败计数 Lua 每次失败 INCR 并设900秒，成功 DEL；达到5后拒绝请求不延长 TTL。429 返回 Retry-After 秒数。注册 Lua 的读判写必须原子，不能在 Go 中先 GET 再 INCR。[Redis INCR 与 Lua 限流说明](https://redis.io/docs/latest/commands/incr/)

IP 限制不能防止代理池、IPv6 地址轮换或同 IP 多人误伤；初期接受该业务取舍。另以请求体上限、HTTP 超时和散列任务并发上限保护进程，默认不额外发明账号注册配额、验证码或设备指纹规则。

## 8. 接口与错误约定

统一外部前缀 `/api/identity/v1`，Nginx 保留完整路径。下表是待落地契约，不表示接口已实现。

|方法与路径|输入/认证|结果|
|---|---|---|
|POST /auth/register|account_name、password、nickname?|201 用户资料、Access、expires_in，设置 Refresh Cookie|
|POST /auth/login|account_name、password|200 同上|
|POST /auth/refresh|Refresh Cookie、受信 Origin、自定义头|200 新 Access、新 Refresh Cookie|
|POST /auth/logout|Refresh Cookie、受信 Origin、自定义头|204，撤销当前会话、清 Cookie|
|GET /me|Access|200 user_id、account_name、nickname|
|POST /internal/v1/identity/introspect|服务凭据及用户 Access；仅私网|200 经验证身份；内部路径不挂公网前缀|
|GET /health/live|无|进程存活|
|GET /health/ready|仅部署探针|依赖和密钥就绪；不输出连接串|

无 Cookie 的 logout 幂等204；畸形或签名无效的 Cookie 清除后401，不撤销任何会话。公共用户响应不含密码摘要、Refresh 哈希、内部会话状态。

错误体为 `{code, message, request_id}`。400 INVALID_ARGUMENT；401 INVALID_CREDENTIALS / TOKEN_INVALID / SESSION_REVOKED；403 ACCOUNT_DISABLED；409 ACCOUNT_EXISTS；429 REGISTER_RATE_LIMITED / LOGIN_RATE_LIMITED；503 DEPENDENCY_UNAVAILABLE。错误 Token 可以触发一次合并刷新；403、429、503不触发刷新循环。Refresh 重放对客户端统一表现为会话失效，审计日志记录具体原因。

## 9. 现有数据库适配

无需新增业务表或字段。account 沿用现有字段与大小写敏感唯一键；login_session 的 session_id 作为家族标识，refresh_token_hash 保存当前 JWT 完整字节摘要，expires_at 表达固定7天期限，revoked_at 作为最终撤销事实。行锁替换哈希不要求额外 version 字段。

现有 DDL 的 refresh_token_hash 注释描述“随机 Refresh Token”，与本次明确选择的 Refresh JWT 在形式上不同，但 BINARY(32) 可以直接保存 JWT 的 SHA-256。后续实施时仅校正注释，不能误以为需要把明文 JWT 写入该字段；本轮不改 DDL。

会话到期后分批清理，但至少保留至 expires_at + 时钟容差；在有效期内不能删除已撤销记录后按“不存在”重新放行。不存在会话一律拒绝。Outbox 使用现有 status、lease_owner、lease_until、retry_count 字段，不增加事件表；账号库不分片。

## 10. 实施顺序与验收

1. 初始化服务模块、配置、日志、健康检查和容器构建；按已安装兼容 SDK 选择绝对路径 Go / gofmt，不在本轮执行。
2. 实现领域规则、密码摘要、JWT 签发验签、MySQL 仓储及事务；完成大小写与唯一账号并发验证。
3. 实现 Redis Lua 注册/登录计数、受信代理 IP 解析；对照具体时间线验证 TTL。
4. 实现注册自动登录、登录、刷新轮换、退出和 /me；覆盖响应丢失及并发刷新。
5. 接入内部鉴权与 BigCache、业务服务认证客户端；验证多实例退出和 Redis 故障行为。
6. 接入 Outbox HTTP 投递及 activity 幂等初始化，最后进行 H5 Cookie、刷新协调和 Nginx 联调。

必要验收场景：

- 同账号并发注册仅一条账号、一份会话创建结果和一个创建事件；重复事件仅初始化一份资产。
- 格式错误不计数；前3次有效注册尝试计数；重复账号也计数；第4次不续期；白名单只跳过注册限流。
- 同名不同大小写账号按既有 ascii_bin 区分；错误账号和错误密码同提示；不存在账号同样进入失败计数。
- 第5次密码错误仍返回凭据错误，后续请求限流；15分钟从最后一次实际失败计数起算，成功清零。
- Refresh 不能访问业务接口，Access 不能刷新；错误算法、kid、issuer、audience、用途及缺失必要声明全部拒绝。
- 两个同时刷新只有一个先成功，第二个使会话撤销；响应丢失不会诱使客户端无限重试旧 Refresh。
- 退出与刷新并发后会话最终已撤销；BigCache 在其他实例仍命中负值也不能绕过 MySQL 撤销。
- 禁用与登录并发不生成可用的新会话；重新启用后旧 Token 仍无效。
- Redis 丢失黑名单不能恢复退出会话；MySQL 故障不能仅凭 JWT 放行；依赖故障返回503。
- Cookie 清除路径一致、CSRF 来源校验有效、日志无令牌和密码；前端多标签页刷新有协调。
- 压测记录验签、Redis、内部调用、SQL、Argon2 各段耗时，内存峰值及连接池等待；不预先承诺 QPS。

本方案中的15分钟 Access、7天会话、2秒负缓存、密码资源上限和30秒连接复核是初始技术参数，可配置；改业务规则须同步修订需求。当前无实际流量、并发目标与部署资源数据，容量结论留待实现后的压测，不因此引入更多组件。
