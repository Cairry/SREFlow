"use client";

import React, { useState, useEffect } from "react";
import { Trash2, Loader2 } from "lucide-react";
import { useToast } from "@/hooks/use-toast";
import { cn } from "@/lib/utils";
import { ModalDialog } from "./ui/dialog";
import { workflowApi, Workflow } from "@/lib/api/workflow";

interface LoadWorkflowDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onLoad: (workflowId: string) => void;
}

export function LoadWorkflowDialog({ isOpen, onClose, onLoad }: LoadWorkflowDialogProps) {
  const [workflows, setWorkflows] = useState<Workflow[]>([]);
  const [loading, setLoading] = useState(true);
  const [deleting, setDeleting] = useState<string | null>(null);

  useEffect(() => {
    if (isOpen) {
      fetchWorkflows();
    }
  }, [isOpen]);

  const fetchWorkflows = async () => {
    setLoading(true);
    try {
      const data = await workflowApi.list();
      setWorkflows(data.workflows || []);
    } catch (error) {
      console.error("Failed to fetch workflows:", error);
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async (id: string, e: React.MouseEvent) => {
    e.stopPropagation();
    setDeleting(id);
    try {
      await workflowApi.delete(id);
      setWorkflows(workflows.filter((w) => w.id !== id));
    } catch (error) {
      console.error("Failed to delete workflow:", error);
      alert(`Failed to delete workflow: ${error instanceof Error ? error.message : 'Unknown error'}`);
    } finally {
      setDeleting(null);
    }
  };

  return (
    <ModalDialog
      isOpen={isOpen}
      onClose={onClose}
      title="Load Workflow"
      description="Select a workflow to load"
      width="lg"
      maxHeight="max-h-[80vh]"
    >
      {loading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="w-6 h-6 text-workflow-text-muted animate-spin" />
        </div>
      ) : workflows.length === 0 ? (
        <div className="text-center py-12">
          <p className="text-workflow-text-muted font-mono text-sm">No saved workflows</p>
          <p className="text-workflow-text-subtle text-xs mt-1">Create and save a workflow to see it here</p>
        </div>
      ) : (
        <div className="space-y-2">
          {workflows.map((workflow) => (
            <div
              key={workflow.id}
              className="w-full text-left p-3 bg-workflow-node-input/50 hover:bg-workflow-node-input border border-workflow-border/50 hover:border-workflow-border rounded-lg transition-all duration-200 group"
            >
              <div className="flex items-start justify-between">
                <button
                  type="button"
                  onClick={() => workflow.id && onLoad(workflow.id)}
                  className="flex-1 min-w-0 text-left"
                >
                  <div className="pr-2">
                    <h3 className="font-mono font-medium text-workflow-text text-sm">{workflow.name}</h3>
                    {workflow.description && (
                      <p className="text-workflow-text-muted text-xs mt-1 line-clamp-2">{workflow.description}</p>
                    )}
                    <div className="flex items-center gap-4 mt-2 text-xs text-workflow-text-subtle">
                      <span>Created: {new Date(workflow.createdAt).toLocaleDateString()}</span>
                      <span>Updated: {new Date(workflow.updatedAt).toLocaleDateString()}</span>
                    </div>
                  </div>
                </button>
                <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity duration-200">
                  <button
                    type="button"
                    onClick={(e) => workflow.id && handleDelete(workflow.id, e)}
                    disabled={deleting === workflow.id}
                    className="p-1.5 text-workflow-text-muted hover:text-rose-500 hover:bg-rose-500/10 rounded transition-colors duration-200 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    {deleting === workflow.id ? (
                      <Loader2 className="w-4 h-4 animate-spin" />
                    ) : (
                      <Trash2 className="w-4 h-4" />
                    )}
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </ModalDialog>
  );
}
