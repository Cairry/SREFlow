"use client";

import React, { useState } from "react";
import { cn } from "@/lib/utils";
import {
  FileText,
  Download,
  Copy,
  Clock,
  Loader2,
  CheckCircle,
  XCircle,
  Sparkles,
  GitBranch,
  Activity,
  Search,
  BarChart3,
  Play,
  Code,
  ChevronRight,
  X,
  Ban,
  MinusCircle,
} from "lucide-react";
import type { WorkflowNode, WorkflowExecution, WorkflowNodeType } from "@/lib/workflow-types";

interface NodeResultDisplay {
  nodeId: string;
  nodeType: string;
  output: string | null;
  outputs?: Record<string, unknown>;
  error?: string;
  timestamp: Date;
  status: 'idle' | 'pending' | 'running' | 'completed' | 'failed' | 'cancelled' | 'skipped';
}

interface OutputPanelProps {
  execution: WorkflowExecution | null;
  nodeResults?: { nodeId: string; output: string | null; outputs?: Record<string, unknown>; error?: string }[];
  isExecuting: boolean;
  onClose: () => void;
  nodes: WorkflowNode[];
  nodeStatuses?: Map<string, 'idle' | 'pending' | 'running' | 'completed' | 'failed' | 'cancelled' | 'skipped'>;
}

export function OutputPanel({
  execution,
  nodeResults,
  isExecuting,
  onClose,
  nodes,
  nodeStatuses,
}: OutputPanelProps) {
  const [copied, setCopied] = useState(false);
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set());

  const toggleNodeExpansion = (nodeId: string) => {
    setExpandedNodes((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(nodeId)) {
        newSet.delete(nodeId);
      } else {
        newSet.add(nodeId);
      }
      return newSet;
    });
  };

  const handleCopy = async (content: string | null) => {
    if (!content) return;
    const contentStr =
      typeof content === "string" ? content : JSON.stringify(content, null, 2);
    await navigator.clipboard.writeText(contentStr);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleDownload = (
    content: string | null,
    filename?: string,
    mimeType?: string
  ) => {
    if (!content) return;
    const contentStr =
      typeof content === "string" ? content : JSON.stringify(content, null, 2);
    const blob = new Blob([contentStr], { type: mimeType || "text/plain" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = filename || "output.txt";
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  const getStatusIcon = (status: string, error?: string) => {
    if (error) return <XCircle className="w-3.5 h-3.5 text-red-500" />;
    switch (status) {
      case "running":
        return <Loader2 className="w-3.5 h-3.5 text-blue-500 animate-spin" />;
      case "completed":
        return <CheckCircle className="w-3.5 h-3.5 text-green-500" />;
      case "failed":
        return <XCircle className="w-3.5 h-3.5 text-red-500" />;
      case "cancelled":
        return <Ban className="w-3.5 h-3.5 text-stone-400" />;
      case "skipped":
        return <MinusCircle className="w-3.5 h-3.5 text-stone-400" />;
      default:
        return <Clock className="w-3.5 h-3.5 text-yellow-500" />;
    }
  };

  const formatOutput = (output: string | null, maxLength: number = 200) => {
    if (!output) return "No output";
    const outputStr =
      typeof output === "string" ? output : JSON.stringify(output, null, 2);
    // 不截断，直接返回原始内容
    return outputStr;
  };

  // Build a map from nodeId -> execution result for quick lookup
  const resultMap = new Map<string, { output: string | null; outputs?: Record<string, unknown>; error?: string }>(
    (nodeResults || execution?.results || []).map((r) => [r.nodeId, { output: r.output, outputs: r.outputs, error: r.error }])
  );

  const hasResults = (nodeResults?.length ?? 0) > 0 || (execution?.results?.length ?? 0) > 0;

  // Build display items from nodes + statuses
  const displayItems: NodeResultDisplay[] = nodes.map((node) => {
    const nodeStatus = nodeStatuses?.get(node.id) || "pending";
    const result = resultMap.get(node.id);

    // Determine effective status: follow canvas logic — read from nodeStatuses, default idle
    const effectiveStatus: 'idle' | 'pending' | 'running' | 'completed' | 'failed' | 'cancelled' | 'skipped' =
      (nodeStatus === 'running' || nodeStatus === 'completed' || nodeStatus === 'failed' || nodeStatus === 'cancelled' || nodeStatus === 'skipped')
        ? nodeStatus
        : isExecuting
          ? 'pending'
          : 'idle';

    return {
      nodeId: node.id,
      nodeType: node.type,
      output: result?.output || null,
      outputs: result?.outputs,
      error: result?.error,
      timestamp: new Date(),
      status: effectiveStatus,
    };
  });

  const statusLabel =
    !execution && isExecuting
      ? "Starting"
      : execution?.status === "completed"
      ? "Completed"
      : execution?.status === "failed"
      ? "Failed"
      : isExecuting
      ? "Running"
      : "Done";

  const statusColor =
    execution?.status === "completed"
      ? "bg-green-500/20 text-green-400"
      : execution?.status === "failed"
      ? "bg-red-500/20 text-red-400"
      : isExecuting
      ? "bg-blue-500/20 text-blue-400"
      : "bg-yellow-500/20 text-yellow-400";

  return (
    <div className="h-full flex flex-col bg-workflow-node-bg">
      {/* Header */}
      <div className="flex items-center justify-between px-4 border-workflow-border shrink-0 mt-3">
        <div className="flex items-center gap-2">
          <span className="font-mono text-sm font-semibold text-workflow-text">
            Execution Progress
          </span>
          <span className={cn("px-2 py-0.5 text-xs font-mono rounded-full flex items-center gap-1", statusColor)}>
            {isExecuting && <Loader2 className="w-3 h-3 animate-spin" />}
            {statusLabel}
          </span>
        </div>
        <button
          type="button"
          onClick={onClose}
          className="p-1.5 hover:bg-workflow-surface-hover rounded-lg transition-colors duration-200 text-workflow-text-muted hover:text-workflow-text"
        >
          <X className="w-4 h-4" />
        </button>
      </div>

      {/* Node list */}
      {displayItems.length > 0 ? (
        <div className="flex-1 overflow-auto">
          <div className="p-4 space-y-2">
            {displayItems.map((item) => {
              const isExpanded = expandedNodes.has(item.nodeId);

              return (
                <div
                  key={item.nodeId}
                  className="border border-workflow-border/50 rounded-lg overflow-hidden bg-workflow-surface/50 transition-colors duration-200"
                >
                  <button
                    type="button"
                    onClick={() => toggleNodeExpansion(item.nodeId)}
                    className="w-full p-3 bg-workflow-node-input/30 hover:bg-workflow-node-input/50 transition-colors duration-200 text-left"
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        {getStatusIcon(item.status, item.error)}
                        <span className="font-mono text-sm text-workflow-text">{item.nodeType}</span>
                        <span className="font-mono text-xs text-workflow-text-muted">({item.nodeId.slice(0, 8)}...)</span>
                      </div>
                      <span className="text-xs text-workflow-text-muted">
                        {isExpanded ? '▼' : '▶'}
                      </span>
                    </div>
                  </button>
                  
                  {isExpanded && (
                    <div className="p-3 border-t border-workflow-border/50 bg-workflow-surface/50">
                      <div className="space-y-2">
                        <div className="flex items-center justify-between">
                          <div className="text-xs font-mono text-workflow-text-muted">Output:</div>
                          <div className="flex gap-1">
                            {item.output && (
                              <button
                                onClick={(e) => {
                                  e.stopPropagation();
                                  handleCopy(item.output!);
                                }}
                                className="p-1 hover:bg-workflow-surface rounded transition-colors duration-200"
                                title="Copy output"
                              >
                                <Copy className="w-3 h-3" />
                              </button>
                            )}
                            {item.output && item.nodeType === "Output" && (
                              <button
                                onClick={(e) => {
                                  e.stopPropagation();
                                  const content = (item.outputs?.output as string) || item.output || "";
                                  const filename = (item.outputs?.filename as string) || "output.md";
                                  handleDownload(content, filename, "text/markdown");
                                }}
                                className="p-1 hover:bg-workflow-surface rounded transition-colors duration-200"
                                title="Download output"
                              >
                                <Download className="w-3 h-3" />
                              </button>
                            )}
                          </div>
                        </div>
                        <div className="p-2 bg-workflow-node-input/30 border border-workflow-border/30 rounded font-mono text-xs text-workflow-text break-all max-h-40 overflow-y-auto">
                          {item.outputs && Object.keys(item.outputs).length > 0 ? (
                            <div className="space-y-1">
                              {Object.entries(item.outputs).map(([key, value]) => {
                                const isErrorField = key === 'error' && value && String(value).length > 0;
                                return (
                                  <div key={key} className="border-b border-workflow-border/30 pb-1 last:border-b-0">
                                    <div className={cn(
                                      "text-xs font-semibold mb-1",
                                      isErrorField ? "text-red-500" : "text-workflow-text-muted"
                                    )}>{key}:</div>
                                    <div className={cn(
                                      "text-xs whitespace-pre-wrap",
                                      isErrorField ? "text-red-400" : "text-workflow-text"
                                    )}>
                                      {typeof value === 'string' ? value : JSON.stringify(value, null, 2)}
                                    </div>
                                  </div>
                                );
                              })}
                            </div>
                          ) : item.error ? (
                            <div className="text-red-400">{item.error}</div>
                          ) : item.output ? (
                            formatOutput(item.output, 500)
                          ) : hasResults ? (
                            <div className="text-xs text-workflow-text-muted italic">No output</div>
                          ) : (
                            <div className="text-xs text-workflow-text-muted italic">Waiting for output...</div>
                          )}
                        </div>
                      </div>
                      <div className="mt-2 text-xs font-mono text-workflow-text-muted">
                        {item.status === "running" ? (
                          <span className="text-blue-400">Processing...</span>
                        ) : item.status === "completed" ? (
                          <span className="text-green-400">Completed</span>
                        ) : item.status === "failed" ? (
                          <span className="text-red-400">Failed</span>
                        ) : item.status === "cancelled" ? (
                          <span className="text-stone-400">Cancelled</span>
                        ) : item.status === "skipped" ? (
                          <span className="text-stone-400">Skipped (no connections)</span>
                        ) : (
                          <span className="text-yellow-400">Pending</span>
                        )}
                        {" • "}
                        {item.timestamp.toLocaleString()}
                      </div>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      ) : (
        <div className="flex-1 flex items-center justify-center p-4">
          <div className="text-center text-workflow-text-subtle">
            <FileText className="w-12 h-12 mx-auto mb-3 opacity-30" />
            <p className="text-sm font-mono">No execution yet</p>
            <p className="text-xs mt-1">Run your workflow to see progress</p>
          </div>
        </div>
      )}
    </div>
  );
}
