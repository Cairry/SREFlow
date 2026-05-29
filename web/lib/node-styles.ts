import type { WorkflowNodeType } from "./workflow-types";

export const NODE_COLORS: Record<WorkflowNodeType, { bg: string; border: string; accent: string; handle?: string }> = {
  LLM: {
    bg: "bg-workflow-node-bg",
    border: "border-gray-500/50 dark:border-gray-500/50",
    accent: "text-gray-600 dark:text-gray-400",
    handle: "bg-gray-500 border-gray-600",
  },
  Condition: {
    bg: "bg-workflow-node-bg",
    border: "border-gray-500/50 dark:border-gray-500/50",
    accent: "text-gray-600 dark:text-gray-400",
    handle: "bg-gray-500 border-gray-600",
  },
  SmartRouter: {
    bg: "bg-workflow-node-bg",
    border: "border-amber-500/50 dark:border-amber-500/50",
    accent: "text-amber-600 dark:text-amber-400",
    handle: "bg-amber-500 border-amber-600",
  },
  Prometheus: {
    bg: "bg-workflow-node-bg",
    border: "border-gray-500/50 dark:border-gray-500/50",
    accent: "text-gray-600 dark:text-gray-400",
    handle: "bg-gray-500 border-gray-600",
  },
  Jaeger: {
    bg: "bg-workflow-node-bg",
    border: "border-gray-500/50 dark:border-gray-500/50",
    accent: "text-gray-600 dark:text-gray-400",
    handle: "bg-gray-500 border-gray-600",
  },
  Elasticsearch: {
    bg: "bg-workflow-node-bg",
    border: "border-gray-500/50 dark:border-gray-500/50",
    accent: "text-gray-600 dark:text-gray-400",
    handle: "bg-gray-500 border-gray-600",
  },
  Loki: {
    bg: "bg-workflow-node-bg",
    border: "border-gray-500/50 dark:border-gray-500/50",
    accent: "text-gray-600 dark:text-gray-400",
    handle: "bg-gray-500 border-gray-600",
  },
  ClickHouse: {
    bg: "bg-workflow-node-bg",
    border: "border-gray-500/50 dark:border-gray-500/50",
    accent: "text-gray-600 dark:text-gray-400",
    handle: "bg-gray-500 border-gray-600",
  },
  SSH: {
    bg: "bg-workflow-node-bg",
    border: "border-gray-500/50 dark:border-gray-500/50",
    accent: "text-gray-600 dark:text-gray-400",
    handle: "bg-gray-500 border-gray-600",
  },
  Output: {
    bg: "bg-workflow-node-bg",
    border: "border-gray-500/50 dark:border-gray-500/50",
    accent: "text-gray-600 dark:text-gray-400",
    handle: "bg-gray-500 border-gray-600",
  },
  Start: {
    bg: "bg-workflow-node-bg",
    border: "border-gray-500/50 dark:border-gray-500/50",
    accent: "text-gray-600 dark:text-gray-400",
    handle: "bg-gray-500 border-gray-600",
  },
  GoScript: {
    bg: "bg-workflow-node-bg",
    border: "border-gray-500/50 dark:border-gray-500/50",
    accent: "text-gray-600 dark:text-gray-400",
    handle: "bg-gray-500 border-gray-600",
  },
  HTTP: {
    bg: "bg-workflow-node-bg",
    border: "border-gray-500/50 dark:border-gray-500/50",
    accent: "text-gray-600 dark:text-gray-400",
    handle: "bg-gray-500 border-gray-600",
  },
  Kubernetes: {
    bg: "bg-workflow-node-bg",
    border: "border-gray-500/50 dark:border-gray-500/50",
    accent: "text-gray-600 dark:text-gray-400",
    handle: "bg-gray-500 border-gray-600",
  },
  MCP: {
    bg: "bg-workflow-node-bg",
    border: "border-purple-500/50 dark:border-purple-500/50",
    accent: "text-purple-600 dark:text-purple-400",
    handle: "bg-purple-500 border-purple-600",
  },
};

export const NODE_ICONS: Record<WorkflowNodeType, string> = {
  LLM: "sparkles",
  Condition: "git-branch",
  SmartRouter: "route",
  Prometheus: "activity",
  Jaeger: "search",
  Elasticsearch: "search",
  Loki: "search",
  ClickHouse: "bar-chart-3",
  SSH: "terminal",
  Output: "file-output",
  Start: "play",
  GoScript: "code",
  HTTP: "globe",
  Kubernetes: "container",
  MCP: "puzzle",
};
