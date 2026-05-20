# 💎 Zeus — Golang API 应用脚手架
`Zeus`是一个基于`Golang`的`API`应用脚手架, `Zeus`的含义`希腊众神之王，寓意核心平台`, 简洁分层的目录结构, 以帮助你快速构建一个高效、可靠的应用程序。

## 📌 目录结构
```
SREFlow/
├── api/                # API 层，接口定义与实现
│   ├── service.go
│   └── user.go
├── cmd/                # 程序入口
│   └── main.go
├── config/             # 配置文件
│   ├── config.go
│   └── config.yaml
├── internal/           # 内部应用逻辑
│   ├── cache/          # 缓存相关
│   │   ├── entry.go
│   │   └── user.go
│   ├── ctx/            # 上下文管理
│   │   └── ctx.go
│   ├── middleware/     # 中间件
│   │   ├── core.go
│   │   └── logger.go
│   ├── models/         # 数据模型
│   │   └── user.go
│   ├── repos/          # 数据仓库
│   │   ├── entry.go
│   │   └── user.go
│   ├── routes/         # 路由
│   │   └── v1.go
│   ├── services/       # 业务逻辑
│   │   ├── entry.go
│   │   └── user.go
│   └── types/          # 类型定义
│       └── user.go
├── pkg/                # 公共库
│   ├── client/         # 客户端工具
│   │   ├── db.go
│   │   └── redis.go
│   ├── response/       # 响应封装
│   │   └── res.go
│   └── tools/          # 工具类
│       └── jwt.go
├── Dockerfile          # Docker 构建文件
├── go.mod              # Go 依赖管理
├── go.sum              # Go 依赖校验
└── README.md           # 项目说明文档
```

---

## 📦️ 基础组件
- **Gin**: https://github.com/gin-gonic/gin
- **Gorm**: https://github.com/go-gorm/gorm
- **Viper**: https://github.com/spf13/viper
- **Golang-jwt**: https://github.com/golang-jwt/jwt
- **Go-redis**: https://github.com/go-redis/redis
- **More...**

---

## 🔍️ 环境要求
- Go 1.23 及以上
- MySQL 5.7+/8.0+
- Redis 6+

---

## ✅ 安装
1. 克隆项目：
   ```bash
   git clone https://github.com/你的用户名/Zeus.git
   cd Zeus
   ```
2. 安装依赖：
   ```bash
   go mod tidy
   ```
   
---

## 🔧 配置
请根据实际情况修改 `config/config.yaml`，如数据库、Redis、服务端口等。

---

## 💡 启动项目
> 本地启动
```bash
go run cmd/main.go
```
---

如有问题欢迎提 Issue 或 PR！