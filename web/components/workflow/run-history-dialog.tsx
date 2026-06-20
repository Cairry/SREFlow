"use client";

import { useState, useEffect } from "react";
import { CheckCircle2, XCircle, Loader2, Ban } from "lucide-react";
import { cn } from "@/lib/utils";
import { workflowApi } from "@/lib/api/workflow";

interface RunHistoryItem {
  id: string;
  workflowId: string;
  workflowName: string;
  status: string;
  finalOutput: string | null;
  outputs?: Record<string, unknown>;
  variables?: Record<string, unknown>;
  startedAt: number;
  finishedAt: number | null;
  createdAt: number;
  updatedAt: number;
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

  const getStatusIcon = (status: string) => {
    switch (status) {
      case "completed":
        return <CheckCircle2 className="w-4 h-4 text-green-500" />;
      case "failed":
        return <XCircle className="w-4 h-4 text-red-500" />;
      case "cancelled":
        return <Ban className="w-4 h-4 text-stone-400" />;
      case "running":
        return <Loader2 className="w-4 h-4 text-blue-500 animate-spin" />;
      default:
        return <Loader2 className="w-4 h-4 text-stone-400 animate-spin" />;
    }
  };

  const formatDuration = (started: number, finished: number | null) => {
    if (!finished) return 'Running...';
    const duration = (finished - started) * 1000;
    if (duration < 1000) return `${duration}ms`;
    return `${(duration / 1000).toFixed(2)}s`;
  };

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
                {getStatusIcon(run.status)}
                <span className={cn(
                  "font-mono text-sm capitalize",
                  run.status === 'completed' ? 'text-green-500' :
                  run.status === 'failed' ? 'text-red-500' :
                  run.status === 'cancelled' ? 'text-stone-400' :
                  'text-blue-500'
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
        ))
      )}
    </div>
  );
}
