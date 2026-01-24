# ops-go

## 项目简介

ops-go是一个基于Go语言开发的运维管理系统，提供了用户管理、角色管理、菜单管理、部门管理、实例管理、密钥管理等功能。

## 技术栈

- 后端：Go + Gin + GORM + MySQL + Redis + Casbin
- 前端：未提供，请自行准备前端项目

## 环境要求

- Go 1.24+
- MySQL 5.7+
- Redis 5.0+

## 快速开始

### 1. 配置环境变量

复制`.env.example`文件为`.env`，并修改相应的配置：

```bash
cp .env.example .env
```

### 2. 安装依赖

```bash
go mod tidy
```

### 3. 启动服务

```bash
go run main.go
```

服务将在`http://localhost:8080`启动，SSH堡垒机服务将在`2222`端口启动。

## 项目结构

```
├── bastion/          # SSH堡垒机模块
├── cmd/              # 命令行工具
├── config/           # 配置文件
├── controllers/      # 控制器
├── middleware/       # 中间件
├── models/           # 数据模型
├── routers/          # 路由配置
├── services/         # 业务逻辑
├── utils/            # 工具函数
├── .env              # 环境变量
├── go.mod            # Go模块依赖
├── go.sum            # Go模块依赖校验
└── main.go           # 项目入口
```

## API文档

启动服务后，可以通过`http://localhost:8080/api/`访问各个API接口。

## 注意事项

1. 首次启动会自动创建数据库表结构
2. 默认没有创建任何用户，需要手动在数据库中插入
3. SSH堡垒机使用`hostkey.pem`作为主机密钥

## 许可证

MIT