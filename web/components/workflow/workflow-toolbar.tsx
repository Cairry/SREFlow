"use client";

import React, { useState, useEffect } from "react"
import { useRouter } from "next/navigation"

import { cn } from "@/lib/utils";
import {
  LayoutGrid,
  Trash2,
  Clock,
  Code2,
  Play,
  Loader2,
  Square,
  Save,
  FolderOpen,
  Plus,
  Sparkles,
  Workflow,
  Terminal,
  History,
  ArrowLeft,
  PanelTop,
  PanelTopClose,
} from "lucide-react";
import { isNavHidden, toggleNav } from "@/lib/nav-toggle";
import type { WorkflowNode, WorkflowNodeType } from "@/lib/workflow-types";
import type { Edge } from "@xyflow/react";

interface WorkflowToolbarProps {
  workflowName: string;
  onExecute: () => void;
  onStop: () => void;
  onSave: () => void;
  onNew: () => void;
  onClear: () => void;
  onOpenHistory: () => void;
  onOpenVersionHistory: () => void;
  onSelectTemplate: (nodes: WorkflowNode[], edges: Edge[], name: string) => void;
  onAddNode: (nodeType: WorkflowNodeType, mcpData?: { serverId: string; serverName: string; toolName: string; description: string }) => void;
  onAutoLayout: () => void;
  onOpenApiCall: () => void;
  isExecuting: boolean;
  isStopping: boolean;
  isSaving: boolean;
  hasChanges: boolean;
  showOutput: boolean;
  showAddNode: boolean;
  onToggleAddNode: (buttonRect?: DOMRect) => void;
}

function NavToggleButton() {
  const [hidden, setHidden] = useState(isNavHidden);

  useEffect(() => {
    const handler = () => setHidden(isNavHidden());
    window.addEventListener("storage", handler);
    window.addEventListener("flowforge-nav-toggle", handler);
    return () => {
      window.removeEventListener("storage", handler);
      window.removeEventListener("flowforge-nav-toggle", handler);
    };
  }, []);

  return (
    <IconButton
      icon={hidden ? <PanelTop className="w-4 h-4" /> : <PanelTopClose className="w-4 h-4" />}
      onClick={toggleNav}
      title={hidden ? "Show navigation" : "Hide navigation"}
    />
  );
}

function ButtonGroup({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex items-center bg-workflow-surface border border-workflow-border rounded-lg overflow-hidden">
      {children}
    </div>
  );
}

function IconButton({
  icon,
  onClick,
  title,
  active,
  disabled,
  variant = "default",
}: {
  icon: React.ReactNode;
  onClick: () => void;
  title: string;
  active?: boolean;
  disabled?: boolean;
  variant?: "default" | "danger";
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      title={title}
      className={cn(
        "p-2.5 transition-colors duration-200 border-r border-workflow-border last:border-r-0",
        disabled && "opacity-40 cursor-not-allowed",
        active && "bg-workflow-surface-hover text-workflow-text",
        !active && !disabled && variant === "default" && "text-workflow-text-muted hover:text-workflow-text hover:bg-workflow-surface-hover",
        !active && !disabled && variant === "danger" && "text-workflow-text-muted hover:text-rose-400 hover:bg-workflow-surface-hover"
      )}
    >
      {icon}
    </button>
  );
}

export function WorkflowToolbar({
  workflowName,
  onExecute,
  onStop,
  onSave,
  onOpenHistory,
  onOpenVersionHistory,
  onAddNode,
  onAutoLayout,
  onOpenApiCall,
  isExecuting,
  isStopping,
  isSaving,
  hasChanges,
  showOutput,
  showAddNode,
  onToggleAddNode,
}: WorkflowToolbarProps) {
  const router = useRouter();

  return (
    <div className="px-4 pt-3 pb-1 flex justify-center">
      <div className="bg-workflow-surface border border-workflow-border rounded-xl px-4 h-14 flex items-center shadow-sm w-fit">
        <div className="flex items-center gap-6">

          {/* Back button */}
          <button
            type="button"
            onClick={() => router.push("/workflows")}
            className="flex items-center gap-1.5 text-workflow-text-muted hover:text-workflow-text transition-colors duration-200 shrink-0"
            title="Back to Workflows"
          >
            <ArrowLeft className="w-4 h-4" />
            <span className="font-mono text-xs">Back</span>
          </button>

          <div className="h-5 w-px bg-workflow-border shrink-0" />

          {/* Workflow name display */}
          <div className="flex items-center gap-2 shrink-0">
            <Workflow className="w-4 h-4 text-workflow-text-muted" />
            <span className="text-workflow-text font-mono text-sm font-medium truncate max-w-48">
              {workflowName}
            </span>
            {hasChanges && (
              <span className="w-1.5 h-1.5 bg-amber-500 rounded-full" title="Unsaved changes" />
            )}
          </div>

          {/* Action button groups */}
          <div className="flex items-center gap-2">
          {/* Add Node */}
          <div className="relative">
            <ButtonGroup>
              <IconButton
                icon={<Plus className="w-4 h-4" />}
                onClick={() => {
                  const btn = document.querySelector('[title="Add node"]');
                  const rect = btn?.getBoundingClientRect();
                  onToggleAddNode(rect);
                }}
                title="Add node"
                active={showAddNode}
              />
            </ButtonGroup>
          </div>

          {/* Auto Layout */}
          <ButtonGroup>
            <IconButton
              icon={<Sparkles className="w-4 h-4" />}
              onClick={onAutoLayout}
              title="Auto layout"
            />
          </ButtonGroup>

          {/* Save */}
          <ButtonGroup>
            <IconButton
              icon={isSaving ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
              onClick={onSave}
              title="Save workflow"
              disabled={isSaving}
            />
          </ButtonGroup>

          {/* Version History */}
          <ButtonGroup>
            <IconButton
              icon={<History className="w-4 h-4" />}
              onClick={onOpenVersionHistory}
              title="Version history"
            />
          </ButtonGroup>

          {/* API Call */}
          <ButtonGroup>
            <IconButton
              icon={<Terminal className="w-4 h-4" />}
              onClick={onOpenApiCall}
              title="API 调用"
            />
          </ButtonGroup>

          {/* Nav Toggle */}
          <ButtonGroup>
            <NavToggleButton />
          </ButtonGroup>
        </div>

          {/* Run/Stop button */}
          <div className="flex items-center gap-2">
            <ButtonGroup>
              <div className="flex items-center">
                {isExecuting ? (
                  <button
                    type="button"
                    onClick={onStop}
                    disabled={isStopping}
                    className={cn(
                      "flex items-center gap-2 px-4 py-2 font-mono text-sm transition-all duration-200",
                      "bg-rose-600 text-white hover:bg-rose-500",
                      isStopping && "opacity-60 cursor-not-allowed"
                    )}
                  >
                    {isStopping ? (
                      <Loader2 className="w-4 h-4 animate-spin" />
                    ) : (
                      <Square className="w-4 h-4" />
                    )}
                    {isStopping ? "Stopping..." : "Stop"}
                  </button>
                ) : (
                  <button
                    type="button"
                    onClick={onExecute}
                    disabled={isExecuting}
                    className="flex items-center gap-2 px-4 py-2 font-mono text-sm bg-foreground text-background hover:bg-foreground/90 transition-all duration-200"
                  >
                    <Play className="w-4 h-4" />
                    Run
                  </button>
                )}
              </div>
              <IconButton
                icon={<Clock className="w-4 h-4" />}
                onClick={onOpenHistory}
                title="Run history"
              />
            </ButtonGroup>
          </div>

        </div>

      </div>
    </div>
  );
}
