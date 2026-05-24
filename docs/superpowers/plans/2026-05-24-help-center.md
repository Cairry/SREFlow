# 帮助中心 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在导航栏新增 Help 菜单，点击进入 `/help` 页面，展示完整的帮助文档（快速入门、核心概念、节点参考）

**Architecture:** 单页面 + 左侧锚点导航布局。帮助内容以中文硬编码在 `web/lib/help-content.ts` 中。页面使用 IntersectionObserver 实现左侧导航高亮。导航栏新增 Help 入口。

**Tech Stack:** Next.js App Router, React, Tailwind CSS, Lucide Icons

---

### Task 1: 创建帮助内容数据文件

**Files:**
- Create: `web/lib/help-content.ts`

- [ ] **Step 1: 创建 help-content.ts**

创建 `web/lib/help-content.ts`，包含所有帮助内容数据：

```typescript
export interface ConfigField {
  name: string;
  type: string;
  required: boolean;
  description: string;
}

export interface NodeHelpContent {
  type: string;
  title: string;
  category: string;
  icon: string;
  description: string;
  configFields: ConfigField[];
  example: Record<string, unknown>;
}

export const NODE_HELP_CONTENT: NodeHelpContent[] = [
  // === Integration ===
  {
    type: "Prometheus",
    title: "Prometheus 指标查询",
    category: "Integration",
    icon: "activity",
    description: "通过 PromQL 查询 Prometheus 监控指标，获取时间序列数据。常用于查询服务的 CPU、内存、请求延迟等运行指标。",
    configFields: [
      { name: "prometheusUrl", type: "string", required: true, description: "Prometheus 服务地址，例如 http://prometheus:9090" },
      { name: "query", type: "string", required: true, description: "PromQL 查询语句，例如 up{job='my-service'}" },
      { name: "timeout", type: "number", required: false, description: "查询超时时间（秒），默认 30 秒" },
    ],
    example: {
      prometheusUrl: "http://prometheus:9090",
      query: "histogram_quantile(0.99, rate(http_request_duration_seconds_bucket{job='api-server'}[5m]))",
      timeout: 30,
    },
  },
  {
    type: "Jaeger",
    title: "Jaeger 链路追踪",
    category: "Integration",
    icon: "search",
    description: "查询 Jaeger 分布式链路追踪数据，用于分析请求在微服务间的调用链路和耗时分布。",
    configFields: [
      { name: "jaegerUrl", type: "string", required: true, description: "Jaeger 服务地址" },
      { name: "service", type: "string", required: false, description: "目标服务名称" },
      { name: "operation", type: "string", required: false, description: "操作名称" },
      { name: "traceId", type: "string", required: false, description: "指定的 Trace ID" },
      { name: "timeRange", type: "string", required: false, description: "时间范围" },
      { name: "limit", type: "number", required: false, description: "返回结果数量限制" },
    ],
    example: {
      jaegerUrl: "http://jaeger:16686",
      service: "api-server",
      operation: "GET /api/users",
      limit: 20,
    },
  },
  {
    type: "Elasticsearch",
    title: "Elasticsearch 日志搜索",
    category: "Integration",
    icon: "search",
    description: "查询 Elasticsearch 索引和文档，用于搜索和分析结构化日志数据。",
    configFields: [
      { name: "elasticsearchUrl", type: "string", required: true, description: "Elasticsearch 服务地址" },
      { name: "index", type: "string", required: true, description: "目标索引名称，例如 app-logs-*" },
      { name: "query", type: "string", required: true, description: "查询关键词或查询 DSL" },
      { name: "queryType", type: "select", required: true, description: "查询类型：match（全文匹配）、term（精确匹配）、range（范围查询）、bool（布尔组合）" },
      { name: "timeRange", type: "string", required: false, description: "时间范围" },
      { name: "size", type: "number", required: false, description: "返回文档数量限制" },
    ],
    example: {
      elasticsearchUrl: "http://elasticsearch:9200",
      index: "app-logs-*",
      query: "error timeout",
      queryType: "match",
      size: 50,
    },
  },
  {
    type: "Loki",
    title: "Loki 日志查询",
    category: "Integration",
    icon: "search",
    description: "使用 LogQL 查询 Loki 日志数据，适用于 Grafana Loki 日志聚合系统的日志检索场景。",
    configFields: [
      { name: "lokiUrl", type: "string", required: true, description: "Loki 服务地址" },
      { name: "query", type: "string", required: true, description: "LogQL 查询语句，例如 {app=\"my-service\"} |= \"error\"" },
      { name: "labels", type: "object", required: false, description: "标签过滤条件，键值对形式" },
      { name: "timeRange", type: "string", required: false, description: "时间范围" },
      { name: "limit", type: "number", required: false, description: "返回结果数量限制" },
    ],
    example: {
      lokiUrl: "http://loki:3100",
      query: '{app="api-server"} |= "timeout" | json | line_format "{{.message}}"',
      limit: 100,
    },
  },
  {
    type: "ClickHouse",
    title: "ClickHouse 数据查询",
    category: "Integration",
    icon: "bar-chart-3",
    description: "执行 ClickHouse SQL 查询，适用于大规模数据分析场景，可查询指标、日志等时序数据。",
    configFields: [
      { name: "clickhouseUrl", type: "string", required: true, description: "ClickHouse 服务地址" },
      { name: "database", type: "string", required: true, description: "目标数据库名称" },
      { name: "query", type: "string", required: true, description: "SQL 查询语句" },
      { name: "username", type: "string", required: false, description: "用户名" },
      { name: "password", type: "string", required: false, description: "密码" },
    ],
    example: {
      clickhouseUrl: "http://clickhouse:8123",
      database: "metrics",
      query: "SELECT toStartOfMinute(timestamp) AS minute, avg(value) FROM cpu_metrics WHERE timestamp > now() - INTERVAL 1 HOUR GROUP BY minute ORDER BY minute",
    },
  },
  {
    type: "Kubernetes",
    title: "Kubernetes 资源查询",
    category: "Integration",
    icon: "activity",
    description: "查询 Kubernetes 集群中 Pod 的运行状态和事件信息，用于排查容器编排相关问题。",
    configFields: [
      { name: "namespace", type: "string", required: false, description: "命名空间，不填则查询所有命名空间" },
      { name: "pod", type: "string", required: false, description: "Pod 名称，填 \"all\" 或留空查询所有 Pod" },
      { name: "resourceType", type: "select", required: false, description: "查询类型：status（Pod 状态）或 event（事件信息）" },
    ],
    example: {
      namespace: "production",
      pod: "api-server-7d9f8c6b5-x2k4j",
      resourceType: "status",
    },
  },
  // === Action ===
  {
    type: "SSH",
    title: "SSH 远程命令",
    category: "Action",
    icon: "terminal",
    description: "通过 SSH 连接到远程服务器执行命令，用于获取系统状态、排查问题等操作。需要预先在 Integrations 中配置 SSH 连接信息。",
    configFields: [
      { name: "host", type: "string", required: true, description: "目标主机地址" },
      { name: "port", type: "number", required: false, description: "SSH 端口，默认 22" },
      { name: "command", type: "string", required: true, description: "要执行的命令，例如 df -h、top -bn1 | head -20" },
      { name: "timeout", type: "number", required: false, description: "命令执行超时时间（秒）" },
    ],
    example: {
      host: "192.168.1.100",
      port: 22,
      command: "df -h && free -m && top -bn1 | head -20",
      timeout: 30,
    },
  },
  {
    type: "GoScript",
    title: "GoScript 脚本执行",
    category: "Action",
    icon: "code",
    description: "执行自定义 GoScript 代码，用于对上游节点输出进行自定义数据处理、格式转换、计算等操作。",
    configFields: [
      { name: "code", type: "string", required: true, description: "GoScript 代码，可使用上游节点的输出变量" },
      { name: "timeout", type: "number", required: false, description: "执行超时时间（秒）" },
    ],
    example: {
      code: `// 解析上游 Prometheus 数据
let result = JSON.parse(input.prometheus);
let maxVal = Math.max(...result.values.map(v => v[1]));
output = { max: maxVal, message: "最大值为 " + maxVal };`,
      timeout: 10,
    },
  },
  {
    type: "HTTP",
    title: "HTTP 请求",
    category: "Action",
    icon: "globe",
    description: "发送 HTTP 请求到外部接口，支持 GET、POST、PUT、PATCH 等方法，可用于调用内部 API、Webhook 通知等场景。",
    configFields: [
      { name: "url", type: "string", required: true, description: "请求 URL" },
      { name: "method", type: "select", required: true, description: "请求方法：GET、POST、PUT、PATCH" },
      { name: "headers", type: "object", required: false, description: "请求头，键值对形式" },
      { name: "body", type: "string", required: false, description: "请求体（JSON 字符串）" },
      { name: "timeout", type: "number", required: false, description: "请求超时时间（秒）" },
    ],
    example: {
      url: "https://api.example.com/alert",
      method: "POST",
      headers: { "Content-Type": "application/json", "Authorization": "Bearer token123" },
      body: '{"text": "服务异常告警", "severity": "critical"}',
      timeout: 15,
    },
  },
  // === AI ===
  {
    type: "LLM",
    title: "LLM 大语言模型",
    category: "AI",
    icon: "sparkles",
    description: "调用大语言模型进行文本生成、分析和推理。可结合上游节点数据，让 AI 自动分析告警原因、生成排查建议等。",
    configFields: [
      { name: "model", type: "string", required: true, description: "模型名称，例如 gpt-4、claude-3-sonnet" },
      { name: "prompt", type: "string", required: true, description: "用户提示词，可引用上游节点变量 {{nodeId.field}}" },
      { name: "systemPrompt", type: "string", required: false, description: "系统提示词，用于设定 AI 角色和行为" },
    ],
    example: {
      model: "gpt-4",
      systemPrompt: "你是一个 SRE 专家，擅长分析系统故障并提供排查建议。",
      prompt: "以下是 Prometheus 监控数据：{{prometheus_1.output}}\n\n请分析异常原因并给出排查建议。",
    },
  },
  // === Logic ===
  {
    type: "Condition",
    title: "Condition 条件判断",
    category: "Logic",
    icon: "git-branch",
    description: "根据条件判断结果控制工作流走向。支持字符串包含、等于、不等于、大于、小于、空值判断等操作符。条件为真时走左侧分支，为假时走右侧分支。",
    configFields: [
      { name: "condition", type: "string", required: true, description: "判断的变量值，可引用上游节点变量 {{nodeId.field}}" },
      { name: "operator", type: "select", required: true, description: "比较操作符：contains（包含）、equals（等于）、notEquals（不等于）、greaterThan（大于）、lessThan（小于）、isEmpty（为空）、isNotEmpty（不为空）" },
      { name: "value", type: "string", required: true, description: "比较的目标值" },
    ],
    example: {
      condition: "{{prometheus_1.output}}",
      operator: "greaterThan",
      value: "0.95",
    },
  },
  {
    type: "Start",
    title: "Start 起始节点",
    category: "Logic",
    icon: "play",
    description: "工作流的起始节点，每个工作流必须包含一个 Start 节点。可在起始节点中定义初始数据，作为后续节点的输入。",
    configFields: [
      { name: "text", type: "string", required: false, description: "初始文本内容，可传递给下游节点使用" },
    ],
    example: {
      text: "开始排查服务异常问题",
    },
  },
  // === Output ===
  {
    type: "Output",
    title: "Output 结果输出",
    category: "Output",
    icon: "file-output",
    description: "输出工作流执行结果。支持文件输出和 Webhook 输出两种方式。文件输出可将结果保存为 txt、md、json、csv、html、pdf 等格式；Webhook 输出可将结果发送到指定 URL。",
    configFields: [
      { name: "outputType", type: "select", required: true, description: "输出方式：file（文件）或 webhook（Webhook 回调）" },
      { name: "customFilename", type: "string", required: false, description: "自定义文件名（仅文件输出）" },
      { name: "fileType", type: "select", required: false, description: "文件格式：txt、md、json、csv、html、pdf（仅文件输出）" },
      { name: "autoDownload", type: "bool", required: false, description: "是否自动下载（仅文件输出）" },
      { name: "includeTimestamp", type: "bool", required: false, description: "文件名是否包含时间戳" },
      { name: "formatContent", type: "bool", required: false, description: "是否格式化输出内容" },
      { name: "webhook.url", type: "string", required: false, description: "Webhook URL（仅 Webhook 输出）" },
      { name: "webhook.method", type: "select", required: false, description: "Webhook 请求方法：GET、POST、PUT、PATCH" },
    ],
    example: {
      outputType: "file",
      fileType: "md",
      customFilename: "rca-report",
      includeTimestamp: true,
      formatContent: true,
    },
  },
  {
    type: "MCP",
    title: "MCP 工具",
    category: "Output",
    icon: "puzzle",
    description: "执行 MCP（Model Context Protocol）工具。需要先在 MCP Tools 页面配置 MCP 服务器，启用后即可在工作流中调用其提供的工具。",
    configFields: [
      { name: "serverId", type: "string", required: true, description: "MCP 服务器 ID" },
      { name: "serverName", type: "string", required: true, description: "MCP 服务器名称" },
      { name: "toolName", type: "string", required: true, description: "要调用的工具名称" },
      { name: "description", type: "string", required: false, description: "工具描述" },
      { name: "arguments", type: "object", required: false, description: "工具调用参数，键值对形式" },
    ],
    example: {
      serverId: "mcp-server-001",
      serverName: "运维工具集",
      toolName: "restart_service",
      arguments: { service_name: "api-server", namespace: "production" },
    },
  },
];

export const NODE_CATEGORIES = [
  { id: "Integration", label: "Integration 数据集成", nodes: NODE_HELP_CONTENT.filter(n => n.category === "Integration") },
  { id: "Action", label: "Action 执行操作", nodes: NODE_HELP_CONTENT.filter(n => n.category === "Action") },
  { id: "AI", label: "AI 智能分析", nodes: NODE_HELP_CONTENT.filter(n => n.category === "AI") },
  { id: "Logic", label: "Logic 流程控制", nodes: NODE_HELP_CONTENT.filter(n => n.category === "Logic") },
  { id: "Output", label: "Output 结果输出", nodes: NODE_HELP_CONTENT.filter(n => n.category === "Output") },
];
```

- [ ] **Step 2: 提交**

```bash
git add web/lib/help-content.ts
git commit -m "feat: add help center content data"
```

---

### Task 2: 创建帮助中心页面

**Files:**
- Create: `web/app/help/page.tsx`

- [ ] **Step 1: 创建 help page**

创建 `web/app/help/page.tsx`，实现左侧锚点导航 + 右侧内容的帮助中心页面：

```tsx
"use client";

import { useEffect, useState, useRef } from "react";
import {
  HelpCircle,
  Rocket,
  BookOpen,
  Activity,
  Search,
  BarChart3,
  Terminal,
  Code,
  Globe,
  Sparkles,
  GitBranch,
  FileOutput,
  Play,
  Puzzle,
  ChevronRight,
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { NODE_HELP_CONTENT, NODE_CATEGORIES } from "@/lib/help-content";
import type { NodeHelpContent } from "@/lib/help-content";

const ICON_MAP: Record<string, React.ReactNode> = {
  activity: <Activity className="w-5 h-5" />,
  search: <Search className="w-5 h-5" />,
  "bar-chart-3": <BarChart3 className="w-5 h-5" />,
  terminal: <Terminal className="w-5 h-5" />,
  code: <Code className="w-5 h-5" />,
  globe: <Globe className="w-5 h-5" />,
  sparkles: <Sparkles className="w-5 h-5" />,
  "git-branch": <GitBranch className="w-5 h-5" />,
  "file-output": <FileOutput className="w-5 h-5" />,
  play: <Play className="w-5 h-5" />,
  puzzle: <Puzzle className="w-5 h-5" />,
};

const SECTIONS = [
  { id: "quickstart", label: "快速入门", icon: <Rocket className="w-4 h-4" /> },
  { id: "concepts", label: "核心概念", icon: <BookOpen className="w-4 h-4" /> },
  ...NODE_CATEGORIES.map((cat) => ({
    id: `node-${cat.id.toLowerCase()}`,
    label: cat.label,
    icon: <HelpCircle className="w-4 h-4" />,
    subItems: cat.nodes.map((n) => ({
      id: `node-${n.type.toLowerCase()}`,
      label: n.title,
    })),
  })),
];

function NodeHelpCard({ node }: { node: NodeHelpContent }) {
  return (
    <Card id={`node-${node.type.toLowerCase()}`} className="scroll-mt-24">
      <CardHeader>
        <div className="flex items-center gap-3">
          <div className="p-2 rounded-lg bg-primary/10 text-primary">
            {ICON_MAP[node.icon] || <HelpCircle className="w-5 h-5" />}
          </div>
          <div>
            <CardTitle className="text-lg">{node.title}</CardTitle>
            <Badge variant="outline" className="mt-1 text-xs">
              {node.category}
            </Badge>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-sm text-muted-foreground leading-relaxed">{node.description}</p>

        <div>
          <h4 className="text-sm font-semibold mb-2">配置参数</h4>
          <div className="rounded-lg border overflow-hidden">
            <table className="w-full text-sm">
              <thead>
                <tr className="bg-muted/50">
                  <th className="px-3 py-2 text-left font-medium">参数名</th>
                  <th className="px-3 py-2 text-left font-medium">类型</th>
                  <th className="px-3 py-2 text-left font-medium">必填</th>
                  <th className="px-3 py-2 text-left font-medium">说明</th>
                </tr>
              </thead>
              <tbody>
                {node.configFields.map((field) => (
                  <tr key={field.name} className="border-t">
                    <td className="px-3 py-2 font-mono text-xs">{field.name}</td>
                    <td className="px-3 py-2">
                      <Badge variant="secondary" className="text-xs">{field.type}</Badge>
                    </td>
                    <td className="px-3 py-2">
                      {field.required ? (
                        <Badge variant="destructive" className="text-xs">必填</Badge>
                      ) : (
                        <Badge variant="outline" className="text-xs">可选</Badge>
                      )}
                    </td>
                    <td className="px-3 py-2 text-muted-foreground">{field.description}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        <div>
          <h4 className="text-sm font-semibold mb-2">配置样例</h4>
          <pre className="rounded-lg bg-muted/50 p-4 text-xs font-mono overflow-x-auto">
            {JSON.stringify(node.example, null, 2)}
          </pre>
        </div>
      </CardContent>
    </Card>
  );
}

export default function HelpPage() {
  const [activeSection, setActiveSection] = useState("quickstart");
  const contentRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const headings = contentRef.current?.querySelectorAll("[id]");
    if (!headings?.length) return;

    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) {
            setActiveSection(entry.target.id);
          }
        }
      },
      { rootMargin: "-80px 0px -60% 0px" }
    );

    headings.forEach((h) => observer.observe(h));
    return () => observer.disconnect();
  }, []);

  const scrollToSection = (id: string) => {
    const el = document.getElementById(id);
    if (el) {
      el.scrollIntoView({ behavior: "smooth" });
      setActiveSection(id);
    }
  };

  return (
    <div className="min-h-screen bg-background">
      <div className="max-w-7xl mx-auto px-4 py-8">
        <div className="flex gap-8">
          {/* Left nav */}
          <nav className="w-56 shrink-0 hidden lg:block">
            <div className="sticky top-24 space-y-1">
              <h3 className="px-3 mb-3 text-sm font-semibold text-foreground">帮助中心</h3>
              {SECTIONS.map((section) => (
                <div key={section.id}>
                  <button
                    type="button"
                    onClick={() => scrollToSection(section.id)}
                    className={`w-full flex items-center gap-2 px-3 py-1.5 rounded-md text-sm transition-colors text-left ${
                      activeSection === section.id
                        ? "bg-primary/10 text-primary font-medium"
                        : "text-muted-foreground hover:text-foreground hover:bg-muted/50"
                    }`}
                  >
                    {section.icon}
                    <span className="truncate">{section.label}</span>
                  </button>
                  {section.subItems?.map((sub) => (
                    <button
                      key={sub.id}
                      type="button"
                      onClick={() => scrollToSection(sub.id)}
                      className={`w-full flex items-center gap-2 pl-8 pr-3 py-1 rounded-md text-xs transition-colors text-left ${
                        activeSection === sub.id
                          ? "bg-primary/10 text-primary font-medium"
                          : "text-muted-foreground hover:text-foreground hover:bg-muted/50"
                      }`}
                    >
                      <ChevronRight className="w-3 h-3" />
                      <span className="truncate">{sub.label}</span>
                    </button>
                  ))}
                </div>
              ))}
            </div>
          </nav>

          {/* Right content */}
          <div ref={contentRef} className="flex-1 min-w-0 space-y-8">
            {/* Quick Start */}
            <section id="quickstart" className="scroll-mt-24">
              <h2 className="text-2xl font-bold mb-4">快速入门</h2>
              <div className="space-y-4">
                <Card>
                  <CardContent className="pt-6">
                    <h3 className="font-semibold mb-3">1. 创建工作区</h3>
                    <p className="text-sm text-muted-foreground">
                      在导航栏的工作区选择器中，点击 &quot;Create First Workspace&quot; 创建你的第一个工作区。工作区用于组织和管理你的工作流和数据集成。
                    </p>
                  </CardContent>
                </Card>
                <Card>
                  <CardContent className="pt-6">
                    <h3 className="font-semibold mb-3">2. 配置集成</h3>
                    <p className="text-sm text-muted-foreground">
                      在 Integrations 页面配置你需要的数据源连接，如 Prometheus、Elasticsearch、SSH 等。配置完成后，这些连接即可在工作流中使用。
                    </p>
                  </CardContent>
                </Card>
                <Card>
                  <CardContent className="pt-6">
                    <h3 className="font-semibold mb-3">3. 创建工作流</h3>
                    <p className="text-sm text-muted-foreground">
                      在 Workflows 页面点击创建新工作流，通过拖拽节点到画布来编排自动化流程。从 Start 节点开始，按需添加数据查询、AI 分析、条件判断等节点，最后以 Output 节点输出结果。
                    </p>
                  </CardContent>
                </Card>
                <Card>
                  <CardContent className="pt-6">
                    <h3 className="font-semibold mb-3">4. 运行和调试</h3>
                    <p className="text-sm text-muted-foreground">
                      点击工具栏的运行按钮执行工作流。运行过程中可以实时查看每个节点的执行状态和输出结果。如果某个节点失败，可以查看错误信息并调整配置后重新运行。
                    </p>
                  </CardContent>
                </Card>
              </div>
            </section>

            {/* Core Concepts */}
            <section id="concepts" className="scroll-mt-24">
              <h2 className="text-2xl font-bold mb-4">核心概念</h2>
              <div className="space-y-4">
                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">Workspace 工作区</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <p className="text-sm text-muted-foreground">
                      工作区是组织和管理资源的基本单元。每个工作区拥有独立的工作流、集成配置和 MCP 工具。你可以在导航栏切换工作区，不同工作区之间的数据互不影响。
                    </p>
                  </CardContent>
                </Card>
                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">Workflow 工作流</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <p className="text-sm text-muted-foreground">
                      工作流是由多个节点按逻辑连接而成的自动化流程。每个工作流从 Start 节点开始，经过数据查询、AI 分析、条件判断等处理步骤，最终通过 Output 节点输出结果。节点之间通过连线传递数据，上游节点的输出可作为下游节点的输入。
                    </p>
                  </CardContent>
                </Card>
                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">Integrations 集成</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <p className="text-sm text-muted-foreground">
                      集成管理外部系统的连接配置，包括 Prometheus、Elasticsearch、SSH 等数据源和服务的连接信息。配置好的集成可在工作流的对应节点中直接引用，无需重复填写连接参数。
                    </p>
                  </CardContent>
                </Card>
                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">MCP Tools MCP 工具</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <p className="text-sm text-muted-foreground">
                      MCP（Model Context Protocol）工具是对外部工具服务的标准化接入。在 MCP Tools 页面配置 MCP 服务器后，启用其提供的工具即可在工作流中使用 MCP 节点调用这些工具，扩展工作流的能力边界。
                    </p>
                  </CardContent>
                </Card>
              </div>
            </section>

            {/* Node reference sections */}
            {NODE_CATEGORIES.map((cat) => (
              <section key={cat.id} id={`node-${cat.id.toLowerCase()}`} className="scroll-mt-24">
                <h2 className="text-2xl font-bold mb-4">{cat.label}</h2>
                <div className="space-y-6">
                  {cat.nodes.map((node) => (
                    <NodeHelpCard key={node.type} node={node} />
                  ))}
                </div>
              </section>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: 提交**

```bash
git add web/app/help/page.tsx
git commit -m "feat: add help center page with navigation and content"
```

---

### Task 3: 在导航栏添加 Help 菜单入口

**Files:**
- Modify: `web/components/navigation.tsx:8-33`

- [ ] **Step 1: 修改 navigation.tsx**

在 `navigation` 数组中添加 Help 项，并在 import 中添加 `HelpCircle` 图标。

修改 import 行（第 8 行），添加 `HelpCircle`：

```typescript
import { Workflow, Zap, Sun, Moon, Monitor, Plus, LayoutGrid, Trash2, Edit2, X, Database, ArrowLeft, Puzzle, HelpCircle } from "lucide-react";
```

修改 `navigation` 数组（第 29-33 行），添加 Help 项：

```typescript
const navigation = [
  { name: "Workflows", href: "/workflows", icon: Workflow },
  { name: "Integrations", href: "/integrations", icon: Database },
  { name: "MCP Tools", href: "/mcp-tools", icon: Puzzle },
  { name: "Help", href: "/help", icon: HelpCircle },
];
```

修改 workspace selector 显示条件（第 241 行），将 Help 页面排除在工作区选择器之外：

```typescript
{!loading && !isWorkflowEditor && (pathname === "/workflows" || pathname === "/integrations" || pathname === "/mcp-tools") && (
```

改为：

```typescript
{!loading && !isWorkflowEditor && (pathname === "/workflows" || pathname === "/integrations" || pathname === "/mcp-tools" || pathname === "/help") && (
```

- [ ] **Step 2: 提交**

```bash
git add web/components/navigation.tsx
git commit -m "feat: add Help menu item to navigation bar"
```

---

### Task 4: 验证

- [ ] **Step 1: 启动开发服务器并验证**

```bash
cd /Users/zuxuan/workspace/github/RCAgent/web && npm run dev
```

验证：
1. 导航栏出现 Help 菜单项
2. 点击 Help 进入 `/help` 页面
3. 左侧导航显示所有章节，点击可滚动到对应位置
4. 左侧导航根据滚动位置自动高亮
5. 快速入门、核心概念内容正确
6. 所有 14 个节点的功能介绍、参数表格、配置样例显示正确
7. 页面在深色/浅色主题下均可正常显示

- [ ] **Step 2: 最终提交**

如有必要修复后提交所有更改。
