# Ops-Go 运维管理系统

## 项目简介

Ops-Go 是一个基于 Go 语言开发的企业级运维管理系统，提供主机管理、SSH 堡垒机、Web Terminal、会话录像与审计、批量任务执行、监控告警、日志中心、CMDB 资产管理等功能。系统采用前后端分离架构，支持多种认证方式和细粒度的权限控制。

### 核心特性

- 🖥️ **Web Terminal** - 浏览器直接连接 SSH，支持 xterm.js 全功能终端
- 🔐 **SSH 堡垒机** - 独立的 SSH 服务（端口 2222），支持命令行交互式操作；连续登录失败自动封禁 IP、非中国 IP 接入拒绝、来源归属地解析（ip2region）
- 🛡️ **高危指令拦截** - 实时检测并阻止危险命令执行，支持精确/前缀/正则三种匹配模式，内置 14 条防护规则，管理员豁免
- 📹 **会话录像** - 自动录制所有 SSH 会话，支持在线回放和下载，按保留天数自动清理
- 📊 **监控告警** - SSH 采集主机指标（CPU/内存/磁盘/网络）写入 Prometheus，自研 ECharts 实时面板，阈值规则评估 + Webhook 告警通知（HMAC 签名、重试、静默与冷却）
- 📗 **日志中心** - 复用堡垒机 SSH 通道增量拉取远程日志（目标机零 Agent）写入 Loki，提供类 Kibana 检索界面（多关键字高亮、时间分布直方图、实时 tail、上下文查看、CSV 导出）与日志告警规则，告警事件与通知链路复用监控模块
- 🗂️ **CMDB 资产管理** - 六类资产模型台账、生命周期流转、标签管理、ECharts 关系拓扑图、IP 网段管理（IPAM）
- ⚙️ **批量任务执行** - 脚本库、任务模板与可视化编排，支持命令/脚本/文件分发五种执行方式
- 👥 **权限管理** - 基于 Casbin 的 RBAC 权限控制，支持细粒度资源授权
- 🔑 **凭证管理** - 支持密码和密钥两种认证方式，凭证加密存储
- 📝 **审计日志** - 操作留痕（模块/类型/敏感标记/业务码/耗时）、Web 与堡垒机双通道登录日志、行为分析（成功率趋势、失败 Top 账号与来源、可疑账号识别）


---

## 系统功能

### 1. 用户管理
- 用户增删改查
- 用户状态管理（启用/禁用）
- 用户角色分配
- 用户部门归属
- 密码修改与重置

### 2. 角色管理
- 角色增删改查
- 角色菜单权限配置
- 角色用户关联
- 角色状态管理

### 3. 菜单管理
- 树形菜单结构
- 菜单权限标识配置
- 菜单图标、排序设置
- 按钮级权限控制

### 4. 部门管理
- 树形部门结构
- 部门增删改查
- 部门状态管理

### 5. 主机管理
- 主机增删改查
- 主机规格配置（CPU、内存、磁盘）
- 主机状态管理（心跳探测在线/离线）
- 局域网主机扫描与批量录入
- 主机分组管理
- 主机凭证绑定

### 6. 凭证管理
- 凭证增删改查
- 支持密码认证和密钥认证
- 支持 SSH、RDP、VNC 协议
- 凭证加密存储（RSA 加密）
- 凭证授权管理

### 7. 主机分组
- 分组增删改查
- 分组权限授权
- 分组内主机管理

### 8. 授权管理
- 用户主机授权
- 用户凭证授权
- 主机组授权
- 凭证组授权

### 9. Web Terminal
- 浏览器 SSH 连接
- 支持多终端窗口
- 终端自适应大小
- 终端尺寸前后端同步（连接时即以实际窗口尺寸创建 PTY，实时同步窗口调整）
- 支持复制粘贴
- 会话自动录制

### 10. 会话审计
- 会话列表查询
- 会话详情查看
- 会话录像回放
- 录像文件下载
- 会话统计分析

### 11. 操作日志
- 用户操作记录
- 登录日志
- 操作类型筛选
- 时间范围查询
- 扩展留痕字段：所属模块、操作类型、敏感操作标记、业务码与业务结果、User-Agent、操作时间
- 独立的登录日志与行为分析见「31. 登录日志与行为分析」

### 12. 系统监控
- 在线用户统计
- 系统资源监控
- 会话统计

### 13. 高危指令管理
- 拦截规则增删改查
- 三种匹配模式：精确匹配、前缀匹配、正则匹配
- 内置 14 条防护规则（rm -rf /、mkfs、dd 写磁盘、fork 炸弹、shutdown 等）
- 自定义规则管理（内置规则仅允许启停，不可删除）
- 规则状态启停控制
- Web Terminal 和 SSH 堡垒机双通道实时拦截
- 超级管理员（user ID = 1）豁免拦截

### 14. 脚本管理
- 脚本库增删改查
- 支持 Shell、Python 等脚本类型
- 脚本内容在线编辑与版本管理
- 脚本与任务模板关联

### 15. 任务模板
- 模板增删改查
- 支持三种任务类型：执行命令、执行脚本、分发文件
- 可配置超时时间、执行凭证、脚本语言
- 模板与脚本库联动

### 16. 任务编排
- 可视化多步骤任务流水线编排
- 支持步骤顺序配置、父子步骤关系
- 步骤失败重试策略（重试次数、失败处理）
- 编排与模板联动执行

### 17. 任务执行
- 五种执行方式：快速命令、快速脚本、快速文件、模板执行、编排执行
- 异步并发执行，支持多主机批量操作
- 实时执行状态追踪（待执行/执行中/已完成/部分失败/全部失败/已取消）
- 单台主机执行结果详情查看
- 执行记录分页查询与时间范围筛选
- 支持手动取消正在执行的任务

### 18. SFTP 文件管理
- 远程目录浏览与文件列表查看
- 文件上传（支持大文件流式传输）
- 文件下载
- 远程文件/目录删除
- 远程文件/目录重命名
- 远程目录创建
- 严格的权限校验（非管理员需授权凭证）

### 19. 活跃会话管理
- 实时查看当前所有活跃的 SSH 堡垒机会话
- 管理员可强制终止指定会话
- 会话信息包括用户、目标主机、连接时长等

### 20. 验证码
- 登录图形验证码
- 防止暴力破解与自动化攻击

### 21. 监控采集
- 采集目标管理（增删改查、启停、手动/自动同步纳管主机）
- SSH 定时下发采集脚本，采集 CPU、内存、磁盘、网络指标
- 指标经 Pushgateway 写入 Prometheus 时序存储
- 采集间隔、超时、并发可通过环境变量配置

### 22. 实时监控面板
- 自研 ECharts 面板：监控总览、单机指标实时曲线
- Prometheus 即时查询（instant）与区间查询（range）API
- 数据源连通状态检测（metrics/status）

### 23. 告警规则与事件
- 阈值告警：比较符 + 持续时长（ForSec），支持自定义 PromQL 表达式
- 规则试算（Test）、启停控制、规则静音（Muted）
- 告警收敛：指纹去重、冷却时间（CooldownSec）、静默期（SilenceUntil）
- 告警事件列表与详情、事件确认（Ack）、删除

### 24. 通知渠道
- Webhook 渠道：HMAC 签名、失败重试、通知日志落库（ops_monitor_notify_log）
- 渠道测试发送与启停管理
- 邮件/钉钉/企业微信渠道已预留模型常量，发送能力待实现

### 25. CMDB 资产台账
- 预置六类资产模型：服务器、网络设备、数据库、中间件、域名、SSL 证书
- 各模型固定属性表单（JSON 存储），资产增删改查与详情抽屉
- 资产生命周期流转（采购→上架→运行→维护→退役），每次流转记录变更日志
- 一键从纳管主机（ops_instance）导入为服务器资产

### 26. CMDB 拓扑 / 标签 / IP 管理
- 资产关联关系维护与 ECharts 力导向拓扑图（邻域聚焦 / 全量视图 ≤300 节点）
- 标签 CRUD、资产打标全量替换、台账标签筛选与 chips 展示
- IPAM 网段管理：CIDR 容量计算，资产 IP 与纳管主机 IP 自动归段，使用率与占用明细

### 27. 日志采集源
- 采集源增删改查、启停、手动立即拉取、位点重置
- 复用堡垒机 SSH 通道下发纯 sh 增量脚本（目标机无需安装 Agent，无 jq/python 依赖）
- inode + 位点续读、日志轮转跟随、通配符取最新文件、单轮字节上限与首轮回溯量可配
- 采集失败退避重试，失败原因回显到界面（文件不存在 / 不可读 / 超时 / 限速丢弃）
- 自定义标签注入前做合法性校验，非法名与 `job`/`host`/`level` 等保留标签被过滤，避免整批写入被拒

### 28. 日志存储与检索（Loki）
- 采集结果按级别与标签分组写入 Loki（`LOKI_URL` 配置，可选 Token 与 TLS 跳过校验）
- 写入被时间窗口拒绝时自动解析可接受边界、钳制时间重推一次，避免位点永久卡死
- 条件转 LogQL：按来源/主机/业务标签/级别/关键字（含排除）过滤，流选择器与行过滤器双重校验
- 检索界面：多关键字高亮、时间分布直方图（自动/1m/5m/15m/1h 步长）、实时 tail（3 秒增量去重 + 自动滚动）、CSV 导出
- 上下文查看：以命中行为界取同流前后 N 行，行数可调

### 29. 日志告警
- 规则条件：窗口内命中条数 ≥ 阈值，支持按主机分别评估或全量汇总
- 规则试算（不落库不通知）、启停、静默期、规则静音、冷却时间
- 事件写入与监控告警共享的 `ops_monitor_alert_event` 表（按 `source` 区分），通知链路复用 Webhook 渠道
- 告警事件页签：级别/状态/主机/时间范围筛选与事件详情

### 30. 堡垒机登录安全
- 同一来源 IP 连续登录失败 10 次自动封禁 30 分钟（`bastion.go` 中常量，封禁期内直接拒绝握手）
- 非中国 IP 接入拒绝（IPv6 暂放行），依赖内置 `data/ip2region.xdb` 归属地库
- 用户不存在/禁用、密码错误、凭证校验失败等分别记录失败原因
- 每次接入与拒绝均写入登录日志，含来源 IP、归属地、UA 解析出的浏览器与系统、成败与原因

### 31. 登录日志与行为分析
- Web 登录与堡垒机接入双通道留痕（`sys_login_log`）：账号、来源 IP 与归属地、UA、成败与失败原因、登录端
- 主动登出同样记录，便于还原完整会话链路
- 行为分析：登录成功率趋势、失败 Top 账号与来源 IP、活跃账号排行、可疑账号识别（失败 ≥5 次）

### 32. 操作审计分析
- 按所属模块、操作类型、敏感标记、时间范围检索操作留痕
- 行为概览：区间内操作总量、失败数、敏感操作数
- Top 榜：模块分布、活跃操作人、敏感操作接口排行
- 慢接口统计：按 URI 聚合平均/最大耗时与调用次数（分析窗口默认 24 小时，最长 30 天）

---

## 未来规划

> 当前迭代重点：监控告警侧还剩两项缺口 —— **通知渠道扩展**（邮件/钉钉/企业微信，模型常量已预留）与 **告警增强**（多条件组合、告警升级策略）。核心可观测性链路（指标 + 日志 + 告警 + 通知）已闭环，后续按下列优先级扩展场景。

### 短期规划 (v1.1)
- [x] 局域网主机扫描与批量录入
- [x] 批量主机导入（Excel/CSV 模板导入）
- [x] 主机状态自动检测（心跳/在线状态）
- [x] 文件上传下载（SFTP）
- [x] 录像自动清理策略
- [x] 危险命令拦截
- [x] 脚本库与任务执行
- [x] 任务模板与编排
- [x] 活跃会话管理与强制下线
- [x] 监控告警系统（采集 / Prometheus 存储 / 面板 / 规则 / Webhook 通知）
- [x] CMDB 资产管理（台账 / 拓扑 / 标签 / IPAM）
- [x] 日志中心（SSH 增量采集 / Loki 存储 / 检索界面 / 日志告警）
- [x] 审计日志增强（登录日志、敏感操作留痕、行为分析）

### 中期规划 (v1.2)
- [ ] 通知渠道扩展（邮件 / 钉钉 / 企业微信）
- [ ] 告警增强：多条件组合告警、告警升级策略（escalation）
- [ ] 日志与登录日志保留期清理策略（按合规要求配置保留天数）
- [ ] 日志采集增强：容器标准输出 / systemd journal 来源、更多级别解析规则
- [ ] 主机指标采集扩展（进程、端口、磁盘 IO 明细、自定义指标）
- [ ] 命令审计规则（正则匹配敏感命令告警）
- [ ] 操作审批流程
- [ ] 定时任务 / Cron 作业调度
- [ ] 录像对象存储（OSS/S3）
- [ ] RDP/VNC 协议支持
- [ ] 多因素认证（MFA/2FA）
- [ ] 登录安全增强（异地登录检测、设备指纹、全量 UA 识别库）

### 长期规划 (v2.0)
- [ ] 容器与 Kubernetes 集群管理（Docker 主机、Node/Pod/Deployment 查看与管理、kubectl Web Terminal、Helm）
- [ ] CI/CD 发布管理（流水线对接、灰度/蓝绿、制品与回滚）
- [ ] 配置管理（Ansible Playbook 集成、模板渲染、配置版本对比与下发）
- [ ] 数据库运维（实例管理、SQL 执行与审核、备份恢复、数据库监控）
- [ ] 操作工单系统（申请-审批-执行闭环、SLA 与报表）
- [ ] 云资源管理（多云纳管、资源同步到 CMDB、费用分析）
- [ ] 网络与证书管理（网络设备、网络拓扑、DNS/CDN、SSL 证书到期监控与自动续签）
- [ ] 通知中心与报表分析（站内消息、消息模板、运维/成本报表、PDF/Excel 导出）
- [ ] 对外 API 网关与 Webhook 事件推送（API Key、限流、审计）
- [ ] 资产自动发现（SNMP / Cloud API）
- [ ] CMDB 自定义模型与字段（当前为预置六类固定 schema）
- [ ] 多租户支持
- [ ] 高可用部署（多实例 + 负载均衡）
- [ ] 知识库 / 文档中心
- [x] Prometheus 监控集成（指标存储与查询已接入；Grafana 集成可选）


---

## 技术栈

### 后端
| 技术 | 版本 | 说明 |
|------|------|------|
| Go | 1.24+ | 编程语言 |
| Gin | 1.10+ | Web 框架 |
| GORM | 1.31+ | ORM 框架 |
| MySQL | 5.7+ | 关系型数据库 |
| Redis | 5.0+ | 缓存数据库 |
| Casbin | 2.x | 权限控制 |
| JWT | 3.x | 身份认证 |
| WebSocket | - | 实时通信 |
| gliderlabs/ssh | 0.3+ | SSH 服务器 |

### 外部依赖（可观测性，可选）
| 组件 | 说明 |
|------|------|
| Prometheus | 指标时序存储与查询（面板、告警评估数据源） |
| Pushgateway | 采集结果写入端（指标分组 `ops_node`） |
| Loki | 日志存储与检索引擎（`/loki/api/v1/*` push 与 query_range），日志中心必需 |

### 内置数据依赖
| 文件 | 说明 |
|------|------|
| `data/ip2region.xdb` | IP 归属地库（约 10MB），启动时整体载入内存；用于登录日志归属地解析与堡垒机「非中国 IP 拒绝」策略 |


### 前端
| 技术 | 版本 | 说明 |
|------|------|------|
| Vue | 3.2+ | 前端框架 |
| TypeScript | 5.0+ | 类型支持 |
| Vite | 4.x | 构建工具 |
| Element Plus | 2.3+ | UI 组件库 |
| Pinia | 2.x | 状态管理 |
| Vue Router | 4.x | 路由管理 |
| xterm.js | 5.x | 终端模拟器 |
| Axios | 1.x | HTTP 客户端 |
| Echarts | 5.x | 图表库 |

---

## 环境要求

### 开发环境
- Go 1.24+
- Node.js 16+
- MySQL 5.7+
- Redis 5.0+

### 生产环境
- Go 1.24+
- MySQL 5.7+ / 8.0+
- Redis 6.0+
- Prometheus + Pushgateway（启用监控告警功能时需要）
- Loki（启用日志中心时需要；未配置时采集与检索界面会提示数据源不可用）
- `data/ip2region.xdb`（登录日志归属地与堡垒机地域策略需要）


### 操作系统支持
- Linux (推荐 CentOS 7+, Ubuntu 18.04+)
- macOS
- Windows (开发测试)

---

## 项目截图

### 登录
![登录](./docs/images/login.png)

### 主机管理
![首页](./docs/images/首页.png)
![主机列表](./docs/images/主机列表.png)
![新增主机](./docs/images/新增主机.png)
![同步局域网主机](./docs/images/同步局域网主机.png)

### Web Terminal 与 SFTP
![远程1](./docs/images/远程1.png)
![远程2](./docs/images/远程2.png)
![堡垒机终端与SFTP文件管理](./docs/images/SSH-1.png)
![文件1](./docs/images/文件1.png)
![文件2](./docs/images/文件2.png)

### 凭证与授权
![凭证](./docs/images/凭证.png)
![授权](./docs/images/授权.png)

### 批量任务
![任务-1](./docs/images/任务-1.png)
![任务编排](./docs/images/任务编排.png)
![任务执行](./docs/images/任务执行.png)

### 会话与安全审计
![会话1](./docs/images/会话1.png)
![高危指令拦截](./docs/images/高危指令拦截.png)
![审计](./docs/images/审计.png)

### CMDB 资产管理
![资产台账](./docs/images/资产台账.png)
![新增资产](./docs/images/新增资产.png)
![资产拓扑](./docs/images/资产拓补.png)
![标签管理](./docs/images/标签管理.png)
![IP管理](./docs/images/IP管理.png)

### 监控告警
![实时监控面板](./docs/images/监控面板.png)
![采集目标](./docs/images/采集目标.png)
![告警规则](./docs/images/新增告警规则.png)
![通知渠道](./docs/images/告警渠道.png)

### 日志中心
![日志检索](./docs/images/日志检索.png)
![采集源](./docs/images/采集源.png)
![添加采集源](./docs/images/添加采集源.png)
![登录日志](./docs/images/登录日志.png)
![操作审计](./docs/images/操作审计.png)


---

## 快速开始

### 1. 克隆项目

```bash
# 克隆后端项目
git clone https://github.com/mona1isa/ops-go.git
cd ops-go

# 克隆前端项目（与后端同级目录）
git clone https://github.com/mona1isa/ops-go-ui.git
```

### 2. 数据库准备

```sql
-- 创建数据库
CREATE DATABASE ops_go DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;

-- 创建用户（可选）
CREATE USER 'ops_go'@'%' IDENTIFIED BY 'your_password';
GRANT ALL PRIVILEGES ON ops_go.* TO 'ops_go'@'%';
FLUSH PRIVILEGES;
```

### 3. 后端配置

```bash
# 复制环境变量配置文件
cp .env.example .env

# 编辑配置文件
vim .env
```

`.env` 配置说明：

```env
# 数据库配置
DB_DSN=ops_go:your_password@tcp(127.0.0.1:3306)/ops_go?charset=utf8mb4&parseTime=True&loc=Asia%2FShanghai

# Web 服务端口
APP_PORT=8080

# Redis 配置
REDIS_ADDRESS=127.0.0.1:6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_POOL_SIZE=10
REDIS_MIN_IDLE_CONN=5

# JWT 配置
JWT_ISSUER=https://ops-go.com
JWT_SECRET=your-jwt-secret-change-me

# 登录图形验证码（默认关闭，需显式开启）
CAPTCHA_ENABLED=false

# 会话录像保留天数（超期由后台清理协程删除）
RECORDING_RETENTION_DAYS=30

# 监控告警（可选，未配置时使用默认地址）
PROMETHEUS_URL=http://localhost:9090      # 面板查询与告警评估数据源
PUSHGATEWAY_URL=http://localhost:9091     # 采集结果写入端
# PROMETHEUS_TOKEN=                        # Prometheus 只读接口 Bearer Token（可选）
# PUSHGATEWAY_TOKEN=                       # Pushgateway Bearer Token（可选）
# PROMETHEUS_INSECURE_SKIP_VERIFY=false    # 自建证书场景跳过校验
# MONITOR_COLLECT_INTERVAL=60              # 采集轮询间隔（秒）
# MONITOR_COLLECT_TIMEOUT=15               # 单机采集超时（秒）
# MONITOR_CONCURRENCY=20                   # 采集并发上限
# MONITOR_EVAL_INTERVAL=30                 # 告警规则评估间隔（秒）
# MONITOR_PUSH_TIMEOUT=10                  # 写入 Pushgateway 超时（秒）
# MONITOR_QUERY_TIMEOUT=15                 # Prometheus 查询超时（秒）
# WEBHOOK_TIMEOUT=10                       # 通知渠道 HTTP 超时（秒）

# 日志中心（可选，未配置时使用默认地址）
LOKI_URL=http://localhost:3100            # 日志存储与检索引擎
# LOKI_TOKEN=                              # Loki Bearer Token（可选）
# LOKI_TIMEOUT=15                          # Loki 请求超时（秒）
# LOKI_INSECURE_SKIP_VERIFY=false          # 自建证书场景跳过校验
# LOG_COLLECT_INTERVAL=15                  # 日志采集轮询间隔（秒）
# LOG_COLLECT_CONCURRENCY=5                # 同时拉取的采集源上限
# LOG_COLLECT_TIMEOUT=20                   # 单源拉取超时（秒）
# LOG_COLLECT_MAX_BYTES=524288             # 单源单轮拉取字节上限
# LOG_COLLECT_INITIAL_BACK=65536           # 首轮无位点时的回溯字节数
# LOG_ALERT_INTERVAL=60                    # 日志告警规则评估间隔（秒）
```

### 4. 后端启动

```bash
# 安装依赖
go mod tidy

# 开发模式启动
go run main.go

# 或编译后启动
go build -o ops-go
./ops-go
```

启动成功后：
- Web API 服务运行在 `http://localhost:8080`
- SSH 堡垒机服务运行在 `ssh://localhost:2222`

同时会自动拉起以下后台协程与菜单种子（无需额外配置，未部署 Prometheus/Loki 时对应模块仅提示数据源不可用，不影响其他功能）：

| 后台任务 | 默认间隔 | 说明 |
|----------|----------|------|
| 主机健康检查 | - | 纳管主机在线状态探测 |
| 会话录像清理 | - | 按 `RECORDING_RETENTION_DAYS` 清理过期录像 |
| 主机指标采集 | 60s | SSH 下发采集脚本 → Pushgateway → Prometheus |
| 指标告警评估 | 30s | 规则评估、事件收敛与 Webhook 通知 |
| 日志采集 | 15s | SSH 增量拉取 → Loki，并发上限 5 |
| 日志告警评估 | 60s | 窗口条数阈值评估，事件与监控共享同一张表 |
| 菜单种子 | 启动时 | 自动写入监控告警、CMDB、日志中心三组菜单与权限（Casbin） |


### 5. 前端配置

```bash
# 进入前端目录
cd ../ops-go-ui

# 安装依赖
npm install
# 或使用 pnpm
pnpm install

# 开发模式启动（默认端口 1080）
npm run dev
```

### 6. 创建管理员账户

首次启动后，需要手动创建管理员账户：

```sql
-- 连接数据库
USE ops_go;

-- 创建管理员用户（密码: admin123）
INSERT INTO sys_user (user_name, nick_name, password, status, created_at, updated_at)
VALUES ('admin', '系统管理员', '$2a$10$N.zmdr9k7uOCQb376NoUnuTJ8iAt6Z5EHsM8lE9lBOsl7iAt6Z5EH', '1', NOW(), NOW());

-- 创建管理员角色
INSERT INTO sys_role (role_name, role_code, status, created_at, updated_at)
VALUES ('管理员', 'admin', '1', NOW(), NOW());

-- 关联用户角色
INSERT INTO sys_user_role (user_id, role_id, created_at)
SELECT u.id, r.id, NOW() FROM sys_user u, sys_role r WHERE u.user_name = 'admin' AND r.role_code = 'admin';
```

---

## 部署指南

### Docker 部署

```bash
# 构建后端镜像
docker build -t ops-go:latest .

# 运行容器
docker run -d \
  --name ops-go \
  -p 8080:8080 \
  -p 2222:2222 \
  -v /path/to/.env:/app/.env \
  -v /path/to/recordings:/app/recordings \
  ops-go:latest
```

### Docker Compose 部署

创建 `docker-compose.yml`：

```yaml
version: '3.8'

services:
  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: root_password
      MYSQL_DATABASE: ops_go
    volumes:
      - mysql_data:/var/lib/mysql
    ports:
      - "3306:3306"

  redis:
    image: redis:6-alpine
    ports:
      - "6379:6379"

  # 监控告警所需（可选，启用监控模块时保留）
  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus/prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus

  pushgateway:
    image: prom/pushgateway:latest
    ports:
      - "9091:9091"

  # 日志中心所需（可选，启用日志模块时保留）
  loki:
    image: grafana/loki:latest
    command: -config.file=/etc/loki/local-config.yaml
    ports:
      - "3100:3100"
    volumes:
      - loki_data:/loki


  backend:
    image: ops-go:latest
    depends_on:
      - mysql
      - redis
    ports:
      - "8080:8080"
      - "2222:2222"
    volumes:
      - ./.env:/app/.env
      - ./recordings:/app/recordings

  frontend:
    image: nginx:alpine
    ports:
      - "80:80"
    volumes:
      - ./ops-go-ui/dist:/usr/share/nginx/html
      - ./nginx.conf:/etc/nginx/nginx.conf

volumes:
  mysql_data:
  prometheus_data:
  loki_data:
```

启动：
```bash
docker-compose up -d
```

### 生产环境配置建议

1. **数据库**
   - 使用 MySQL 8.0+
   - 开启慢查询日志
   - 配置主从复制

2. **Redis**
   - 设置访问密码
   - 配置持久化（AOF/RDB）

3. **应用**
   - 使用 Nginx 反向代理
   - 配置 HTTPS
   - 启用 Gzip 压缩
   - 按需收紧跨域来源：`middleware/cors.go` 中 `AllowOrigins` 当前为 `*`，生产环境应改为具体域名列表

4. **安全**
   - 修改默认端口
   - 配置防火墙规则
   - 定期备份数据库
   - 定期更新依赖
   - `.env` 含数据库/Redis 口令与 `JWT_SECRET`，请勿提交到仓库（建议 `git rm --cached .env` 后仅保留 `.env.example`）
   - 凭证加密的 RSA 私钥目前硬编码在 `utils/rsa_util.go`，上线前应替换为独立密钥并改为外部注入

---

## 开发指南

### 项目结构

```
ops-go/
├── bastion/              # SSH 堡垒机模块
│   └── bastion.go        # SSH 服务器实现（登录失败封禁、地域策略、会话接管）
├── cmd/                  # 命令行工具
│   └── bastion/
│       └── hostkey.pem   # SSH 主机密钥
├── config/               # 配置文件（Redis/Casbin 模型与策略）
├── controllers/          # 控制器层
│   ├── instance/         # 实例相关接口（主机、凭证、终端、SFTP、脚本、任务、会话等）
│   │   └── api/          # 请求/响应 DTO
│   ├── monitor/          # 监控接口（采集目标、告警规则、事件、通知渠道、指标查询）
│   ├── logs/             # 日志中心接口（检索/直方图/上下文、采集源、日志告警、登录日志、操作审计）
│   ├── cmdb/             # CMDB 接口（资产、标签、关系、网段）
│   └── system/           # 系统相关接口（用户、角色、菜单、部门、日志等）
│       └── api/          # 请求/响应 DTO
├── data/                 # 内置数据文件（ip2region.xdb IP 归属地库）
├── middleware/           # 中间件
│   ├── auth.go           # JWT 认证
│   ├── casbin.go         # 权限校验
│   ├── cors.go           # 跨域处理
│   └── log.go            # 操作留痕（模块/类型/敏感标记/业务码/耗时）
├── models/               # 数据模型（GORM 实体，AutoMigrate 自动建表）
├── recordings/           # 会话录像存储（按日期归档）
├── routers/              # 路由配置（monitor_router / logs_router / cmdb_router 等）
├── services/             # 业务逻辑层
│   ├── instance/         # 实例服务（主机、SFTP、任务引擎、会话、健康检查、录像清理等）
│   ├── monitor/          # 监控服务（采集、Prometheus 数据源、规则评估、Webhook 通知）
│   │   ├── datasource/   # Prometheus / Pushgateway 客户端
│   │   └── webhook/      # 通知发送（签名、重试、落库）
│   ├── logs/             # 日志中心服务（采集源、检索、日志告警、登录日志、操作审计、菜单种子）
│   │   └── datasource/   # Loki 客户端、LogQL 构建、tail.sh 脚本与输出解析
│   ├── cmdb/             # CMDB 服务（资产、标签、关系拓扑、IPAM、菜单种子）
│   └── system/           # 系统服务（用户、角色、菜单、部门、日志等）
├── utils/                # 工具函数（RSA 加密、SFTP 客户端、ip2region 归属地查询等）
├── .env                  # 环境变量配置
├── go.mod                # Go 模块定义
├── go.sum                # 依赖校验
├── install.sh            # 一键安装脚本
└── main.go               # 程序入口（启动后台协程与菜单种子）
```

### 添加新功能模块

1. **创建数据模型** (`models/xxx.go`)
```go
type Xxx struct {
    ID   uint   `gorm:"primaryKey" json:"id"`
    Name string `gorm:"type:varchar(100)" json:"name"`
    models.Base
}
```

2. **创建服务层** (`services/xxx_service.go`)
```go
type XxxService struct{}

func (s *XxxService) GetList() ([]Xxx, error) {
    // 业务逻辑
}
```

3. **创建控制器** (`controllers/xxx_controller.go`)
```go
type XxxController struct{}

func (c *XxxController) List(ctx *gin.Context) {
    // 处理请求
}
```

4. **注册路由** (`routers/xxx_router.go`)
```go
func (r *XxxRouter) Setup(api *gin.RouterGroup) {
    group := api.Group("/xxx")
    {
        group.GET("/list", controller.List)
    }
}
```

### API 开发规范

1. **响应格式**
```json
{
  "code": 200,
  "msg": "success",
  "data": {}
}
```

2. **分页格式**
```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "data": [],
    "total": 100
  }
}
```

3. **错误处理**
```go
if err != nil {
    ctx.JSON(http.StatusOK, gin.H{
        "code": 500,
        "msg":  err.Error(),
    })
    return
}
```

---

## 系统访问

### Web 界面

| 地址 | 说明 |
|------|------|
| http://localhost:1080 | 前端开发地址 |
| http://localhost:8080 | 后端 API 地址 |

默认管理员账户：
- 用户名：`admin`
- 密码：`admin123`（需手动创建，参考上文）

### SSH 堡垒机

```bash
# 连接堡垒机
ssh -p 2222 admin@localhost

# 堡垒机命令
L        # 查看主机列表
R        # 刷新主机列表
<IP/ID>  # 连接指定主机
H        # 显示帮助
C        # 清屏
exit     # 退出
```

### API 接口

| 模块 | 路径 | 说明 |
|------|------|------|
| 用户 | `/api/user/*` | 用户管理 |
| 角色 | `/api/role/*` | 角色管理 |
| 菜单 | `/api/menu/*` | 菜单管理 |
| 部门 | `/api/dept/*` | 部门管理 |
| 主机 | `/api/instance/*` | 主机管理 |
| 凭证 | `/api/key/*` | 凭证管理 |
| 分组 | `/api/group/*` | 分组管理 |
| 授权 | `/api/auth/*` | 授权管理 |
| 会话 | `/api/session-record/*` | 会话审计 |
| 日志 | `/api/log/*` | 操作日志 |
| 活跃会话 | `/api/active-sessions/*` | 活跃会话管理 |
| 高危指令 | `/api/dangerous-command/*` | 高危指令规则管理 |
| 脚本 | `/api/script/*` | 脚本库管理 |
| 任务模板 | `/api/task-template/*` | 任务模板管理 |
| 任务编排 | `/api/task-pipeline/*` | 任务编排管理 |
| 任务执行 | `/api/task-execution/*` | 任务执行与记录 |
| 文件管理 | `/api/sftp/*` | SFTP 文件操作 |
| 验证码 | `/api/captcha/*` | 图形验证码 |
| 监控采集 | `/api/monitor/target/*` | 采集目标管理 |
| 告警规则 | `/api/monitor/rule/*` | 规则管理、静默、试算 |
| 告警事件 | `/api/monitor/event/*` | 事件查询与确认 |
| 通知渠道 | `/api/monitor/channel/*` | Webhook 渠道管理 |
| 指标查询 | `/api/monitor/metrics/*` | 总览/即时/区间查询 |
| 资产台账 | `/api/cmdb/asset/*` | 资产管理与生命周期流转 |
| 资产标签 | `/api/cmdb/tag/*` | 标签管理与打标 |
| 资产拓扑 | `/api/cmdb/relation/*` | 关联关系与拓扑图 |
| IP 管理 | `/api/cmdb/subnet/*` | 网段与 IP 占用 |
| 日志检索 | `/api/logs/query`、`/api/logs/histogram`、`/api/logs/context` | 日志查询、时间分布、上下文 |
| 日志标签 | `/api/logs/labels`、`/api/logs/label-values` | 检索条件下拉取值 |
| 日志数据源 | `/api/logs/status` | Loki 连通状态 |
| 日志采集源 | `/api/logs/source/*` | 采集源管理、手动拉取、位点重置 |
| 日志告警 | `/api/logs/alert/*` | 规则管理、静默、试算、告警事件 |
| 登录日志 | `/api/logs/login/*` | 登录日志与行为分析 |
| 操作审计 | `/api/logs/audit/*` | 操作留痕查询与审计分析 |

---

## 常见问题

### 1. 数据库连接失败
**问题**: `Error 1045 (28000): Access denied`

**解决方案**:
- 检查数据库用户名和密码
- 检查数据库用户权限
- 确认 `.env` 文件中的 `DB_DSN` 配置正确

### 2. Redis 连接失败
**问题**: `dial tcp 127.0.0.1:6379: connect: connection refused`

**解决方案**:
- 确认 Redis 服务已启动
- 检查 `REDIS_ADDRESS` 配置
- 如有密码，确认 `REDIS_PASSWORD` 配置

### 3. JWT Token 无效
**问题**: `token is expired` 或 `signature is invalid`

**解决方案**:
- 检查 `JWT_SECRET` 配置是否一致
- Token 有效期默认为 2 小时，请重新登录
- 确认系统时间正确

### 4. SSH 堡垒机无法连接
**问题**: SSH 连接被拒绝

**解决方案**:
- 确认堡垒机服务已启动（端口 2222）
- 检查防火墙是否放行 2222 端口
- 确认用户已创建且状态为启用

### 5. Web Terminal 无法连接
**问题**: WebSocket 连接失败

**解决方案**:
- 检查浏览器控制台错误信息
- 确认后端 WebSocket 服务正常
- 如使用 Nginx，需配置 WebSocket 代理：
```nginx
location /ws {
    proxy_pass http://backend;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
}
```

### 6. 终端内容不换行 / 行宽异常
**问题**: 输入长文本时不自动换行，在同一行重复覆盖显示

**解决方案**:
- 确认前后端均已部署最新版本（终端尺寸同步功能需要前后端同步更新）
- 检查浏览器 DevTools → Network → WS，确认 WebSocket 连接 URL 中包含 `cols` 和 `rows` 参数
- 确认 SSH 连接成功后前端发送了 `resize` 类型消息
- 如使用 Nginx 反向代理，确保 WebSocket 连接 URL 中的查询参数被正确传递

### 7. 录像回放失败
**问题**: 录像加载失败或播放异常

**解决方案**:
- 检查 `recordings` 目录权限
- 确认录像文件格式正确（asciinema v2）
- 查看浏览器控制台错误信息

### 8. 权限校验失败
**问题**: `Permission denied`

**解决方案**:
- 检查用户角色配置
- 确认 Casbin 规则已正确配置
- 查看 `casbin_rule` 表中的权限规则

### 9. 监控/日志页面提示数据源不可用
**问题**: 面板显示 Prometheus 不可用，或日志检索页顶部提示「日志数据源不可用」

**解决方案**:
- 确认 `PROMETHEUS_URL` / `PUSHGATEWAY_URL` / `LOKI_URL` 指向实际地址，且服务已启动
- 自建证书或走内网 HTTPS 时设置 `PROMETHEUS_INSECURE_SKIP_VERIFY=true` / `LOKI_INSECURE_SKIP_VERIFY=true`
- 需要鉴权时配置 `PROMETHEUS_TOKEN` / `PUSHGATEWAY_TOKEN` / `LOKI_TOKEN`
- 数据源不可用不影响主机、终端、任务等其他模块，采集协程会在恢复后自动继续

### 10. 日志采集不到数据
**问题**: 采集源显示已启用，但检索界面无数据

**解决方案**:
- 在「采集源」页点击「立即拉取」查看返回原因（文件不存在 / 不可读 / 超时）
- 读取 `/var/log/*` 通常需要 root 凭证或将采集账号加入 `adm` 组
- 首次拉取只回溯 `LOG_COLLECT_INITIAL_BACK`（默认 64KB）内容，历史日志较少时属正常
- 确认目标机与堡垒机之间 SSH 通道可用（同主机管理里的凭证与网络策略）
- 本机 Loki 会拒绝过旧的日志条目，采集端已做时间钳制重试，若仍失败请查看后端日志中的 `写入日志失败` 原因

---

## 性能优化

### 后端优化

1. **数据库**
   - 添加必要的索引
   - 使用连接池
   - 开启查询缓存

2. **Redis**
   - 缓存热点数据
   - 合理设置过期时间

3. **应用**
   - 使用 Goroutine 处理并发
   - 合理配置 GOMAXPROCS

### 前端优化

1. **构建优化**
   - 代码分割
   - Tree Shaking
   - Gzip 压缩

2. **运行时优化**
   - 组件懒加载
   - 虚拟滚动
   - 图片懒加载

---

## 安全建议

1. **密码安全**
   - 使用强密码
   - 定期更换密码
   - 启用密码复杂度校验

2. **网络安全**
   - 使用 HTTPS
   - 配置 CORS 白名单
   - 启用访问日志

3. **数据安全**
   - 敏感数据加密存储
   - 定期备份数据库
   - 配置访问控制

4. **会话安全**
   - 设置合理的 Token 过期时间
   - 登录失败次数限制
   - 敏感操作二次验证

---

## 贡献指南

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 提交 Pull Request

### 代码规范

- Go: 遵循 [Effective Go](https://golang.org/doc/effective_go)
- Vue: 遵循 [Vue Style Guide](https://vuejs.org/style-guide/)
- 提交信息遵循 [Conventional Commits](https://www.conventionalcommits.org/)

---

## 许可证

本项目基于 [MIT](LICENSE) 许可证开源。

---

## 联系方式

- Issues: [GitHub Issues](https://github.com/mona1isa/ops-go/issues)
- 作者: Zhany_v

---

## 致谢

感谢以下开源项目：

- [Gin](https://github.com/gin-gonic/gin)
- [GORM](https://gorm.io/)
- [Vue.js](https://vuejs.org/)
- [Element Plus](https://element-plus.org/)
- [xterm.js](https://xtermjs.org/)
- [asciinema-player](https://github.com/asciinema/asciinema-player)
