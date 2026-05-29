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
  example: { label: string; value: string; isCode?: boolean }[];
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
      { name: "query", type: "string", required: true, description: "PromQL 查询语句，例如 up{job='my-service'}" },
      { name: "timeout", type: "number", required: false, description: "查询超时时间（秒），默认 30 秒" },
    ],
    example: [
      { label: "query", value: "histogram_quantile(0.99, rate(http_request_duration_seconds_bucket{job='api-server'}[5m]))" },
      { label: "timeout", value: "30" },
    ],
  },
  {
    type: "Jaeger",
    title: "Jaeger 链路追踪",
    category: "Integration",
    icon: "search",
    description: "查询 Jaeger 分布式链路追踪数据，用于分析请求在微服务间的调用链路和耗时分布。可通过 Tags 精确过滤追踪数据，例如按 HTTP 状态码、错误类型等筛选。",
    configFields: [
      { name: "service", type: "string", required: false, description: "目标服务名称" },
      { name: "tags", type: "string", required: false, description: "搜索标签，例如 http.status_code=5.* 或 error=true" },
      { name: "timeRange", type: "string", required: false, description: "时间范围" },
      { name: "limit", type: "number", required: false, description: "返回结果数量限制" },
    ],
    example: [
      { label: "service", value: "api-server" },
      { label: "tags", value: "http.status_code=5.*" },
      { label: "limit", value: "20" },
    ],
  },
  {
    type: "Elasticsearch",
    title: "Elasticsearch 日志搜索",
    category: "Integration",
    icon: "search",
    description: "查询 Elasticsearch 索引和文档，用于搜索和分析结构化日志数据。",
    configFields: [
      { name: "index", type: "string", required: true, description: "目标索引名称，例如 app-logs-*" },
      { name: "query", type: "string", required: true, description: "查询关键词或查询 DSL" },
      { name: "queryType", type: "select", required: true, description: "查询类型：match（全文匹配）、term（精确匹配）、range（范围查询）、bool（布尔组合）" },
      { name: "timeRange", type: "string", required: false, description: "时间范围" },
      { name: "size", type: "number", required: false, description: "返回文档数量限制" },
    ],
    example: [
      { label: "index", value: "app-logs-*" },
      { label: "query", value: "error timeout" },
      { label: "queryType", value: "match" },
      { label: "size", value: "50" },
    ],
  },
  {
    type: "Loki",
    title: "Loki 日志查询",
    category: "Integration",
    icon: "search",
    description: "使用 LogQL 查询 Loki 日志数据，适用于 Grafana Loki 日志聚合系统的日志检索场景。",
    configFields: [
      { name: "query", type: "string", required: true, description: 'LogQL 查询语句，例如 {app="my-service"} |= "error"' },
      { name: "labels", type: "object", required: false, description: "标签过滤条件，键值对形式" },
      { name: "timeRange", type: "string", required: false, description: "时间范围" },
      { name: "limit", type: "number", required: false, description: "返回结果数量限制" },
    ],
    example: [
      { label: "query", value: '{app="api-server"} |= "timeout" | json | line_format "{{.message}}"' },
      { label: "limit", value: "100" },
    ],
  },
  {
    type: "ClickHouse",
    title: "ClickHouse 数据查询",
    category: "Integration",
    icon: "bar-chart-3",
    description: "执行 ClickHouse SQL 查询，适用于大规模数据分析场景，可查询指标、日志等时序数据。",
    configFields: [
      { name: "database", type: "string", required: true, description: "目标数据库名称" },
      { name: "query", type: "string", required: true, description: "SQL 查询语句" },
    ],
    example: [
      { label: "database", value: "metrics" },
      { label: "query", value: "SELECT toStartOfMinute(timestamp) AS minute, avg(value) FROM cpu_metrics WHERE timestamp > now() - INTERVAL 1 HOUR GROUP BY minute ORDER BY minute" },
    ],
  },
  {
    type: "Kubernetes",
    title: "Kubernetes 资源查询",
    category: "Integration",
    icon: "activity",
    description: "查询 Kubernetes 集群中 Pod 的运行状态和事件信息，用于排查容器编排相关问题。",
    configFields: [
      { name: "namespace", type: "string", required: false, description: "命名空间，不填则查询所有命名空间" },
      { name: "pod", type: "string", required: false, description: 'Pod 名称，填 "all" 或留空查询所有 Pod' },
      { name: "resourceType", type: "select", required: false, description: "查询类型：status（Pod 状态）或 event（事件信息）" },
    ],
    example: [
      { label: "namespace", value: "production" },
      { label: "pod", value: "api-server-7d9f8c6b5-x2k4j" },
      { label: "resourceType", value: "status" },
    ],
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
    example: [
      { label: "host", value: "192.168.1.100" },
      { label: "port", value: "22" },
      { label: "command", value: "df -h && free -m && top -bn1 | head -20" },
      { label: "timeout", value: "30" },
    ],
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
    example: [
      { label: "code", value: `import "fmt"

func main() string {
    return "hi!"
}`, isCode: true },
      { label: "timeout", value: "10" },
    ],
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
    example: [
      { label: "url", value: "https://api.example.com/alert" },
      { label: "method", value: "POST" },
      { label: "headers", value: '{"Content-Type": "application/json", "Authorization": "Bearer token123"}' },
      { label: "body", value: '{"text": "服务异常告警", "severity": "critical"}' },
      { label: "timeout", value: "15" },
    ],
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
    example: [
      { label: "model", value: "gpt-4" },
      { label: "systemPrompt", value: "你是一个 SRE 专家，擅长分析系统故障并提供排查建议。", isCode: true },
      { label: "prompt", value: "以下是 Prometheus 监控数据：{{prometheus_1.output}}\n\n请分析异常原因并给出排查建议。", isCode: true },
    ],
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
    example: [
      { label: "condition", value: "{{prometheus_1.output}}" },
      { label: "operator", value: "greaterThan" },
      { label: "value", value: "0.95" },
    ],
  },
  {
    type: "SmartRouter",
    title: "SmartRouter 智能路由",
    category: "Logic",
    icon: "route",
    description: "基于 LLM 的智能路由节点。自动分析上游输入数据，选择最合适的分支继续执行。与 Condition 的固定规则不同，SmartRouter 由 AI 根据数据语义动态判断路由方向，适合需要理解上下文语义的复杂分流场景，如告警分类、故障方向判断等。运行时会自动合并 LLM Integration 配置（apiKey、baseURL）。",
    configFields: [
      { name: "model", type: "string", required: false, description: "LLM 模型名称，例如 gpt-4o，默认 gpt-4o" },
      { name: "branches", type: "string", required: true, description: '分支定义 JSON 数组，每项包含 id、label、description。例如：[{"id":"downstream","label":"下游服务异常","description":"日志中显示下游调用超时或 5xx"}]' },
    ],
    example: [
      { label: "model", value: "gpt-4o" },
      { label: "branches", value: '[\n  {"id":"downstream","label":"下游服务异常","description":"日志中显示下游调用超时或 5xx"},\n  {"id":"resource","label":"资源瓶颈","description":"CPU/内存/磁盘等资源指标异常"},\n  {"id":"config","label":"配置变更","description":"近期有部署或配置变更"}\n]', isCode: true },
    ],
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
    example: [
      { label: "text", value: "开始排查服务异常问题" },
    ],
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
    example: [
      { label: "outputType", value: "file" },
      { label: "fileType", value: "md" },
      { label: "customFilename", value: "rca-report" },
      { label: "includeTimestamp", value: "true" },
      { label: "formatContent", value: "true" },
    ],
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
    example: [
      { label: "serverId", value: "mcp-server-001" },
      { label: "serverName", value: "运维工具集" },
      { label: "toolName", value: "restart_service" },
      { label: "arguments", value: '{"service_name": "api-server", "namespace": "production"}' },
    ],
  },
];

export const NODE_CATEGORIES = [
  { id: "Integration", label: "Integration 数据集成", nodes: NODE_HELP_CONTENT.filter(n => n.category === "Integration") },
  { id: "Action", label: "Action 执行操作", nodes: NODE_HELP_CONTENT.filter(n => n.category === "Action") },
  { id: "AI", label: "AI 智能分析", nodes: NODE_HELP_CONTENT.filter(n => n.category === "AI") },
  { id: "Logic", label: "Logic 流程控制", nodes: NODE_HELP_CONTENT.filter(n => n.category === "Logic") },
  { id: "Output", label: "Output 结果输出", nodes: NODE_HELP_CONTENT.filter(n => n.category === "Output") },
];
