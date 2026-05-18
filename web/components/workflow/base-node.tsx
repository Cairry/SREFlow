"use client";

import React from "react";

import { Handle, Position, type NodeProps } from "@xyflow/react";
import { cn } from "@/lib/utils";
import type { WorkflowNodeType, NodeRunStatus } from "@/lib/workflow-types";
import { NODE_COLORS } from "@/lib/node-styles";
import {
  Sparkles,
  GitBranch,
  FileOutput,
  Type,
  Activity,
  Search,
  Database,
  BarChart3,
  X,
  Loader2,
  CheckCircle,
  XCircle,
  Clock,
  Code,
  Play,
  Ban,
  Globe,
  MinusCircle,
  Puzzle,
} from "lucide-react";
import { useReactFlow } from "@xyflow/react";

const ICONS: Record<WorkflowNodeType, React.ReactNode> = {
  LLM: <Sparkles className="w-4 h-4" />,
  Condition: <GitBranch className="w-4 h-4" />,
  Prometheus: <Activity className="w-4 h-4" />,
  Jaeger: <Search className="w-4 h-4" />,
  Elasticsearch: <Search className="w-4 h-4" />,
  Loki: <Search className="w-4 h-4" />,
  ClickHouse: <BarChart3 className="w-4 h-4" />,
  Output: <FileOutput className="w-4 h-4" />,
  Start: <Play className="w-4 h-4" />,
  GoScript: <Code className="w-4 h-4" />,
  HTTP: <Globe className="w-4 h-4" />,
  MCP: <Puzzle className="w-4 h-4" />,
};

const STATUS_ICONS: Record<NodeRunStatus, React.ReactNode> = {
  idle: null,
  pending: <Clock className="w-3 h-3 text-yellow-500" />,
  running: <Loader2 className="w-3 h-3 text-blue-500 animate-spin" />,
  completed: <CheckCircle className="w-3 h-3 text-green-500" />,
  failed: <XCircle className="w-3 h-3 text-red-500" />,
  cancelled: <Ban className="w-3 h-3 text-stone-400" />,
  skipped: <MinusCircle className="w-3 h-3 text-stone-400" />,
};

const STATUS_BORDER_COLORS: Record<NodeRunStatus, string> = {
  idle: "border-workflow-border",
  pending: "border-yellow-500/50",
  running: "border-blue-500/50",
  completed: "border-green-500/50",
  failed: "border-red-500/50",
  cancelled: "border-stone-500/50",
  skipped: "border-stone-500/50",
};

interface BaseNodeProps extends Omit<NodeProps<any>, 'data'> {
  nodeType: WorkflowNodeType;
  children: React.ReactNode;
  hasInput?: boolean;
  hasOutput?: boolean;
  hasTrueOutput?: boolean;
  hasFalseOutput?: boolean;
  data?: Record<string, unknown>;
  runStatus?: NodeRunStatus;
}

export function BaseNode({
  nodeType,
  children,
  hasInput = true,
  hasOutput = true,
  hasTrueOutput = false,
  hasFalseOutput = false,
  selected,
  data,
  id,
  runStatus,
}: BaseNodeProps) {
  const colors = NODE_COLORS[nodeType] || {
    bg: "bg-workflow-node-bg",
    border: "border-gray-500/50",
    accent: "text-gray-600",
    handle: "bg-gray-500 border-gray-600",
  };
  const icon = ICONS[nodeType] || <Type className="w-4 h-4" />;
  const { deleteElements } = useReactFlow();

  const handleDelete = (e: React.MouseEvent) => {
    e.stopPropagation();
    deleteElements({ nodes: [{ id }] });
  };

  const statusBorderColor = runStatus ? STATUS_BORDER_COLORS[runStatus] : colors.border;
  
  // 状态背景色（更明显的视觉反馈）
  const statusBgColor = runStatus ? {
    pending: "bg-yellow-50/20",
    running: "bg-blue-50/20",
    completed: "bg-green-50/20",
    failed: "bg-red-50/20",
    cancelled: "bg-stone-50/20",
    skipped: "bg-stone-50/20",
    idle: ""
  }[runStatus] : "";

  return (
    <div
      className={cn(
        "min-w-[280px] rounded-lg border-2 shadow-lg transition-all duration-300 group",
        colors.bg,
        statusBorderColor,
        statusBgColor,
        selected && "ring-2 ring-primary/30",
        runStatus === 'running' && "shadow-blue-500/20",
        runStatus === 'completed' && "shadow-green-500/20",
        runStatus === 'failed' && "shadow-red-500/20"
      )}
    >
      {hasInput && (
        <Handle
          type="target"
          position={Position.Left}
          className="!w-1 !h-4 !bg-workflow-handle !border !border-workflow-handle-border/30 rounded-sm"
        />
      )}

      <div className={cn("flex items-center gap-2 px-3 py-2 border-b border-workflow-border-subtle", colors.accent)}>
        {icon}
        <span className="font-mono text-sm font-medium flex-1 text-workflow-text">{nodeType}</span>
        {runStatus && runStatus !== 'idle' && (
          <div className="flex items-center gap-1">
            {STATUS_ICONS[runStatus]}
          </div>
        )}
        <button
          type="button"
          onClick={handleDelete}
          className="p-1 rounded hover:bg-workflow-surface-hover text-workflow-text-muted hover:text-workflow-text transition-all duration-200 opacity-0 group-hover:opacity-100 group-focus-within:opacity-100"
          aria-label="Delete node"
        >
          <X className="w-3.5 h-3.5" />
        </button>
      </div>

      <div className="p-3 space-y-3">{children}</div>

      {hasOutput && !hasTrueOutput && (
        <Handle
          type="source"
          position={Position.Right}
          className="!w-1 !h-4 !bg-workflow-handle !border !border-workflow-handle-border/30 rounded-sm"
        />
      )}

      {hasTrueOutput && (
        <>
          <Handle
            type="source"
            position={Position.Right}
            id="true"
            style={{ top: "40%" }}
            className="!w-1 !h-4 !bg-gray-500 !border !border-gray-300/30 rounded-sm"
          />
          <Handle
            type="source"
            position={Position.Right}
            id="false"
            style={{ top: "70%" }}
            className="!w-1 !h-4 !bg-gray-500 !border !border-gray-300/30 rounded-sm"
          />
          <div className="absolute right-[-40px] top-[35%] text-xs text-gray-500 dark:text-gray-400 font-mono">true</div>
          <div className="absolute right-[-40px] top-[65%] text-xs text-gray-500 dark:text-gray-400 font-mono">false</div>
        </>
      )}
    </div>
  );
}
