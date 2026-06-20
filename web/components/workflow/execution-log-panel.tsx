"use client";

import React, { useState, useEffect, useRef } from "react";
import { cn } from "@/lib/utils";
import {
  Play,
  Square,
  Download,
  Copy,
  Clock,
  Loader2,
  CheckCircle,
  XCircle,
  Ban,
  MinusCircle,
  ChevronDown,
  ChevronUp,
  FileText,
  History,
} from "lucide-react";
import type { WorkflowNode, NodeRunStatus, ExecutionResult } from "@/lib/workflow-types";
import type { RunHistoryItem } from "@/lib/api/workflow";

// Workflow execution status (only these 4 states)
export type WorkflowExecutionStatus = 'running' | 'completed' | 'failed' | 'cancelled';

interface LogEntry {
  timestamp: Date;
  nodeId: string;
  nodeType: string;
  status: NodeRunStatus;
  output?: string | null;
  outputs?: Record<string, unknown>;
  error?: string;
}

interface ExecutionLogPanelProps {
  workflowId: string | null;
  isExecuting: boolean;
  executionId: string | null;
  executionStatus: WorkflowExecutionStatus | null;
  nodeResults: ExecutionResult[];
  nodes: WorkflowNode[];
  nodeStatuses: Map<string, NodeRunStatus>;
  onClear: () => void;
  onStart: () => void;
  onStop: () => void;
}

export function ExecutionLogPanel({
  workflowId,
  isExecuting,
  executionId,
  executionStatus,
  nodeResults,
  nodes,
  nodeStatuses,
  onClear,
  onStart,
  onStop,
}: ExecutionLogPanelProps) {
  const [copied, setCopied] = useState(false);
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set());
  const [isMinimized, setIsMinimized] = useState(true);
  const [activeTab, setActiveTab] = useState<'execution' | 'history'>('execution');
  const [history, setHistory] = useState<RunHistoryItem[]>([]);
  const [selectedHistory, setSelectedHistory] = useState<RunHistoryItem | null>(null);
  const [historyLoading, setHistoryLoading] = useState(false);
  const logEndRef = useRef<HTMLDivElement>(null);

  // Load history when tab switches to history
  useEffect(() => {
    if (activeTab === 'history' && workflowId) {
      setHistoryLoading(true);

      fetch(`/api/workflow/${workflowId}/run-history?limit=50`)
        .then(res => res.json())
        .then(data => setHistory((data.data?.history || []) as RunHistoryItem[]))
        .catch(console.error)
        .finally(() => setHistoryLoading(false));
    }
  }, [activeTab, workflowId]);

  // Build a map from nodeId -> execution result for quick lookup
  const resultMap = new Map<string, { output: string | null; outputs?: Record<string, unknown>; error?: string }>(
    nodeResults.map((r) => [r.nodeId, { output: r.output, outputs: r.outputs, error: r.error }])
  );

  // Build log entries from node statuses
  const logEntries: LogEntry[] = nodes.map((node) => {
    const status = nodeStatuses.get(node.id) || 'pending';
    const result = resultMap.get(node.id);
    
    return {
      timestamp: new Date(),
      nodeId: node.id,
      nodeType: node.type,
      status,
      output: result?.output,
      outputs: result?.outputs,
      error: result?.error,
    };
  });

  // Auto-scroll to bottom when new logs arrive
  useEffect(() => {
    if (!isMinimized && isExecuting && activeTab === 'execution') {
      logEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }
  }, [logEntries, isMinimized, isExecuting, activeTab]);

  // Auto-expand running/completed nodes
  useEffect(() => {
    if (isExecuting && activeTab === 'execution') {
      const activeNodes = new Set<string>();
      nodeStatuses.forEach((status, nodeId) => {
        if (status === 'running' || status === 'completed' || status === 'failed') {
          activeNodes.add(nodeId);
        }
      });
      setExpandedNodes(prev => {
        const newSet = new Set(prev);
        activeNodes.forEach(id => newSet.add(id));
        return newSet;
      });
    }
  }, [nodeStatuses, isExecuting, activeTab]);

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
    const contentStr = typeof content === "string" ? content : JSON.stringify(content, null, 2);
    await navigator.clipboard.writeText(contentStr);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleDownload = (content: string | null, filename?: string, mimeType?: string) => {
    if (!content) return;
    const contentStr = typeof content === "string" ? content : JSON.stringify(content, null, 2);
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

  // 判断历史运行中是否存在节点级错误
  const hasNodeError = (outputs?: Record<string, Record<string, unknown>>) => {
    if (!outputs) return false;
    return Object.values(outputs).some(
      nodeOut => nodeOut && typeof nodeOut === 'object' && nodeOut.error != null && String(nodeOut.error).length > 0
    );
  };

  const getHistoryStatusIcon = (run: RunHistoryItem) => {
    const hasError = run.status === 'failed' || hasNodeError(run.outputs as Record<string, Record<string, unknown>> | undefined);
    if (hasError) return <XCircle className="w-4 h-4 text-red-500" />;
    switch (run.status) {
      case "completed":
        return <CheckCircle className="w-4 h-4 text-green-500" />;
      case "cancelled":
        return <Ban className="w-4 h-4 text-stone-400" />;
      case "running":
        return <Loader2 className="w-4 h-4 text-blue-500 animate-spin" />;
      default:
        return <Clock className="w-4 h-4 text-stone-400" />;
    }
  };

  const runningCount = logEntries.filter(e => e.status === 'running').length;
  const completedCount = logEntries.filter(e => e.status === 'completed').length;
  const failedCount = logEntries.filter(e => e.status === 'failed').length;
  const totalNodes = nodes.length;

  const formatDuration = (started: number, finished: number | null) => {
    if (!finished) return 'Running...';
    const duration = (finished - started) * 1000;
    if (duration < 1000) return `${duration}ms`;
    return `${(duration / 1000).toFixed(2)}s`;
  };

  return (
    <div className="border-t border-workflow-border bg-workflow-node-bg flex flex-col" style={{ height: isMinimized ? 'auto' : '320px' }}>
      {/* Header with tabs */}
      <div className="flex items-center justify-between px-4 py-2 border-b border-workflow-border shrink-0">
        <div className="flex items-center gap-1">
          {/* Tab buttons */}
          <button
            type="button"
            onClick={() => setActiveTab('execution')}
            className={cn(
              "flex items-center gap-1.5 px-3 py-1.5 text-xs font-mono rounded-t transition-colors",
              activeTab === 'execution'
                ? "bg-workflow-surface text-workflow-text border-b-2 border-blue-500"
                : "text-workflow-text-muted hover:text-workflow-text"
            )}
          >
            <Play className="w-3.5 h-3.5" />
            Execution
          </button>
          <button
            type="button"
            onClick={() => setActiveTab('history')}
            className={cn(
              "flex items-center gap-1.5 px-3 py-1.5 text-xs font-mono rounded-t transition-colors",
              activeTab === 'history'
                ? "bg-workflow-surface text-workflow-text border-b-2 border-blue-500"
                : "text-workflow-text-muted hover:text-workflow-text"
            )}
          >
            <History className="w-3.5 h-3.5" />
            History
          </button>
        </div>
        
        <div className="flex items-center gap-1">
          {/* Control buttons - only show in execution tab */}
          {activeTab === 'execution' && (
            <>
              {isExecuting ? (
                <button
                  type="button"
                  onClick={onStop}
                  className="flex items-center gap-1 px-2 py-1 text-xs font-mono bg-red-500/20 text-red-400 hover:bg-red-500/30 rounded transition-colors"
                >
                  <Square className="w-3 h-3" />
                  Stop
                </button>
              ) : (
                <button
                  type="button"
                  onClick={onStart}
                  className="flex items-center gap-1 px-2 py-1 text-xs font-mono bg-blue-500/20 text-blue-400 hover:bg-blue-500/30 rounded transition-colors"
                >
                  <Play className="w-3 h-3" />
                  Run
                </button>
              )}
              {!isExecuting && logEntries.length > 0 && (
                <button
                  type="button"
                  onClick={onClear}
                  className="p-1 hover:bg-workflow-surface-hover rounded transition-colors text-workflow-text-muted hover:text-workflow-text text-xs font-mono"
                >
                  Clear
                </button>
              )}
            </>
          )}
          <button
            type="button"
            onClick={() => setIsMinimized(!isMinimized)}
            className="p-1 hover:bg-workflow-surface-hover rounded transition-colors text-workflow-text-muted hover:text-workflow-text"
          >
            {isMinimized ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
          </button>
        </div>
      </div>

      {/* Content area */}
      {!isMinimized && (
        activeTab === 'execution' ? (
          logEntries.length > 0 ? (
            <div className="flex-1 overflow-auto">
              <div className="p-3 space-y-1">
                <div className="text-xs font-mono text-workflow-text-muted mb-2">
                  {completedCount + failedCount}/{totalNodes} nodes
                  {runningCount > 0 && ` • ${runningCount} running`}
                </div>
                {logEntries.map((entry) => {
                  const isExpanded = expandedNodes.has(entry.nodeId);

                  return (
                    <div
                      key={entry.nodeId}
                      className="border border-workflow-border/50 rounded-lg overflow-hidden bg-workflow-surface/30 transition-colors duration-200"
                    >
                      <button
                        type="button"
                        onClick={() => toggleNodeExpansion(entry.nodeId)}
                        className="w-full px-3 py-2 bg-workflow-node-input/20 hover:bg-workflow-node-input/40 transition-colors duration-200 text-left"
                      >
                        <div className="flex items-center justify-between">
                          <div className="flex items-center gap-2">
                            {getStatusIcon(entry.status, entry.error)}
                            <span className="font-mono text-sm text-workflow-text">{entry.nodeType}</span>
                            <span className="font-mono text-xs text-workflow-text-muted">{entry.nodeId}</span>
                          </div>
                          <span className="text-xs text-workflow-text-muted">
                            {isExpanded ? '▼' : '▶'}
                          </span>
                        </div>
                      </button>
                      
                      {isExpanded && (
                        <div className="px-3 py-2 border-t border-workflow-border/50 bg-workflow-surface/30">
                          <div className="space-y-2">
                            <div className="flex items-center justify-between">
                              <div className="text-xs font-mono text-workflow-text-muted">Output:</div>
                              <div className="flex gap-1">
                                {entry.output && (
                                  <button
                                    onClick={(e) => {
                                      e.stopPropagation();
                                      handleCopy(entry.output!);
                                    }}
                                    className="p-1 hover:bg-workflow-surface rounded transition-colors duration-200"
                                    title="Copy output"
                                  >
                                    <Copy className="w-3 h-3" />
                                  </button>
                                )}
                                {entry.output && entry.nodeType === "Output" && (
                                  <button
                                    onClick={(e) => {
                                      e.stopPropagation();
                                      const content = (entry.outputs?.output as string) || entry.output || "";
                                      const filename = (entry.outputs?.filename as string) || "output.md";
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
                            <div className="p-2 bg-workflow-node-input/30 border border-workflow-border/30 rounded font-mono text-xs text-workflow-text break-all max-h-32 overflow-y-auto">
                              {entry.outputs && Object.keys(entry.outputs).length > 0 ? (
                                <div className="space-y-1">
                                  {Object.entries(entry.outputs).map(([key, value]) => {
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
                              ) : entry.error ? (
                                <div className="text-red-400">{entry.error}</div>
                              ) : entry.output ? (
                                entry.output
                              ) : (
                                <div className="text-xs text-workflow-text-muted italic">
                                  {entry.status === 'running' ? 'Processing...' : 'No output'}
                                </div>
                              )}
                            </div>
                          </div>
                        </div>
                      )}
                    </div>
                  );
                })}
                <div ref={logEndRef} />
              </div>
            </div>
          ) : (
            <div className="flex-1 flex items-center justify-center p-4">
              <div className="text-center text-workflow-text-subtle">
                <Play className="w-10 h-10 mx-auto mb-2 opacity-30" />
                <p className="text-sm font-mono">No execution yet</p>
                <p className="text-xs mt-1">Click Run to start</p>
              </div>
            </div>
          )
        ) : (
          /* History tab */
          <div className="flex-1 overflow-auto">
            <div className="p-3 space-y-2">
              {historyLoading ? (
                <div className="flex items-center justify-center py-12">
                  <Loader2 className="w-6 h-6 animate-spin text-workflow-text-muted" />
                </div>
              ) : !workflowId ? (
                <div className="text-center py-12 text-workflow-text-muted font-mono text-sm">
                  Save your workflow first to view run history
                </div>
              ) : history.length === 0 ? (
                <div className="text-center py-12 text-workflow-text-muted font-mono text-sm">
                  No runs yet
                </div>
              ) : selectedHistory ? (
                <div className="flex-1 overflow-auto">
                  <div className="p-3 space-y-2">
                    <button
                      type="button"
                      onClick={() => setSelectedHistory(null)}
                      className="flex items-center gap-1 text-xs font-mono text-workflow-text-muted hover:text-workflow-text"
                    >
                      <ChevronDown className="w-3 h-3 rotate-90" />
                      Back to list
                    </button>
                    
                    {/* Node Outputs - expandable style like execution */}
                    {selectedHistory.outputs && Object.keys(selectedHistory.outputs).length > 0 && (
                      <div className="space-y-1">
                        <div className="text-xs font-mono text-workflow-text-muted mb-1">
                          {Object.keys(selectedHistory.outputs).length} nodes executed
                        </div>
                        {Object.entries(selectedHistory.outputs).map(([nodeId, output]) => {
                          const isExpanded = expandedNodes.has(nodeId);
                          const outputStr = typeof output === 'string' ? output : JSON.stringify(output, null, 2);
                          const nodeError = typeof output === 'object' && output !== null && 'error' in output
                            ? String((output as Record<string, unknown>).error)
                            : undefined;
                          
                          return (
                            <div
                              key={nodeId}
                              className="border border-workflow-border/50 rounded-lg overflow-hidden bg-workflow-surface/30"
                            >
                              <button
                                type="button"
                                onClick={() => toggleNodeExpansion(nodeId)}
                                className="w-full px-3 py-2 bg-workflow-node-input/20 hover:bg-workflow-node-input/40 transition-colors duration-200 text-left"
                              >
                                <div className="flex items-center justify-between">
                                  <div className="flex items-center gap-2">
                                    {getStatusIcon(nodeError ? 'failed' : 'completed', nodeError)}
                                    <span className="font-mono text-sm text-workflow-text">{nodeId}</span>
                                  </div>
                                  <span className="text-xs text-workflow-text-muted">
                                    {isExpanded ? '▼' : '▶'}
                                  </span>
                                </div>
                              </button>
                              
                              {isExpanded && (
                                <div className="px-3 py-2 border-t border-workflow-border/50 bg-workflow-surface/30">
                                  <div className="flex items-center justify-between mb-2">
                                    <div className="text-xs font-mono text-workflow-text-muted">Output:</div>
                                    <div className="flex gap-1">
                                      {outputStr && (
                                        <button
                                          onClick={(e) => {
                                            e.stopPropagation();
                                            handleCopy(outputStr);
                                          }}
                                          className="p-1 hover:bg-workflow-surface rounded transition-colors duration-200"
                                          title="Copy output"
                                        >
                                          <Copy className="w-3 h-3" />
                                        </button>
                                      )}
                                    </div>
                                  </div>
                                  <div className="p-2 bg-workflow-node-input/30 border border-workflow-border/30 rounded font-mono text-xs text-workflow-text break-all max-h-32 overflow-y-auto">
                                    {outputStr || <span className="text-workflow-text-muted italic">No output</span>}
                                  </div>
                                </div>
                              )}
                            </div>
                          );
                        })}
                      </div>
                    )}
                    
                    {/* Variables */}
                    {selectedHistory.variables && Object.keys(selectedHistory.variables).length > 0 && (
                      <div className="border border-workflow-border/50 rounded-lg overflow-hidden bg-workflow-surface/30">
                        <button
                          type="button"
                          onClick={() => toggleNodeExpansion('__variables__')}
                          className="w-full px-3 py-2 bg-workflow-node-input/20 hover:bg-workflow-node-input/40 transition-colors duration-200 text-left"
                        >
                          <div className="flex items-center justify-between">
                            <div className="flex items-center gap-2">
                              <FileText className="w-3.5 h-3.5 text-workflow-text-muted" />
                              <span className="font-mono text-sm text-workflow-text">Variables</span>
                            </div>
                            <span className="text-xs text-workflow-text-muted">
                              {expandedNodes.has('__variables__') ? '▼' : '▶'}
                            </span>
                          </div>
                        </button>
                        
                        {expandedNodes.has('__variables__') && (
                          <div className="px-3 py-2 border-t border-workflow-border/50 bg-workflow-surface/30">
                            <div className="p-2 bg-workflow-node-input/30 border border-workflow-border/30 rounded font-mono text-xs text-workflow-text whitespace-pre-wrap break-all max-h-32 overflow-y-auto">
                              {JSON.stringify(selectedHistory.variables, null, 2)}
                            </div>
                          </div>
                        )}
                      </div>
                    )}
                    
                    {/* Final Output */}
                    {selectedHistory.finalOutput && (
                      <div className="border border-workflow-border/50 rounded-lg overflow-hidden bg-workflow-surface/30">
                        <button
                          type="button"
                          onClick={() => toggleNodeExpansion('__final__')}
                          className="w-full px-3 py-2 bg-workflow-node-input/20 hover:bg-workflow-node-input/40 transition-colors duration-200 text-left"
                        >
                          <div className="flex items-center justify-between">
                            <div className="flex items-center gap-2">
                              {getStatusIcon(selectedHistory.status)}
                              <span className="font-mono text-sm text-workflow-text">Final Output</span>
                            </div>
                            <span className="text-xs text-workflow-text-muted">
                              {expandedNodes.has('__final__') ? '▼' : '▶'}
                            </span>
                          </div>
                        </button>
                        
                        {expandedNodes.has('__final__') && (
                          <div className="px-3 py-2 border-t border-workflow-border/50 bg-workflow-surface/30">
                            <div className="flex items-center justify-between mb-2">
                              <div className="text-xs font-mono text-workflow-text-muted">Result:</div>
                              <div className="flex gap-1">
                                <button
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    handleCopy(selectedHistory.finalOutput!);
                                  }}
                                  className="p-1 hover:bg-workflow-surface rounded transition-colors duration-200"
                                  title="Copy output"
                                >
                                  <Copy className="w-3 h-3" />
                                </button>
                              </div>
                            </div>
                            <div className="p-2 bg-workflow-node-input/30 border border-workflow-border/30 rounded font-mono text-xs text-workflow-text break-all max-h-64 overflow-y-auto">
                              {selectedHistory.finalOutput}
                            </div>
                          </div>
                        )}
                      </div>
                    )}
                    
                    {/* Show info if no outputs at all */}
                    {!selectedHistory.outputs && !selectedHistory.variables && !selectedHistory.finalOutput && (
                      <div className="text-center py-8 text-workflow-text-muted font-mono text-sm">
                        No output data available
                      </div>
                    )}
                  </div>
                </div>
              ) : (
                <div className="flex-1 overflow-auto">
                  <div className="p-3 space-y-2">
                    {history.map((run: RunHistoryItem) => (
                      <button
                        key={run.id}
                        type="button"
                        onClick={() => setSelectedHistory(run)}
                        className={cn(
                          "w-full p-3 rounded-lg border text-left transition-colors duration-200",
                          selectedHistory?.id === run.id
                            ? "bg-workflow-node-input/50 border-workflow-border hover:bg-workflow-node-input"
                            : "bg-workflow-surface/30 border-workflow-border/50 hover:bg-workflow-surface/50"
                        )}
                      >
                        <div className="flex items-center justify-between mb-1">
                          <div className="flex items-center gap-2">
                            {getHistoryStatusIcon(run)}
                            <span className={cn(
                              "font-mono text-sm capitalize"
                            )}>
                              {run.status}
                            </span>
                            <span className="font-mono text-xs text-workflow-text-muted">
                              {formatDuration(run.startedAt, run.finishedAt)}
                            </span>
                          </div>
                          <span className="font-mono text-xs text-workflow-text-muted">
                            {new Date(run.startedAt * 1000).toLocaleString()}
                          </span>
                        </div>
                        {run.finalOutput && (
                          <p className="font-mono text-xs text-workflow-text-muted mt-1 truncate">
                            {run.finalOutput}
                          </p>
                        )}
                      </button>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>
        )
      )}
    </div>
  );
}
