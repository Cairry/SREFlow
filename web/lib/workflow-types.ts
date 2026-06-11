import type { Node, Edge } from "@xyflow/react";

// Node Data Types
export interface BaseNodeData {
  runStatus?: NodeRunStatus;
  [key: string]: unknown;
}

export interface LLMNodeData extends BaseNodeData {
  model: string;
  prompt: string;
  systemPrompt?: string;
}

export interface SmartRouterNodeData extends BaseNodeData {
  model: string;
  branches: string; // JSON string of branch definitions
}

export interface ConditionNodeData extends BaseNodeData {
  condition: string;
  operator: "contains" | "equals" | "notEquals" | "greaterThan" | "lessThan" | "isEmpty" | "isNotEmpty";
  value: string;
}

export interface PrometheusNodeData extends BaseNodeData {
  prometheusUrl: string;
  query: string;
  timeout?: number;
}

export interface JaegerNodeData extends BaseNodeData {
  jaegerUrl: string;
  service?: string;
  tags?: string;
  timeRange?: string;
  limit?: number;
}

export interface ElasticsearchNodeData extends BaseNodeData {
  elasticsearchUrl: string;
  index: string;
  query: string;
  queryType: "match" | "term" | "range" | "bool";
  timeRange?: string;
  size?: number;
}

export interface LokiNodeData extends BaseNodeData {
  lokiUrl: string;
  query: string;
  labels?: Record<string, string>;
  timeRange?: string;
  limit?: number;
}

export interface ClickHouseNodeData extends BaseNodeData {
  clickhouseUrl: string;
  database: string;
  query: string;
  username?: string;
  password?: string;
}

export interface SSHNodeData extends BaseNodeData {
  host: string;
  port?: number;
  command: string;
  timeout?: number;
}

export interface KubernetesNodeData extends BaseNodeData {
  namespace?: string;
  pod?: string; // "all" or specific pod name
  resourceType?: "status" | "event";
}

export interface WebhookConfig {
  url: string;
  method: "GET" | "POST" | "PUT" | "PATCH";
  headers?: Record<string, string>;
  body?: string;
  timeout?: number;
}

export interface OutputNodeData extends BaseNodeData {
  outputType: "file" | "webhook";
  customFilename?: string;
  fileType?: "txt" | "md" | "json" | "csv" | "html" | "pdf";
  autoDownload?: boolean;
  includeTimestamp?: boolean;
  formatContent?: boolean;
  webhook?: WebhookConfig;
}

export interface StartNodeData extends BaseNodeData {
  text: string;
}

export interface GoScriptNodeData extends BaseNodeData {
  code: string;
  timeout?: number;
}

export interface MCPNodeData extends BaseNodeData {
  serverId: string;
  serverName: string;
  toolName: string;
  description: string;
  arguments: Record<string, string>;
}

export interface AgentNodeData extends BaseNodeData {
  model: string;
  prompt?: string;
  maxIterations?: number;
  skillIds?: string[];
  selectedTools?: string[]; // "serverId::toolName" format
}

export interface HTTPNodeData extends BaseNodeData {
  url: string;
  method: "GET" | "POST" | "PUT" | "PATCH";
  headers?: Record<string, string>;
  body?: string;
  timeout?: number;
}

// Union type for all node data
export type WorkflowNodeData =
  | LLMNodeData
  | ConditionNodeData
  | SmartRouterNodeData
  | PrometheusNodeData
  | JaegerNodeData
  | ElasticsearchNodeData
  | LokiNodeData
  | ClickHouseNodeData
  | SSHNodeData
  | KubernetesNodeData
  | OutputNodeData
  | StartNodeData
  | GoScriptNodeData
  | HTTPNodeData
  | MCPNodeData
  | AgentNodeData;

// Node types enum
export type WorkflowNodeType =
  | "LLM"
  | "Condition"
  | "SmartRouter"
  | "Prometheus"
  | "Jaeger"
  | "Elasticsearch"
  | "Loki"
  | "ClickHouse"
  | "SSH"
  | "Kubernetes"
  | "Output"
  | "Start"
  | "GoScript"
  | "HTTP"
  | "MCP"
  | "Agent";

// Workflow Node with typed data
export type WorkflowNode = Node<WorkflowNodeData, WorkflowNodeType>;

// Workflow definition
export interface Workflow {
  id: string;
  name: string;
  description?: string;
  nodes: WorkflowNode[];
  edges: Edge[];
  createdAt: Date;
  updatedAt: Date;
}

// Execution result
export interface ExecutionResult {
  nodeId: string;
  nodeType: WorkflowNodeType;
  output: string | null;
  outputs?: Record<string, unknown>; // raw outputs map from backend
  error?: string;
  timestamp: Date;
}

export interface WorkflowExecution {
  id: string;
  workflowId: string;
  status: "pending" | "running" | "completed" | "failed";
  results: ExecutionResult[];
  finalOutput?: string;
  startedAt: Date;
  completedAt?: Date;
  runningNodes?: string[]; // 当前正在运行的节点ID
  completedNodes?: string[]; // 已完成的节点ID
  failedNodes?: string[]; // 失败的节点ID
}

export type NodeRunStatus = "idle" | "pending" | "running" | "completed" | "failed" | "cancelled" | "skipped";

// SSE 事件类型
export interface ExecutionSSEEvent {
  type: "workflow_started" | "node_started" | "node_completed" | "node_failed" | "workflow_completed" | "workflow_failed";
  executionId: string;
  nodeId?: string;
  nodeType?: string;
  status: string;
  error?: string;
  outputs?: Record<string, unknown>;
  timestamp: number;
  // 每次事件推送完整的节点状态快照
  nodeStates: Record<string, string>; // nodeId -> state
}
