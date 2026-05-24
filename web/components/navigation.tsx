"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import Image from "next/image";
import { Workflow, Zap, Sun, Moon, Monitor, Plus, LayoutGrid, Trash2, Edit2, X, Database, ArrowLeft, Puzzle, HelpCircle } from "lucide-react";
import { useTheme } from "next-themes";
import { useState, useEffect } from "react";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { workspaceApi, Workspace } from "@/lib/api/workspace";

const navigation = [
  { name: "Workflows", href: "/workflows", icon: Workflow },
  { name: "Integrations", href: "/integrations", icon: Database },
  { name: "MCP Tools", href: "/mcp-tools", icon: Puzzle },
  { name: "Help", href: "/help", icon: HelpCircle },
];

const WORKSPACE_STORAGE_KEY = "flowforge-current-workspace-id";

export function Navigation() {
  const pathname = usePathname();
  const router = useRouter();
  const { resolvedTheme, setTheme } = useTheme();
  const [mounted, setMounted] = useState(false);
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [currentWorkspaceId, setCurrentWorkspaceId] = useState<string>("");
  const [showCreateWorkspaceDialog, setShowCreateWorkspaceDialog] = useState(false);
  const [showManageWorkspacesDialog, setShowManageWorkspacesDialog] = useState(false);
  const [showDeleteWorkspaceDialog, setShowDeleteWorkspaceDialog] = useState(false);
  const [deletingWorkspace, setDeletingWorkspace] = useState<Workspace | null>(null);
  const [editingWorkspace, setEditingWorkspace] = useState<Workspace | null>(null);
  const [editWorkspaceName, setEditWorkspaceName] = useState("");
  const [newWorkspaceName, setNewWorkspaceName] = useState("");
  const [loading, setLoading] = useState(true);

  // Prevent hydration mismatch
  useEffect(() => {
    setMounted(true);
  }, []);

  // Load workspaces and restore selected workspace from localStorage
  useEffect(() => {
    loadWorkspaces();
  }, []);

  // When workspaces load, restore selected workspace from localStorage or default to first
  useEffect(() => {
    if (workspaces.length > 0 && !currentWorkspaceId) {
      const savedWorkspaceId = localStorage.getItem(WORKSPACE_STORAGE_KEY);
      if (savedWorkspaceId && workspaces.some(w => w.id === savedWorkspaceId)) {
        setCurrentWorkspaceId(savedWorkspaceId);
      } else {
        setCurrentWorkspaceId(workspaces[0].id);
        localStorage.setItem(WORKSPACE_STORAGE_KEY, workspaces[0].id);
      }
      setLoading(false);
    } else if (workspaces.length === 0) {
      setLoading(false);
    }
  }, [workspaces, currentWorkspaceId]);

  const loadWorkspaces = async () => {
    try {
      const data = await workspaceApi.list();
      setWorkspaces(data.workspaces || []);
    } catch (error) {
      console.error("Failed to load workspaces:", error);
      setWorkspaces([]);
    }
  };

  const handleWorkspaceChange = (workspaceId: string) => {
    localStorage.setItem(WORKSPACE_STORAGE_KEY, workspaceId);
    window.location.reload();
  };

  const handleCreateWorkspace = async () => {
    if (!newWorkspaceName.trim()) return;

    try {
      const newWorkspace = await workspaceApi.create({
        name: newWorkspaceName,
        ownerId: "default-user",
      });

      setShowCreateWorkspaceDialog(false);
      setNewWorkspaceName("");
      localStorage.setItem(WORKSPACE_STORAGE_KEY, newWorkspace.id);
      window.location.reload();
    } catch (error) {
      console.error("Failed to create workspace:", error);
    }
  };

  const handleEditWorkspaceClick = (workspace: Workspace) => {
    setEditingWorkspace(workspace);
    setEditWorkspaceName(workspace.name);
  };

  const handleEditWorkspace = async () => {
    if (!editingWorkspace || !editWorkspaceName.trim()) {
      return;
    }

    try {
      await workspaceApi.update(editingWorkspace.id, { name: editWorkspaceName.trim() });
      await loadWorkspaces();
      setEditingWorkspace(null);
      setEditWorkspaceName("");
    } catch (error) {
      console.error("Failed to edit workspace:", error);
    }
  };

  const handleDeleteWorkspaceClick = (workspace: Workspace) => {
    setDeletingWorkspace(workspace);
    setShowDeleteWorkspaceDialog(true);
  };

  const handleConfirmDeleteWorkspace = async () => {
    if (!deletingWorkspace) return;

    try {
      await workspaceApi.delete(deletingWorkspace.id);
      setShowDeleteWorkspaceDialog(false);
      setDeletingWorkspace(null);
      await loadWorkspaces();
      if (deletingWorkspace.id === currentWorkspaceId) {
        const remaining = workspaces.filter(w => w.id !== deletingWorkspace.id);
        if (remaining.length > 0) {
          setCurrentWorkspaceId(remaining[0].id);
          localStorage.setItem(WORKSPACE_STORAGE_KEY, remaining[0].id);
        } else {
          setCurrentWorkspaceId("");
          localStorage.removeItem(WORKSPACE_STORAGE_KEY);
        }
      }
    } catch (error) {
      console.error("Failed to delete workspace:", error);
    }
  };

  const ThemeToggle = () => (
    <Button
      variant="ghost"
      size="sm"
      onClick={() => setTheme(resolvedTheme === "dark" ? "light" : "dark")}
      className="h-8 w-8 p-0"
      title="Toggle theme"
    >
      {!mounted ? (
        <Monitor className="h-4 w-4" />
      ) : resolvedTheme === "dark" ? (
        <Sun className="h-4 w-4" />
      ) : (
        <Moon className="h-4 w-4" />
      )}
    </Button>
  );

  const currentWorkspace = workspaces.find(w => w.id === currentWorkspaceId);
  const isWorkflowEditor = pathname.startsWith("/workflows/") && pathname !== "/workflows";

  return (
    <nav className="border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="container mx-auto px-4">
        <div className="flex h-16 items-center justify-between">
          <div className="flex items-center space-x-8">
            {isWorkflowEditor ? (
              // Back button for workflow editor
              <Button
                variant="ghost"
                asChild
                className="flex items-center space-x-2"
              >
                <Link href="/workflows" className="flex items-center space-x-2">
                  <ArrowLeft className="h-4 w-4" />
                  <span>Back to Workflows</span>
                </Link>
              </Button>
            ) : (
              // Normal navigation
              <>
                <Link href="/" className="flex items-center space-x-2">
                  <Image
                    src="/logo.png"
                    alt="SREFlow AI"
                    width={28}
                    height={28}
                    className="rounded-md"
                  />
                  <span className="font-mono font-semibold text-workflow-text tracking-tight text-xl font-bold">
                    SREFlow AI
                  </span>
                </Link>

                <div className="h-5 w-px bg-workflow-border" />

                <div className="flex space-x-1">
                  {navigation.map((item) => {
                    const isActive = pathname === item.href;
                    return (
                      <Button
                        key={item.name}
                        variant={isActive ? "default" : "ghost"}
                        asChild
                        className="flex items-center space-x-2"
                      >
                        <Link href={item.href}>
                          <item.icon className="h-4 w-4" />
                          <span>{item.name}</span>
                        </Link>
                      </Button>
                    );
                  })}
                </div>
              </>
            )}
          </div>

          {/* Right side - Workspace Selector and Theme Toggle */}
          <div className="flex items-center gap-3">
            {/* Workspace Selector - Only show on workflows/integrations list pages, not in workflow editor */}
            {!loading && !isWorkflowEditor && (pathname === "/workflows" || pathname === "/integrations" || pathname === "/mcp-tools" || pathname === "/help") && (
              <div className="flex items-center">
                <LayoutGrid className="h-4 w-4 mr-2 text-muted-foreground" />
                {workspaces.length === 0 ? (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setShowCreateWorkspaceDialog(true)}
                    className="gap-2"
                  >
                    <Plus className="h-4 w-4" />
                    Create First Workspace
                  </Button>
                ) : (
                  <Select value={currentWorkspaceId} onValueChange={handleWorkspaceChange}>
                    <SelectTrigger className="w-[180px]">
                      <SelectValue placeholder="Select workspace" />
                    </SelectTrigger>
                    <SelectContent>
                      {workspaces.map((workspace) => (
                        <SelectItem key={workspace.id} value={workspace.id}>
                          {workspace.name}
                        </SelectItem>
                      ))}
                      <div className="border-t my-1" />
                      <Button
                        variant="ghost"
                        className="w-full justify-start text-muted-foreground gap-2"
                        onClick={() => setShowManageWorkspacesDialog(true)}
                      >
                        <LayoutGrid className="h-4 w-4" />
                        Manage Workspaces
                      </Button>
                    </SelectContent>
                  </Select>
                )}
              </div>
            )}

            <ThemeToggle />
          </div>
        </div>
      </div>

      {/* Create Workspace Dialog */}
      <Dialog open={showCreateWorkspaceDialog} onOpenChange={setShowCreateWorkspaceDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Create New Workspace</DialogTitle>
            <DialogDescription>
              Create a workspace to organize your workflows and data sources.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div>
              <label className="block text-sm font-medium mb-1">Workspace Name</label>
              <Input
                value={newWorkspaceName}
                onChange={(e) => setNewWorkspaceName(e.target.value)}
                placeholder="My Workspace"
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowCreateWorkspaceDialog(false)}>
              Cancel
            </Button>
            <Button onClick={handleCreateWorkspace} disabled={!newWorkspaceName.trim()}>
              Create Workspace
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Manage Workspaces Dialog */}
      <Dialog open={showManageWorkspacesDialog} onOpenChange={setShowManageWorkspacesDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Manage Workspaces</DialogTitle>
            <DialogDescription>
              View and delete workspaces. Deleting a workspace will also delete all associated workflows and integrations.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-3 py-4 max-h-[300px] overflow-y-auto">
            {workspaces.length === 0 ? (
              <p className="text-sm text-muted-foreground text-center py-4">No workspaces found</p>
            ) : (
              workspaces.map((workspace) => (
                <div
                  key={workspace.id}
                  className="flex items-center justify-between p-3 rounded-lg border border-workflow-border bg-workflow-surface"
                >
                  {editingWorkspace?.id === workspace.id ? (
                    <div className="flex items-center gap-2 flex-1">
                      <Input
                        value={editWorkspaceName}
                        onChange={(e) => setEditWorkspaceName(e.target.value)}
                        placeholder="Workspace name"
                        className="flex-1"
                        autoFocus
                      />
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={handleEditWorkspace}
                        className="text-green-600 hover:text-green-700"
                      >
                        ✓
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => {
                          setEditingWorkspace(null);
                          setEditWorkspaceName("");
                        }}
                        className="text-muted-foreground hover:text-foreground"
                      >
                        <X className="h-4 w-4" />
                      </Button>
                    </div>
                  ) : (
                    <>
                      <div>
                        <p className="font-medium text-workflow-text">
                          {workspace.name}
                          {workspace.id === currentWorkspaceId && (
                            <span className="ml-2 text-xs text-primary">(current)</span>
                          )}
                        </p>
                        <p className="text-xs text-workflow-text-muted">ID: {workspace.id}</p>
                      </div>
                      <div className="flex gap-1">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleEditWorkspaceClick(workspace)}
                          className="text-muted-foreground hover:text-foreground"
                        >
                          <Edit2 className="h-4 w-4" />
                        </Button>
                        {workspaces.length > 1 && (
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleDeleteWorkspaceClick(workspace)}
                            className="text-destructive hover:text-destructive hover:bg-destructive/10"
                          >
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        )}
                      </div>
                    </>
                  )}
                </div>
              ))
            )}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowManageWorkspacesDialog(false)}>
              Close
            </Button>
            <Button onClick={() => { setShowManageWorkspacesDialog(false); setShowCreateWorkspaceDialog(true); }}>
              <Plus className="h-4 w-4 mr-1" />
              New Workspace
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Workspace Confirmation Dialog */}
      <Dialog open={showDeleteWorkspaceDialog} onOpenChange={setShowDeleteWorkspaceDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Confirm Delete Workspace</DialogTitle>
            <DialogDescription asChild>
              <>
                Are you sure you want to delete workspace{" "}
                <span className="font-semibold text-workflow-text">{deletingWorkspace?.name}</span>?
                All workflows and integrations in this workspace will be deleted. This action cannot be undone.
              </>
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowDeleteWorkspaceDialog(false)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleConfirmDeleteWorkspace}
            >
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </nav>
  );
}
