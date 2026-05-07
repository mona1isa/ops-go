# Ops-Go 运维管理系统

## 项目简介

Ops-Go 是一个基于 Go 语言开发的企业级运维管理系统，提供主机管理、SSH 堡垒机、Web Terminal、会话录像与审计等功能。系统采用前后端分离架构，支持多种认证方式和细粒度的权限控制。

### 核心特性

- 🖥️ **Web Terminal** - 浏览器直接连接 SSH，支持 xterm.js 全功能终端
- 🔐 **SSH 堡垒机** - 独立的 SSH 服务（端口 2222），支持命令行交互式操作
- 📹 **会话录像** - 自动录制所有 SSH 会话，支持在线回放和下载
- 👥 **权限管理** - 基于 Casbin 的 RBAC 权限控制，支持细粒度资源授权
- 🔑 **凭证管理** - 支持密码和密钥两种认证方式，凭证加密存储
- 📊 **审计日志** - 完整的操作日志记录，支持日志查询和导出

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
- 主机状态管理
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

### 12. 系统监控
- 在线用户统计
- 系统资源监控
- 会话统计

---

## 未来规划

### 短期规划 (v1.1)
- [ ] 批量主机导入（Excel/CSV）
- [ ] 主机状态自动检测
- [x] 文件上传下载（SFTP）
- [ ] 录像自动清理策略
- [x] 危险命令拦截

### 中期规划 (v1.2)
- [ ] RDP/VNC 协议支持
- [ ] 多因素认证（MFA）
- [ ] 操作审批流程
- [ ] 命令审计规则
- [ ] 录像对象存储（OSS/S3）

### 长期规划 (v2.0)
- [ ] Kubernetes 集群管理
- [ ] 自动化运维（Ansible 集成）
- [ ] 监控告警集成
- [ ] 多租户支持
- [ ] 高可用部署

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

### 操作系统支持
- Linux (推荐 CentOS 7+, Ubuntu 18.04+)
- macOS
- Windows (开发测试)

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

# CORS 配置
CORS_ORIGINS=http://localhost:3000,http://localhost:8080

# RSA 私钥（可选，用于凭证加密）
# RSA_PRIVATE_KEY=-----BEGIN RSA PRIVATE KEY-----
# ...
# -----END RSA PRIVATE KEY-----
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

### 5. 前端配置

```bash
# 进入前端目录
cd ../ops-go-ui

# 安装依赖
npm install
# 或使用 pnpm
pnpm install

# 开发模式启动
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
   - 设置合理的 CORS 域名

4. **安全**
   - 修改默认端口
   - 配置防火墙规则
   - 定期备份数据库
   - 定期更新依赖

---

## 开发指南

### 项目结构

```
ops-go/
├── bastion/              # SSH 堡垒机模块
│   └── bastion.go        # SSH 服务器实现
├── cmd/                  # 命令行工具
│   └── bastion/
│       └── hostkey.pem   # SSH 主机密钥
├── config/               # 配置文件
├── controllers/          # 控制器层
│   ├── instance/         # 实例相关接口
│   └── system/           # 系统相关接口
├── docs/                 # 文档
├── middleware/           # 中间件
│   ├── auth.go           # JWT 认证
│   ├── casbin.go         # 权限校验
│   ├── cors.go           # 跨域处理
│   └── log.go            # 日志记录
├── models/               # 数据模型
├── recordings/           # 会话录像存储
├── routers/              # 路由配置
├── services/             # 业务逻辑层
│   ├── instance/         # 实例服务
│   └── system/           # 系统服务
├── utils/                # 工具函数
├── .env.example          # 环境变量示例
├── go.mod                # Go 模块定义
├── go.sum                # 依赖校验
└── main.go               # 程序入口
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
| http://localhost:3000 | 前端开发地址 |
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

### 6. 录像回放失败
**问题**: 录像加载失败或播放异常

**解决方案**:
- 检查 `recordings` 目录权限
- 确认录像文件格式正确（asciinema v2）
- 查看浏览器控制台错误信息

### 7. 权限校验失败
**问题**: `Permission denied`

**解决方案**:
- 检查用户角色配置
- 确认 Casbin 规则已正确配置
- 查看 `casbin_rule` 表中的权限规则

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
