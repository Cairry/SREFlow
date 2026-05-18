"use client";

import { useState, useEffect, useRef } from "react";
import { Clock, Loader2, RotateCcw, FileText, Star, StarOff, Trash2 } from "lucide-react";
import { cn } from "@/lib/utils";
import { ModalDialog } from "./ui/dialog";
import { workflowApi, WorkflowVersion } from "@/lib/api/workflow";

interface WorkflowVersionHistoryDialogProps {
  isOpen: boolean;
  onClose: () => void;
  workflowId: string | null;
  refreshTrigger: number;
  onRestored: (restored: {
    versionNum: number;
    contentHash: string;
    nodes: any[];
    edges: any[];
    name: string;
    description: string;
    currentVersionNum: number;
  }) => void;
}

function VersionContent({
  workflowId,
  refreshTrigger,
  onRestore,
  isRestoring,
  restoringVersion,
  isOpen,
  onSetRelease,
  onDelete,
}: {
  workflowId: string | null;
  refreshTrigger: number;
  onRestore: (version: number) => void;
  isRestoring: boolean;
  restoringVersion: number | null;
  isOpen: boolean;
  onSetRelease: (version: number, isRelease: boolean) => void;
  onDelete: (version: number) => void;
}) {
  const [versions, setVersions] = useState<WorkflowVersion[]>([]);
  const [loading, setLoading] = useState(false);
  const [currentVersionNum, setCurrentVersionNum] = useState<number>(1);

  const loadVersions = () => {
    if (!workflowId) return;
    setLoading(true);
    workflowApi
      .getVersionHistory(workflowId, 50)
      .then((data) => {
        setVersions(data.versions || []);
        setCurrentVersionNum(data.currentVersionNum ?? 1);
      })
      .catch((err) => {
        console.error("Failed to load version history:", err);
        setVersions([]);
      })
      .finally(() => {
        setLoading(false);
      });
  };

  useEffect(() => {
    if (workflowId && isOpen) {
      loadVersions();
    }
  }, [workflowId, refreshTrigger, isOpen]);

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="w-6 h-6 animate-spin text-workflow-text-muted" />
      </div>
    );
  }

  if (!workflowId) {
    return (
      <div className="text-center py-12 text-workflow-text-muted font-mono text-sm">
        Save your workflow first to view version history
      </div>
    );
  }

  if (versions.length === 0) {
    return (
      <div className="text-center py-12">
        <FileText className="w-12 h-12 mx-auto mb-3 text-workflow-text-muted opacity-50" />
        <div className="text-workflow-text-muted font-mono text-sm">
          No version history yet
        </div>
        <div className="text-workflow-text-muted/60 font-mono text-xs mt-1">
          Versions are saved each time you update the workflow
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-2 max-h-[60vh] overflow-y-auto max-w-[80vh]">
      {versions.map((v) => {
        const isCurrent = v.version === currentVersionNum;
        const isThisRestoring = isRestoring && restoringVersion === v.version;

        return (
          <div
            key={v.version}
            className={cn(
              "p-3 rounded-lg border transition-colors duration-200",
              isCurrent
                ? "bg-emerald-500/5 border-emerald-500/30"
                : "bg-workflow-node-input/50 border-workflow-border/50 hover:bg-workflow-node-input hover:border-workflow-border"
            )}
          >
            <div className="flex items-start justify-between gap-2">
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-1">
                  <Clock
                    className={cn(
                      "w-3.5 h-3.5 shrink-0",
                      isCurrent ? "text-emerald-500" : "text-workflow-text-muted"
                    )}
                  />
                  <span
                    className={cn(
                      "font-mono text-sm font-semibold",
                      isCurrent ? "text-emerald-500" : "text-workflow-text"
                    )}
                  >
                    Version {v.version}
                  </span>
                  {isCurrent && (
                    <span className="px-1.5 py-0.5 rounded text-[10px] font-mono font-semibold bg-emerald-500/20 text-emerald-500 border border-emerald-500/30">
                      当前版本
                    </span>
                  )}
                </div>
                <p className="text-xs font-mono text-workflow-text-subtle pl-5">
                  {new Date(v.createdAt).toLocaleString()}
                </p>
              </div>

              <div className="flex items-center gap-1 shrink-0">
                {/* Release 按钮：未 Release 可设置，已 Release 可取消 */}
                <button
                  type="button"
                  onClick={() => onSetRelease(v.version, v.isRelease)}
                  className={cn(
                    "flex items-center gap-1.5 px-2 py-1 rounded-md text-xs font-mono",
                    "border transition-colors duration-200",
                    v.isRelease
                      ? "bg-amber-500/20 border-amber-500/50 text-amber-400 cursor-pointer hover:bg-amber-500/30"
                      : "bg-workflow-surface border-workflow-border text-workflow-text-muted hover:text-amber-400 hover:border-amber-500/50"
                  )}
                  title={v.isRelease ? "取消 Release" : "设置为 Release"}
                >
                  <Star className={cn("w-3 h-3", v.isRelease ? "fill-amber-400" : "")} />
                  Release
                </button>

                {!isCurrent && (
                  <>
                    {/* 前往按钮 */}
                    <button
                      type="button"
                      onClick={() => onRestore(v.version)}
                      disabled={isRestoring}
                      className={cn(
                        "flex items-center gap-1.5 px-2 py-1 rounded-md text-xs font-mono",
                        "border transition-colors duration-200",
                        isRestoring
                          ? "bg-workflow-surface border-workflow-border/30 text-workflow-text-muted/50 cursor-not-allowed"
                          : "bg-workflow-surface border-workflow-border text-workflow-text-muted hover:text-workflow-text hover:border-workflow-border-subtle"
                      )}
                      title={isThisRestoring ? "正在前往..." : "前往此版本"}
                    >
                      {isThisRestoring ? (
                        <Loader2 className="w-3 h-3 animate-spin" />
                      ) : (
                        <RotateCcw className="w-3 h-3" />
                      )}
                      {isThisRestoring ? "正在前往..." : "前往"}
                    </button>

                    {/* 删除按钮 */}
                    <button
                      type="button"
                      onClick={() => onDelete(v.version)}
                      className="flex items-center gap-1 px-2 py-1 rounded-md text-xs font-mono bg-workflow-surface border border-workflow-border/50 text-workflow-text-muted/60 hover:text-red-400 hover:border-red-500/50 transition-colors duration-200"
                      title="删除此版本"
                    >
                      <Trash2 className="w-3 h-3" />
                    </button>
                  </>
                )}
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
}

export function WorkflowVersionHistoryDialog({
  isOpen,
  onClose,
  workflowId,
  refreshTrigger,
  onRestored,
}: WorkflowVersionHistoryDialogProps) {
  const [isRestoring, setIsRestoring] = useState(false);
  const [restoringVersion, setRestoringVersion] = useState<number | null>(null);
  const [showOverlay, setShowOverlay] = useState(false);
  const overlayTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [refreshTriggerInner, setRefreshTriggerInner] = useState(0);

  useEffect(() => {
    return () => {
      if (overlayTimerRef.current) {
        clearTimeout(overlayTimerRef.current);
      }
    };
  }, []);

  const handleRestore = async (version: number) => {
    if (!workflowId || isRestoring) return;

    onClose();

    overlayTimerRef.current = setTimeout(() => {
      setShowOverlay(true);
    }, 50);

    setIsRestoring(true);
    setRestoringVersion(version);

    try {
      const result = await workflowApi.gotoVersion(workflowId, version);

      if (overlayTimerRef.current) {
        clearTimeout(overlayTimerRef.current);
        overlayTimerRef.current = null;
      }
      setShowOverlay(false);
      setIsRestoring(false);
      setRestoringVersion(null);

      onRestored({
        versionNum: result.versionNum,
        contentHash: result.contentHash,
        nodes: result.nodes,
        edges: result.edges,
        name: result.name,
        description: result.description,
        currentVersionNum: result.currentVersionNum,
      });
    } catch (err) {
      console.error("Failed to goto version:", err);
      if (overlayTimerRef.current) {
        clearTimeout(overlayTimerRef.current);
        overlayTimerRef.current = null;
      }
      setShowOverlay(false);
      setIsRestoring(false);
      setRestoringVersion(null);
    }
  };

  const handleSetRelease = async (version: number, isRelease: boolean) => {
    if (!workflowId) return;
    try {
      if (isRelease) {
        await workflowApi.cancelRelease(workflowId, version);
      } else {
        await workflowApi.setRelease(workflowId, version);
      }
      setRefreshTriggerInner(n => n + 1);
    } catch (err) {
      console.error(`Failed to ${isRelease ? "cancel" : "set"} release:`, err);
    }
  };

  const handleDelete = async (version: number) => {
    if (!workflowId) return;
    if (!window.confirm(`确定删除版本 v${version}？此操作不可恢复。`)) return;
    try {
      await workflowApi.deleteVersion(workflowId, version);
      setRefreshTriggerInner(n => n + 1);
    } catch (err) {
      console.error("Failed to delete version:", err);
    }
  };

  return (
    <>
      {/* 全屏前往中遮罩 */}
      {showOverlay && (
        <div className="fixed inset-0 z-[100] flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="bg-workflow-surface border border-workflow-border rounded-xl px-8 py-6 flex flex-col items-center gap-3 shadow-2xl">
            <Loader2 className="w-8 h-8 animate-spin text-emerald-500" />
            <p className="text-workflow-text font-mono text-sm">版本切换中...</p>
            <p className="text-workflow-text-muted font-mono text-xs">
              请稍候，数据更新完成后将自动渲染
            </p>
          </div>
        </div>
      )}

      <ModalDialog
        isOpen={isOpen}
        onClose={onClose}
        title="版本历史"
        description="查看工作流的所有历史版本，可前往任意版本"
        width="lg"
        maxHeight="max-h-[80vh]"
      >
        <VersionContent
          workflowId={workflowId}
          refreshTrigger={refreshTrigger + refreshTriggerInner}
          onRestore={handleRestore}
          isRestoring={isRestoring}
          restoringVersion={restoringVersion}
          isOpen={isOpen}
          onSetRelease={handleSetRelease}
          onDelete={handleDelete}
        />
      </ModalDialog>
    </>
  );
}
