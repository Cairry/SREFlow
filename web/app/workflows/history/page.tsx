"use client";

import { useState, useEffect } from "react";
import { Clock, CheckCircle2, XCircle, Loader2, ArrowLeft, Trash2 } from "lucide-react";
import Link from "next/link";
import { cn } from "@/lib/utils";
import { workflowApi, RunHistoryItem } from "@/lib/api/workflow";

export default function RunHistoryPage() {
  const [history, setHistory] = useState<RunHistoryItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [deleting, setDeleting] = useState<string | null>(null);

  useEffect(() => {
    loadHistory();
  }, []);

  const loadHistory = async () => {
    setLoading(true);
    try {
      const data = await workflowApi.listRunHistory(100);
      setHistory(data.history || []);
    } catch (error) {
      console.error("Failed to load run history:", error);
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async (executionId: string) => {
    setDeleting(executionId);
    try {
      await workflowApi.deleteRunHistory(executionId);
      await loadHistory();
    } catch (error) {
      console.error("Failed to delete run history:", error);
    } finally {
      setDeleting(null);
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case "completed":
        return <CheckCircle2 className="w-4 h-4 text-emerald-500" />;
      case "failed":
        return <XCircle className="w-4 h-4 text-rose-500" />;
      default:
        return <Loader2 className="w-4 h-4 text-amber-500 animate-spin" />;
    }
  };

  const getDuration = (startedAt: number, finishedAt: number | null) => {
    if (!finishedAt) return "Running...";
    const duration = (finishedAt - startedAt) / 1000;
    return `${duration.toFixed(2)}s`;
  };

  return (
    <div className="min-h-screen bg-workflow-bg">
      <div className="border-b border-workflow-border bg-workflow-surface">
        <div className="container mx-auto px-4 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <Link
                href="/workflows"
                className="flex items-center gap-2 text-workflow-text hover:text-workflow-text-muted transition-colors duration-200"
              >
                <ArrowLeft className="w-4 h-4" />
                Back to Workflows
              </Link>
              <div className="flex items-center gap-2">
                <Clock className="w-5 h-5 text-workflow-text-muted" />
                <h1 className="text-xl font-semibold text-workflow-text font-mono">Run History</h1>
              </div>
            </div>
            <button
              onClick={loadHistory}
              disabled={loading}
              className="flex items-center gap-2 px-3 py-1.5 bg-workflow-node-input border border-workflow-border rounded-lg text-sm text-workflow-text font-mono hover:bg-workflow-node-input/80 transition-colors duration-200 disabled:opacity-50"
            >
              {loading ? <Loader2 className="w-4 h-4 animate-spin" /> : <Clock className="w-4 h-4" />}
              Refresh
            </button>
          </div>
        </div>
      </div>

      <div className="container mx-auto px-4 py-6">
        {loading ? (
          <div className="flex items-center justify-center py-20">
            <Loader2 className="w-8 h-8 animate-spin text-workflow-text-muted" />
          </div>
        ) : history.length === 0 ? (
          <div className="text-center py-20">
            <Clock className="w-16 h-16 mx-auto mb-4 text-workflow-text-muted opacity-50" />
            <h3 className="text-lg font-medium text-workflow-text mb-2">No run history yet</h3>
            <p className="text-workflow-text-muted font-mono">Run some workflows to see their execution history here.</p>
          </div>
        ) : (
          <div className="space-y-3">
            {history.map((run) => (
              <div
                key={run.id}
                className="bg-workflow-surface border border-workflow-border rounded-lg p-4 hover:border-workflow-border-subtle transition-colors duration-200"
              >
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <div className="flex items-center gap-3 mb-2">
                      {getStatusIcon(run.status)}
                      <div>
                        <h3 className="font-medium text-workflow-text font-mono">{run.workflowName}</h3>
                        <p className="text-sm text-workflow-text-muted font-mono capitalize">{run.status}</p>
                      </div>
                    </div>
                    <div className="grid grid-cols-3 gap-4 text-sm font-mono">
                      <div>
                        <span className="text-workflow-text-muted">Started:</span>
                        <span className="ml-2 text-workflow-text">
                          {new Date(run.startedAt * 1000).toLocaleString()}
                        </span>
                      </div>
                      <div>
                        <span className="text-workflow-text-muted">Duration:</span>
                        <span className="ml-2 text-workflow-text">
                          {getDuration(run.startedAt, run.finishedAt)}
                        </span>
                      </div>
                      <div>
                        <span className="text-workflow-text-muted">Workflow ID:</span>
                        <span className="ml-2 text-workflow-text font-mono text-xs">
                          {run.workflowId.slice(0, 8)}...
                        </span>
                      </div>
                    </div>
                    {run.finalOutput && (
                      <div className="mt-3 p-2 bg-workflow-node-input rounded text-xs font-mono text-workflow-text-muted">
                        <span className="text-workflow-text-subtle">Output:</span>
                        <span className="ml-2 block">
                          {run.finalOutput}
                        </span>
                      </div>
                    )}
                  </div>
                  <button
                    onClick={() => handleDelete(run.id)}
                    disabled={deleting === run.id}
                    className="ml-4 p-2 text-workflow-text-muted hover:text-rose-500 transition-colors duration-200 disabled:opacity-50"
                    title="Delete run history"
                  >
                    {deleting === run.id ? (
                      <Loader2 className="w-4 h-4 animate-spin" />
                    ) : (
                      <Trash2 className="w-4 h-4" />
                    )}
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
