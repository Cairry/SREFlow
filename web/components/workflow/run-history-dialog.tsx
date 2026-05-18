"use client";

import { useState, useEffect } from "react";
import { CheckCircle2, XCircle, Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";
import { ModalDialog } from "./ui/dialog";
import { workflowApi, RunHistoryItem } from "@/lib/api/workflow";

// 后端返回的NodeResult结构
interface BackendNodeResult {
  outputs: { [key: string]: any };
  error?: string;
  error_type?: string;
  success?: boolean;
  latency?: string | number;
  usage?: string;
  body?: string;
  headers?: string;
  statusCode?: string;
  url?: string;
  output?: string;
  request?: { body?: string };
  responseHeaders?: Record<string, string>;
  next?: string[];
}

// 前端显示用的NodeResult结构
interface DisplayNodeResult {
  nodeId: string;
  nodeType: string;
  output: any;
  error?: string;
  timestamp: number;
}

interface RunHistoryItem {
  id: string;
  workflowId: string;
  workflowName: string;
  status: string;
  finalOutput: string | null;
  startedAt: number;
  finishedAt: number | null;
  createdAt: number;
  updatedAt: number;
  nodeResults?: { [key: string]: BackendNodeResult };
  outputs?: { [key: string]: any };
  variables?: { [key: string]: any };
}

interface RunHistoryDialogProps {
  isOpen: boolean;
  onClose: () => void;
  workflowId: string | null;
  onSelectRun: (run: RunHistoryItem) => void;
}

// RunDetails 组件，用于显示运行详情
export function RunDetails({ run, onClose }: { run: RunHistoryItem; onClose: () => void }) {
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set());

  const toggleNode = (nodeId: string) => {
    const newExpanded = new Set(expandedNodes);
    if (newExpanded.has(nodeId)) {
      newExpanded.delete(nodeId);
    } else {
      newExpanded.add(nodeId);
    }
    setExpandedNodes(newExpanded);
  };

  const formatDuration = (started: number, finished: number | null) => {
    if (!finished) return 'Running...';
    const duration = (finished - started) * 1000;
    if (duration < 1000) return `${duration}ms`;
    return `${(duration / 1000).toFixed(2)}s`;
  };

  return (
    <div className="space-y-4">
      {/* 基本信息 */}
      <div className="bg-workflow-node-input/30 rounded-lg">
        <h3 className="font-mono text-sm font-semibold text-workflow-text mb-3">Run Information</h3>
        <div className="grid grid-cols-2 gap-3 text-xs font-mono">
          <div>
            <span className="text-workflow-text-muted">Run ID:</span>
            <span className="text-workflow-text ml-2">{run.id.slice(0, 8)}...</span>
          </div>
          <div>
            <span className="text-workflow-text-muted">Status:</span>
            <span className={`ml-2 capitalize ${run.status === 'completed' ? 'text-green-500' : run.status === 'failed' ? 'text-red-500' : 'text-blue-500'}`}>
              {run.status}
            </span>
          </div>
          <div>
            <span className="text-workflow-text-muted">Started:</span>
            <span className="text-workflow-text ml-2">{new Date(run.startedAt * 1000).toLocaleString()}</span>
          </div>
          <div>
            <span className="text-workflow-text-muted">Duration:</span>
            <span className="text-workflow-text ml-2">{formatDuration(run.startedAt, run.finishedAt)}</span>
          </div>
        </div>
        {run.finalOutput && (
          <div className="mt-3">
            <span className="text-workflow-text-muted text-xs">Final Output:</span>
            <div className="mt-1 p-2 bg-workflow-surface rounded border border-workflow-border/50 font-mono text-xs text-workflow-text break-all">
              {run.finalOutput}
            </div>
          </div>
        )}
      </div>

      {/* 节点结果 */}
      {(run.nodeResults && Object.keys(run.nodeResults).length > 0) || (run.outputs && Object.keys(run.outputs).length > 0) && (
        <div>
          <h3 className="font-mono text-sm font-semibold text-workflow-text mb-3">Node Results</h3>
          <div className="space-y-2">
            {run.nodeResults && Object.keys(run.nodeResults).length > 0 && Object.entries(run.nodeResults).map(([nodeId, result]) => (
              <div key={nodeId} className="border border-workflow-border/50 rounded-lg overflow-hidden">
                <button
                  type="button"
                  onClick={() => toggleNode(nodeId)}
                  className="w-full p-3 bg-workflow-node-input/30 hover:bg-workflow-node-input/50 transition-colors duration-200 text-left"
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      {result.error ? (
                        <XCircle className="w-4 h-4 text-red-500" />
                      ) : (
                        <CheckCircle2 className="w-4 h-4 text-green-500" />
                      )}
                      <span className="font-mono text-sm text-workflow-text">Node</span>
                      <span className="font-mono text-xs text-workflow-text-muted">({nodeId.slice(0, 8)}...)</span>
                    </div>
                    <span className="text-xs text-workflow-text-muted">
                      {expandedNodes.has(nodeId) ? '▼' : '▶'}
                    </span>
                  </div>
                </button>

                {expandedNodes.has(nodeId) && (
                  <div className="p-3 border-t border-workflow-border/50 bg-workflow-surface/50">
                    {result.error ? (
                      <div className="space-y-2">
                        <div className="text-xs font-mono text-red-500">Error:</div>
                        <div className="p-2 bg-red-500/10 border border-red-500/30 rounded font-mono text-xs text-red-400 break-all">
                          {result.error}
                        </div>
                      </div>
                    ) : (
                      <div className="space-y-2">
                        <div className="text-xs font-mono text-workflow-text-muted">Output:</div>
                        <div className="p-2 bg-workflow-node-input/30 border border-workflow-border/30 rounded font-mono text-xs text-workflow-text break-all max-h-40 overflow-y-auto">
                          {result.outputs && Object.keys(result.outputs).length > 0 ? (
                            <div className="space-y-1">
                              {Object.entries(result.outputs).map(([key, value]) => (
                                <div key={key} className="border-b border-workflow-border/30 pb-1 last:border-b-0">
                                  <div className="text-xs font-semibold text-workflow-text-muted mb-1">{key}:</div>
                                  <div className="text-xs text-workflow-text">
                                    {typeof value === 'string' ? value : JSON.stringify(value, null, 2)}
                                  </div>
                                </div>
                              ))}
                            </div>
                          ) : (
                            <div className="text-xs text-workflow-text-muted italic">No output</div>
                          )}
                        </div>
                      </div>
                    )}
                    <div className="mt-2 text-xs font-mono text-workflow-text-muted">
                      {new Date(run.startedAt * 1000).toLocaleString()}
                    </div>
                  </div>
                )}
              </div>
            ))}
            {run.outputs && Object.keys(run.outputs).length > 0 && Object.entries(run.outputs).map(([nodeId, nodeOutput]) => (
              <div key={nodeId} className="border border-workflow-border/50 rounded-lg overflow-hidden">
                <button
                  type="button"
                  onClick={() => toggleNode(nodeId)}
                  className="w-full p-3 bg-workflow-node-input/30 hover:bg-workflow-node-input/50 transition-colors duration-200 text-left"
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      {nodeOutput?.error ? (
                        <XCircle className="w-4 h-4 text-red-500" />
                      ) : (
                        <CheckCircle2 className="w-4 h-4 text-green-500" />
                      )}
                      <span className="font-mono text-sm text-workflow-text">{nodeId}</span>
                    </div>
                    <span className="text-xs text-workflow-text-muted">
                      {expandedNodes.has(nodeId) ? '▼' : '▶'}
                    </span>
                  </div>
                </button>
                {expandedNodes.has(nodeId) && (
                  <div className="p-3 border-t border-workflow-border/50 bg-workflow-surface/50">
                    {nodeOutput?.error ? (
                      <div className="space-y-2">
                        <div className="text-xs font-mono text-red-500">Error:</div>
                        <div className="p-2 bg-red-500/10 border border-red-500/30 rounded font-mono text-xs text-red-400 break-all">
                          {nodeOutput.error}
                        </div>
                      </div>
                    ) : nodeId.startsWith('output-') && (nodeOutput?.body || nodeOutput?.url || nodeOutput?.request || nodeOutput?.requestbody) ? (
                      <div className="space-y-3">
                        {nodeOutput.url && (
                          <div>
                            <div className="text-xs font-mono text-workflow-text-muted mb-1">URL</div>
                            <div className="p-2 bg-workflow-node-input/30 border border-workflow-border/30 rounded font-mono text-xs text-workflow-text break-all">
                              {nodeOutput.url}
                            </div>
                          </div>
                        )}
                        {nodeOutput.requestbody && (
                          <div>
                            <div className="text-xs font-mono text-workflow-text-muted mb-1">Request Body</div>
                            <div className="p-2 bg-workflow-node-input/30 border border-workflow-border/30 rounded font-mono text-xs text-workflow-text break-all max-h-32 overflow-y-auto">
                              {(() => {
                                try { return JSON.stringify(JSON.parse(nodeOutput.requestbody), null, 2); }
                                catch { return String(nodeOutput.requestbody); }
                              })()}
                            </div>
                          </div>
                        )}
                        {nodeOutput.statusCode && (
                          <div className="flex items-center gap-2">
                            <span className="text-xs font-mono text-workflow-text-muted">Status:</span>
                            <span className={`text-xs font-mono font-semibold ${Number(nodeOutput.statusCode) >= 400 ? 'text-red-500' : 'text-green-500'}`}>
                              {nodeOutput.statusCode}
                            </span>
                          </div>
                        )}
                        {nodeOutput.output && (
                          <div>
                            <div className="text-xs font-mono text-workflow-text-muted mb-1">Response Body</div>
                            <div className="p-2 bg-workflow-node-input/30 border border-workflow-border/30 rounded font-mono text-xs text-workflow-text break-all max-h-40 overflow-y-auto">
                              {(() => {
                                try { return JSON.stringify(JSON.parse(nodeOutput.output), null, 2); }
                                catch { return String(nodeOutput.output); }
                              })()}
                            </div>
                          </div>
                        )}
                        {nodeOutput.responseHeaders && (
                          <div>
                            <div className="text-xs font-mono text-workflow-text-muted mb-1">Response Headers</div>
                            <div className="p-2 bg-workflow-node-input/30 border border-workflow-border/30 rounded font-mono text-xs text-workflow-text break-all max-h-20 overflow-y-auto">
                              {(() => {
                                try { return JSON.stringify(nodeOutput.responseHeaders, null, 2); }
                                catch { return String(nodeOutput.responseHeaders); }
                              })()}
                            </div>
                          </div>
                        )}
                      </div>
                    ) : (
                      <div className="space-y-2">
                        <div className="text-xs font-mono text-workflow-text-muted">Output:</div>
                        <div className="p-2 bg-workflow-node-input/30 border border-workflow-border/30 rounded font-mono text-xs text-workflow-text break-all max-h-40 overflow-y-auto">
                          {typeof nodeOutput === 'object' ? (
                            Object.entries(nodeOutput).map(([key, val]) => (
                              <div key={key} className="border-b border-workflow-border/30 pb-1 last:border-b-0">
                                <div className="text-xs font-semibold text-workflow-text-muted mb-1">{key}:</div>
                                <div className="text-xs text-workflow-text whitespace-pre-wrap">
                                  {typeof val === 'object' ? JSON.stringify(val, null, 2) : String(val ?? '')}
                                </div>
                              </div>
                            ))
                          ) : (
                            <div className="text-xs text-workflow-text whitespace-pre-wrap">{String(nodeOutput ?? '')}</div>
                          )}
                        </div>
                      </div>
                    )}
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* 变量 */}
      {run.variables && Object.keys(run.variables).length > 0 && (
        <div>
          <h3 className="font-mono text-sm font-semibold text-workflow-text mb-3">Variables</h3>
          <div className="bg-workflow-node-input/30 rounded-lg p-3 border border-workflow-border/50">
            <div className="space-y-2">
              {Object.entries(run.variables).map(([key, value]) => (
                <div key={key} className="grid grid-cols-3 gap-2 text-xs font-mono">
                  <span className="text-workflow-text-muted truncate">{key}:</span>
                  <span className="col-span-2 text-workflow-text break-all">
                    {typeof value === 'string' ? value : JSON.stringify(value)}
                  </span>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

// RunHistoryContent 组件，用于显示历史记录内容
export function RunHistoryContent({ 
  workflowId, 
  onSelectRun 
}: { 
  workflowId: string | null; 
  onSelectRun: (run: RunHistoryItem) => void;
}) {
  const [history, setHistory] = useState<RunHistoryItem[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (workflowId) {
      setLoading(true);
      workflowApi.getRunHistory(workflowId, 50)
        .then((data) => {
          setHistory(data.history || []);
        })
        .catch(console.error)
        .finally(() => setLoading(false));
    }
  }, [workflowId]);

  return (
    <div className="space-y-2">
      {loading ? (
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
      ) : (
        history.map((run) => (
          <button
            key={run.id}
            type="button"
            onClick={() => onSelectRun(run)}
            className={cn(
              "w-full p-3 rounded-lg border text-left transition-colors duration-200",
              "bg-workflow-node-input/50 border-workflow-border/50 hover:bg-workflow-node-input hover:border-workflow-border"
            )}
          >
            <div className="flex items-center justify-between mb-1">
              <div className="flex items-center gap-2">
                {run.status === "completed" ? (
                  <CheckCircle2 className="w-4 h-4 text-green-500" />
                ) : run.status === "failed" ? (
                  <XCircle className="w-4 h-4 text-red-500" />
                ) : (
                  <Loader2 className="w-4 h-4 text-blue-500 animate-spin" />
                )}
                <span className="font-mono text-sm text-workflow-text capitalize">
                  {run.status}
                </span>
              </div>
              <span className="font-mono text-xs text-workflow-text-muted">
                {new Date(run.startedAt * 1000).toLocaleString()}
              </span>
            </div>
            {run.finalOutput && (
              <p className="font-mono text-xs text-workflow-text-muted mt-1">
                {run.finalOutput}
              </p>
            )}
          </button>
        ))
      )}
    </div>
  );
}

export function RunHistoryDialog({
  isOpen,
  onClose,
  workflowId,
  onSelectRun,
}: RunHistoryDialogProps) {
  return (
    <ModalDialog
      isOpen={isOpen}
      onClose={onClose}
      title="Run History"
      description={workflowId ? "View execution history" : "Save workflow to see history"}
      width="lg"
      maxHeight="max-h-[80vh]"
    >
      <RunHistoryContent
        workflowId={workflowId}
        onSelectRun={(run) => {
          onSelectRun(run);
          onClose();
        }}
      />
    </ModalDialog>
  );
}
