# SREFlow

> AI 驱动的 SRE 工作流自动化平台 — 将告警响应、故障诊断与根因分析编排为可复用的 Workflow。

## ✨ 核心特性

- **可视化 Workflow 引擎** — 基于 DAG 的层级执行模型，支持并发节点、条件分支、变量传递与重试策略
- **丰富的内置节点** — 覆盖数据采集、AI 分析、逻辑处理与输出四大类别（见下方节点列表）
- **MCP Server** — 内置 Model Context Protocol 服务端，可被 LLM Agent 直接调用编排工作流
- **多租户 Workspace** — 工作空间级别的资源隔离，集成与工作流按空间管理
- **拓扑管理** — 可视化定义服务拓扑关系，供 Workflow 节点引用
- **多数据源集成** — 开箱即用的 Prometheus / Jaeger / Loki / Elasticsearch / ClickHouse / Kubernetes / SSH 连接管理
- **Web 控制台** — Next.js + React Flow 前端，拖拽式工作流编排与实时执行监控

## 🧩 内置节点类型

| 类别 | 节点 | 说明 |
|------|------|------|
| 📥 数据采集 | `prometheus` | PromQL 查询 |
| | `jaeger` | Jaeger 链路追踪查询 |
| | `loki` | LogQL 日志查询 |
| | `elasticsearch` | Elasticsearch 全文检索 |
| | `clickhouse` | ClickHouse SQL 查询 |
| | `kubernetes` | K8s 资源查询 (Pods / Events / Logs) |
| | `ssh` | 远程命令执行 |
| | `topology` | 读取拓扑关系 |
| 🧠 处理分析 | `llm` | 大语言模型推理 (支持多模型) |
| | `condition` | 条件分支路由 |
| | `goscript` | Go 脚本执行 (yaegi 解释器) |
| | `mcptool` | MCP 工具调用 |
| 📤 输入输出 | `start` | 工作流入口 |
| | `output` | 工作流结果输出 |
| | `http` | HTTP 请求节点 |

## 🏗️ 架构概览

```
┌──────────────┐      ┌──────────────┐      ┌───────────────────┐
│  Web Console │ ───▶ │  API Server  │ ───▶ │  Workflow Engine   │
│  (Next.js)   │      │  (Gin)       │      │  (DAG Executor)    │
└──────────────┘      └──────┬───────┘      └────────┬──────────┘
                             │                        │
                     ┌───────┴───────┐        ┌──────┴──────┐
                     │  Integrations  │        │  MCP Server  │
                     │  (DB / Redis)  │        │  (SSE/HTTP)  │
                     └───────────────┘        └─────────────┘
```

## 📁 目录结构

```
SREFlow/
├── api/                    # API 层 — 路由处理与请求绑定
│   ├── api.go              #   统一响应封装
│   ├── workflow.go         #   工作流 CRUD 与执行 API
│   ├── topology.go         #   拓扑管理 API
│   ├── workspace.go        #   工作空间 API
│   ├── integration.go      #   集成管理 API
│   └── mcp.go              #   MCP Server 管理 API
├── cmd/
│   ├── server.go           # API Server 入口
│   └── mcp/main.go         # MCP Server 入口
├── config/
│   ├── config.go           # 配置加载
│   └── config.yaml         # 配置文件
├── internal/
│   ├── cache/              # 缓存层
│   ├── ctx/                # 上下文管理
│   ├── middleware/          # 中间件 (CORS / Auth)
│   ├── models/             # 数据模型
│   ├── repos/              # 数据仓库层
│   ├── routes/             # 路由注册
│   ├── services/           # 业务逻辑层
│   ├── types/              # 请求/响应类型定义
│   └── workflow/           # 工作流引擎核心
│       ├── engine.go       #   引擎主入口
│       ├── executor.go     #   DAG 层级执行器
│       ├── state_manager.go#   执行状态管理
│       ├── monitor.go      #   执行监控
│       ├── retry.go        #   重试策略
│       ├── variable_resolver.go # 变量解析
│       ├── workflow_parser.go   # 工作流解析
│       ├── node_registry.go     # 节点注册器
│       ├── node_registry_bootstrap.go # 内置节点注册
│       └── nodes/          #   内置节点实现
│           ├── prometheus.go
│           ├── jaeger.go
│           ├── loki.go
│           ├── elasticsearch.go
│           ├── clickhouse.go
│           ├── kubernetes.go
│           ├── ssh.go
│           ├── llm.go
│           ├── condition.go
│           ├── goscript.go
│           ├── mcptool.go
│           └── ...
├── pkg/
│   ├── client/             # DB / Redis 客户端
│   └── tools/              # JWT / 命令行工具
├── web/                    # 前端 (Next.js + React Flow)
├── data/                   # SQLite 数据存储
├── Dockerfile              # Docker 构建
├── Dockerfile-otel         # OpenTelemetry 构建
├── mcp_config.json         # MCP Server 示例配置
├── go.mod
└── go.sum
```

## 🛠️ 技术栈

- **后端**: Go 1.23+ / Gin / GORM / Viper / JWT / go-redis
- **工作流引擎**: 自研 DAG 执行器 + yaegi Go 脚本解释器
- **Kubernetes**: client-go
- **MCP**: mark3labs/mcp-go
- **前端**: Next.js 16 / React 19 / TailwindCSS / Radix UI / Monaco Editor
- **数据库**: MySQL 5.7+ 或 SQLite（默认）
- **缓存**: Redis 6+

## 🚀 快速开始

### 环境要求

- Go 1.23+
- Node.js 18+ & pnpm
- MySQL 5.7+ / 8.0+（或使用内置 SQLite）
- Redis 6+

### 后端

```bash
# 克隆项目
git clone https://github.com/your-username/SREFlow.git
cd SREFlow

# 安装依赖
go mod tidy

# 修改配置 (数据库、Redis、端口等)
vim config/config.yaml

# 启动 API Server
go run cmd/server.go
```

### MCP Server

```bash
# 启动 MCP Server (StreamableHTTP 模式)
go run cmd/mcp/main.go -transport streamable-http -addr :8088

# 或 SSE 模式
go run cmd/mcp/main.go -transport sse -addr :8088
```

MCP Server 暴露以下工具供 LLM Agent 调用：

| 工具名 | 说明 |
|--------|------|
| `workflow_list` | 列出工作空间下所有工作流 |
| `workflow_execute` | 执行指定工作流 |
| `workflow_status` | 查询执行状态 |
| `workflow_result` | 获取执行结果 |

### 前端

```bash
cd web

# 安装依赖
pnpm install

# 启动开发服务器
pnpm dev
```

## 🐳 Docker 部署

```bash
# 标准构建
docker build -f Dockerfile -t sreflow .

# OpenTelemetry 构建
docker build -f Dockerfile-otel -t sreflow-otel .
```

## 🔧 配置说明

配置文件位于 `config/config.yaml`，主要配置项：

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| `Server.port` | API 服务端口 | `9001` |
| `Server.mode` | 运行模式 (release / debug / test) | `release` |
| `Database.type` | 数据库类型 (mysql / sqlite) | `sqlite` |
| `Database.path` | SQLite 数据库路径 | `data/rcaagent.db` |
| `Redis.host` | Redis 地址 | `w8t-redis` |
| `Jwt.expire` | Token 过期时间 (秒) | `18000` |

MCP Server 外部集成配置见 `mcp_config.json`。

## 📡 API 概览

所有 API 以 `/api` 为前缀，通过 `X-Workspace-Id` Header 做工作空间隔离。

| 模块 | 路径前缀 | 说明 |
|------|----------|------|
| Workflows | `/api/workflows` | 工作流 CRUD + 执行 + 运行历史 |
| Workspaces | `/api/workspaces` | 工作空间管理 |
| Integrations | `/api/integrations` | 数据源集成 (含 K8s 连通性测试) |
| Topology | `/api/topology` | 拓扑管理 |
| MCP Servers | `/api/mcp-servers` | MCP Server 管理 + 工具列表 + 状态 |

### 执行工作流

```
POST /api/workflow/:id/execute?mode={json|text}
```

**Query 参数：**

| 参数 | 必填 | 说明 |
|------|------|------|
| `mode` | 否 | 执行模式，默认 `text`。`text`：从 body 的 `input` 字段读取内容输出到 Start 节点；`json`：将整个 body 内容直接作为 Start 节点的输出 |

**mode=text（默认）：**

从 body 中读取 `input` 字段，将 `input` 内容传递给 Start 节点。适用于传递简单的文本或预格式化字符串。

```bash
curl -X POST 'http://localhost:9001/api/workflow/{id}/execute?mode=text' \
  -H 'Content-Type: application/json' \
  -d '{"input": "hello world"}'
```

Start 节点 output: `{"text": "hello world"}`

**mode=json：**

将整个 body 作为 JSON 直接输出到 Start 节点。适用于需要传递结构化数据（如告警信息）的场景，下游节点可通过 `start.fieldName` 引用各字段。

```bash
curl -X POST 'http://localhost:9001/api/workflow/{id}/execute?mode=json' \
  -H 'Content-Type: application/json' \
  -d '{"alert": "CPU > 90%", "host": "node-1", "severity": "critical"}'
```

Start 节点 output: `{"alert": "CPU > 90%", "host": "node-1", "severity": "critical"}`

## 🤝 参与贡献

欢迎提交 Issue 和 Pull Request！
