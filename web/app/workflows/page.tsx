"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { ArrowRight, Workflow as WorkflowIcon, Plus, Trash2, LayoutGrid, Search, Edit } from "lucide-react";
import { workflowApi, Workflow } from "@/lib/api/workflow";

const WORKSPACE_STORAGE_KEY = "flowforge-current-workspace-id";

interface Template {
  id: string;
  name: string;
  description: string;
  updatedAt: string;
  icon: React.ReactNode;
  workflowData: {
    nodes: any[];
    edges: any[];
  };
}

export default function WorkflowsPage() {
  const [searchTerm, setSearchTerm] = useState<string>("");
  const [templates, setTemplates] = useState<Template[]>([]);
  const [allWorkflows, setAllWorkflows] = useState<any[]>([]);
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [showEditDialog, setShowEditDialog] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [newTemplate, setNewTemplate] = useState({
    name: "",
    description: "",
  });
  const [editingTemplate, setEditingTemplate] = useState({
    name: "",
    description: "",
  });
  const [currentWorkspaceId, setCurrentWorkspaceId] = useState<string>("");
  const [noWorkspace, setNoWorkspace] = useState(false);

  // Load workflows from API
  useEffect(() => {
    loadCurrentWorkspace();
    loadWorkflows();
  }, []);

  const loadCurrentWorkspace = () => {
    if (typeof window !== "undefined") {
      const savedWorkspaceId = localStorage.getItem(WORKSPACE_STORAGE_KEY);
      if (savedWorkspaceId) {
        setCurrentWorkspaceId(savedWorkspaceId);
      } else {
        setNoWorkspace(true);
      }
    }
  };

  const loadWorkflows = async () => {
    try {
      const data = await workflowApi.list();
      const workflows: Workflow[] = data.workflows || [];
      setAllWorkflows(workflows as any[]);

      const convertedTemplates = workflows.map((workflow) => ({
        id: workflow.id,
        name: workflow.name,
        description: workflow.description,
        updatedAt: workflow.updatedAt,
        icon: <WorkflowIcon className="w-6 h-6" />,
        workflowData: {
          nodes: workflow.nodes || [],
          edges: workflow.edges || [],
        },
      }));

      setTemplates(convertedTemplates);
    } catch (error) {
      console.error("Failed to load workflows:", error);
    }
  };

  // Reload workflows when workspace changes
  useEffect(() => {
    if (currentWorkspaceId) {
      setNoWorkspace(false);
      const workspaceWorkflows = allWorkflows.filter((w: any) => w.workspaceId === currentWorkspaceId);
      const convertedTemplates = workspaceWorkflows.map((workflow: any) => ({
        id: workflow.id,
        name: workflow.name,
        description: workflow.description,
        updatedAt: workflow.updatedAt,
        icon: <WorkflowIcon className="w-6 h-6" />,
        workflowData: {
          nodes: workflow.nodes || [],
          edges: workflow.edges || [],
        },
      }));
      setTemplates(convertedTemplates);
    }
  }, [currentWorkspaceId, allWorkflows]);

  const filteredTemplates = templates.filter(template =>
    template.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    template.description.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const handleUseTemplate = (template: Template) => {
    // Store template data in sessionStorage
    sessionStorage.setItem('selectedTemplate', JSON.stringify({
      name: template.name,
      nodes: template.workflowData.nodes,
      edges: template.workflowData.edges,
    }));
  };

  const handleCreateTemplate = async () => {
    const workspaceId = localStorage.getItem(WORKSPACE_STORAGE_KEY);
    if (!workspaceId) {
      alert("Please select or create a workspace first");
      return;
    }

    try {
      const result = await workflowApi.create({
        name: newTemplate.name,
        description: newTemplate.description,
        nodes: [],
        edges: [],
      });

      setShowCreateDialog(false);
      setNewTemplate({ name: "", description: "" });

      const newWorkflow: Workflow = {
        ...result,
        workspaceId: result.workspaceId || workspaceId,
        nodes: result.nodes || [],
        edges: result.edges || [],
      } as Workflow;

      setAllWorkflows(prev => [newWorkflow as any, ...prev]);

      const newTemplateData: Template = {
        id: result.id,
        name: result.name,
        description: result.description,
        updatedAt: result.updatedAt || new Date().toISOString(),
        icon: <WorkflowIcon className="w-6 h-6" />,
        workflowData: {
          nodes: newWorkflow.nodes || [],
          edges: newWorkflow.edges || [],
        },
      };
      setTemplates(prev => [newTemplateData, ...prev]);
    } catch (error) {
      console.error("Failed to create template:", error);
    }
  };

  const handleDeleteClick = (id: string) => {
    setDeletingId(id);
    setShowDeleteDialog(true);
  };

  const handleConfirmDelete = async () => {
    if (!deletingId) return;

    try {
      await workflowApi.delete(deletingId);
      loadWorkflows();
      setShowDeleteDialog(false);
      setDeletingId(null);
    } catch (error) {
      console.error("Failed to delete template:", error);
    }
  };

  const handleCancelDelete = () => {
    setShowDeleteDialog(false);
    setDeletingId(null);
  };

  const handleEditClick = (template: Template) => {
    setEditingId(template.id);
    setEditingTemplate({
      name: template.name,
      description: template.description,
    });
    setShowEditDialog(true);
  };

  const handleUpdateWorkflow = async () => {
    if (!editingId) return;

    const workspaceId = localStorage.getItem(WORKSPACE_STORAGE_KEY);
    if (!workspaceId) {
      alert("Please select or create a workspace first");
      return;
    }

    try {
      const existingWorkflow = await workflowApi.getById(editingId);

      const result = await workflowApi.update(editingId, {
        name: editingTemplate.name,
        description: editingTemplate.description,
        nodes: existingWorkflow.nodes || [],
        edges: existingWorkflow.edges || [],
      });

      setShowEditDialog(false);
      setEditingId(null);
      setEditingTemplate({ name: "", description: "" });

      setTemplates(prev => prev.map(template =>
        template.id === editingId
          ? {
              ...template,
              name: result.name,
              description: result.description,
              updatedAt: result.updatedAt,
            }
          : template
      ));

      setAllWorkflows(prev => prev.map(workflow =>
        workflow.id === editingId
          ? {
              ...workflow,
              name: result.name,
              description: result.description,
              updatedAt: result.updatedAt,
            }
          : workflow
      ));
    } catch (error) {
      console.error("Failed to update workflow:", error);
      alert(`Failed to update workflow: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  };

  const handleCancelEdit = () => {
    setShowEditDialog(false);
    setEditingId(null);
    setEditingTemplate({ name: "", description: "" });
  };

  if (noWorkspace) {
    return (
      <div className="min-h-screen bg-background p-6">
        <div className="max-w-7xl mx-auto">
          <div className="flex items-center justify-center min-h-[60vh]">
            <div className="text-center">
              <div className="flex justify-center mb-4">
                <div className="p-4 rounded-full bg-muted">
                  <LayoutGrid className="w-12 h-12 text-muted-foreground" />
                </div>
              </div>
              <h2 className="text-2xl font-semibold mb-2">No Workspace Selected</h2>
              <p className="text-muted-foreground mb-6 max-w-md">
                Please select or create a workspace from the navigation bar to start managing your workflows.
              </p>
              <Button onClick={() => window.location.reload()}>
                Refresh Page
              </Button>
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-background p-6">
      <div className="max-w-7xl mx-auto">
        {/* Header with Search and Create */}
        <div className="mb-8 flex justify-between items-center gap-4">
          <div className="relative max-w-sm flex-1">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-workflow-text-muted" />
            <Input
              placeholder="Search workflows..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="pl-9"
            />
          </div>

          {currentWorkspaceId && (
            <Button
              className="gap-2"
              onClick={() => setShowCreateDialog(true)}
            >
              <Plus className="w-4" />
              Create
            </Button>
          )}

          <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Create New Workflow</DialogTitle>
                <DialogDescription>
                  Create a new workflow in your current workspace
                </DialogDescription>
              </DialogHeader>
              <div className="space-y-4 py-4">
                <div>
                  <label className="block text-sm font-mono text-workflow-text-muted mb-1">Name</label>
                  <Input
                    value={newTemplate.name}
                    onChange={(e) => setNewTemplate({ ...newTemplate, name: e.target.value })}
                    placeholder="Workflow name"
                    className="bg-workflow-node-input border-workflow-border"
                  />
                </div>
                <div>
                  <label className="block text-sm font-mono text-workflow-text-muted mb-1">Description</label>
                  <textarea
                    value={newTemplate.description}
                    onChange={(e) => setNewTemplate({ ...newTemplate, description: e.target.value })}
                    placeholder="Workflow description"
                    className="w-full min-h-[80px] px-3 py-2 text-sm bg-workflow-node-input border border-workflow-border rounded-md resize-none focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
                    rows={3}
                  />
                </div>
              </div>
              <DialogFooter>
                <Button variant="outline" onClick={() => setShowCreateDialog(false)}>
                  Cancel
                </Button>
                <Button onClick={handleCreateTemplate} disabled={!newTemplate.name.trim()}>
                  Create Workflow
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>

          <Dialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>确认删除</DialogTitle>
                <DialogDescription asChild>
                  <>
                    您确定要删除工作流{" "}
                    <span className="font-semibold text-workflow-text">
                      {templates.find((t) => t.id === deletingId)?.name}
                    </span>{" "}
                    吗？此操作无法撤销。
                  </>
                </DialogDescription>
              </DialogHeader>
              <DialogFooter>
                <Button variant="outline" onClick={handleCancelDelete}>
                  取消
                </Button>
                <Button variant="destructive" onClick={handleConfirmDelete}>
                  删除
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>

          <Dialog open={showEditDialog} onOpenChange={setShowEditDialog}>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Edit Workflow</DialogTitle>
                <DialogDescription>
                  Update the name and description of your workflow
                </DialogDescription>
              </DialogHeader>
              <div className="space-y-4 py-4">
                <div>
                  <label className="block text-sm font-mono text-workflow-text-muted mb-1">Name</label>
                  <Input
                    value={editingTemplate.name}
                    onChange={(e) => setEditingTemplate({ ...editingTemplate, name: e.target.value })}
                    placeholder="Workflow name"
                    className="bg-workflow-node-input border-workflow-border"
                  />
                </div>
                <div>
                  <label className="block text-sm font-mono text-workflow-text-muted mb-1">Description</label>
                  <textarea
                    value={editingTemplate.description}
                    onChange={(e) => setEditingTemplate({ ...editingTemplate, description: e.target.value })}
                    placeholder="Workflow description"
                    className="w-full min-h-[80px] px-3 py-2 text-sm bg-workflow-node-input border border-workflow-border rounded-md resize-none focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
                    rows={3}
                  />
                </div>
              </div>
              <DialogFooter>
                <Button variant="outline" onClick={handleCancelEdit}>
                  Cancel
                </Button>
                <Button onClick={handleUpdateWorkflow} disabled={!editingTemplate.name.trim()}>
                  Update Workflow
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>

        {/* Workflows Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {filteredTemplates.map((template) => (
            <Card key={template.id} className="group hover:shadow-lg transition-all duration-200">
              <CardHeader className="pb-3">
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-3">
                    <div className="p-2 rounded-lg bg-workflow-surface border border-workflow-border">
                      {template.icon}
                    </div>
                    <div className="flex-1">
                      <CardTitle className="text-lg font-semibold text-workflow-text group-hover:text-primary transition-colors">
                        {template.name}
                      </CardTitle>
                    </div>
                  </div>
                  <div className="flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleEditClick(template)}
                      className="h-8 w-8 p-0"
                    >
                      <Edit className="w-4 h-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleDeleteClick(template.id)}
                      className="h-8 w-8 p-0"
                    >
                      <Trash2 className="w-4 h-4" />
                    </Button>
                  </div>
                </div>
              </CardHeader>
              <CardContent className="pt-0">
                <CardDescription className="text-workflow-text-muted mb-4 line-clamp-2">
                  {template.description || '-'}
                </CardDescription>
                <div className="flex justify-end mb-4">
                  <div className="text-xs text-workflow-text-subtle">
                    Updated: {new Date(template.updatedAt).toLocaleString()}
                  </div>
                </div>
                <Link
                  href={`/workflows/${template.id}`}
                  className="flex-1"
                  onClick={() => handleUseTemplate(template)}
                >
                  <Button className="w-full gap-2 group-hover:bg-primary group-hover:text-primary-foreground transition-colors">
                    Enter Workflow
                    <ArrowRight className="w-4 h-4" />
                  </Button>
                </Link>
              </CardContent>
            </Card>
          ))}
        </div>

        {filteredTemplates.length === 0 && (
          <div className="text-center py-12">
            <h3 className="text-lg font-medium text-workflow-text">
              <WorkflowIcon className="h-12 w-12 mx-auto text-workflow-text-muted mb-4" />
              {searchTerm
                ? `No workflows found matching "${searchTerm}"`
                : 'No workflows found'
              }
            </h3>
          </div>
        )}
      </div>
    </div>
  );
}
