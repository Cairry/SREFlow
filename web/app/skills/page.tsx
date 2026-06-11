"use client";

import { useEffect, useState } from "react";
import {
  BookOpen,
  Plus,
  Trash2,
  Search,
  Loader2,
  Edit,
  LayoutGrid,
  Power,
  PowerOff,
} from "lucide-react";
import { toast } from "@/hooks/use-toast";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import { skillApi, Skill } from "@/lib/api/skill";

export default function SkillsPage() {
  const [skills, setSkills] = useState<Skill[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState("");
  const [createOpen, setCreateOpen] = useState(false);
  const [deleteId, setDeleteId] = useState<string | null>(null);
  const [editingSkill, setEditingSkill] = useState<Skill | null>(null);
  const [editOpen, setEditOpen] = useState(false);
  const [workspaceId, setWorkspaceId] = useState<string>("");
  const [noWorkspace, setNoWorkspace] = useState(false);

  // Form state
  const [newName, setNewName] = useState("");
  const [newDescription, setNewDescription] = useState("");
  const [newContent, setNewContent] = useState("");
  const [newEnabled, setNewEnabled] = useState(true);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    const stored = localStorage.getItem("flowforge-current-workspace-id");
    if (stored) {
      setWorkspaceId(stored);
    } else {
      setNoWorkspace(true);
    }
  }, []);

  const loadSkills = async () => {
    if (!workspaceId) {
      setNoWorkspace(true);
      return;
    }
    setNoWorkspace(false);
    setLoading(true);
    try {
      const data = await skillApi.list();
      setSkills(data.skills || []);
    } catch (error) {
      console.error("Failed to load skills:", error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadSkills();
  }, [workspaceId]);

  const filteredSkills = skills.filter(
    (sk) =>
      sk.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      sk.description?.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const resetForm = () => {
    setNewName("");
    setNewDescription("");
    setNewContent("");
    setNewEnabled(true);
  };

  const handleCreate = async () => {
    if (!newName || !newContent) return;
    setSubmitting(true);
    try {
      await skillApi.create({
        name: newName,
        description: newDescription || undefined,
        content: newContent,
        enabled: newEnabled,
      });
      resetForm();
      setCreateOpen(false);
      loadSkills();
    } catch (error) {
      toast({
        title: "Failed to create skill",
        description: error instanceof Error ? error.message : "Unknown error",
        variant: "destructive",
      });
    } finally {
      setSubmitting(false);
    }
  };

  const handleUpdate = async () => {
    if (!editingSkill || !newName) return;
    setSubmitting(true);
    try {
      await skillApi.update(editingSkill.id, {
        name: newName,
        description: newDescription || undefined,
        content: newContent,
        enabled: newEnabled,
      });
      resetForm();
      setEditOpen(false);
      setEditingSkill(null);
      loadSkills();
    } catch (error) {
      toast({
        title: "Failed to update skill",
        description: error instanceof Error ? error.message : "Unknown error",
        variant: "destructive",
      });
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteId) return;
    try {
      await skillApi.delete(deleteId);
      setDeleteId(null);
      loadSkills();
    } catch (error) {
      toast({
        title: "Failed to delete skill",
        description: error instanceof Error ? error.message : "Unknown error",
        variant: "destructive",
      });
    }
  };

  const handleToggleEnabled = async (skill: Skill) => {
    try {
      await skillApi.update(skill.id, { enabled: !skill.enabled });
      loadSkills();
    } catch (error) {
      toast({
        title: "Failed to update skill",
        description: error instanceof Error ? error.message : "Unknown error",
        variant: "destructive",
      });
    }
  };

  const startEdit = (skill: Skill) => {
    setEditingSkill(skill);
    setNewName(skill.name);
    setNewDescription(skill.description || "");
    setNewContent(skill.content || "");
    setNewEnabled(skill.enabled);
    setEditOpen(true);
  };

  if (noWorkspace) {
    return (
      <div className="min-h-screen bg-workflow-bg p-6">
        <div className="max-w-7xl mx-auto">
          <div className="flex items-center justify-center min-h-[60vh]">
            <div className="text-center">
              <div className="flex justify-center mb-4">
                <div className="p-4 rounded-full bg-muted">
                  <LayoutGrid className="w-12 h-12 text-muted-foreground" />
                </div>
              </div>
              <h2 className="text-2xl font-semibold mb-2 text-workflow-text">
                No Workspace Selected
              </h2>
              <p className="text-workflow-text-muted mb-6 max-w-md">
                Please select or create a workspace from the navigation bar to
                manage skills.
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

  const formContent = (
    <div className="space-y-4 py-4 max-h-[60vh] overflow-y-auto">
      <div className="space-y-2">
        <Label className="text-workflow-text">Name *</Label>
        <Input
          value={newName}
          onChange={(e) => setNewName(e.target.value)}
          placeholder="e.g. Kubernetes Troubleshooting"
          className="bg-workflow-bg border-workflow-border text-workflow-text"
        />
      </div>
      <div className="space-y-2">
        <Label className="text-workflow-text">Description *</Label>
        <Input
          value={newDescription}
          onChange={(e) => setNewDescription(e.target.value)}
          placeholder="Brief description of this skill..."
          className="bg-workflow-bg border-workflow-border text-workflow-text"
        />
      </div>
      <div className="space-y-2">
        <Label className="text-workflow-text">
          Content *
        </Label>
        <Textarea
          value={newContent}
          onChange={(e) => setNewContent(e.target.value)}
          placeholder="Enter the skill's system prompt content. This will be injected into the Agent's context as domain knowledge..."
          className="bg-workflow-bg border-workflow-border text-workflow-text resize-none font-mono"
          rows={12}
        />
        <p className="text-[11px] text-workflow-text-muted font-mono">
          Tool hints: use <code className="text-primary bg-primary/10 px-1 rounded">@tool:server.name</code> to specify a tool, <code className="text-primary bg-primary/10 px-1 rounded">@server:name</code> to include all tools from a server. Example: <code className="text-primary bg-primary/10 px-1 rounded">@tool:prometheus-server.prometheus_query @server:kubernetes</code>
        </p>
      </div>
      <div className="flex items-center gap-2">
        <Switch checked={newEnabled} onCheckedChange={setNewEnabled} />
        <Label className="text-workflow-text">Enabled</Label>
      </div>
    </div>
  );

  return (
    <div className="min-h-screen bg-workflow-bg p-6">
      <div className="max-w-7xl mx-auto">
        {/* Create Dialog */}
        <Dialog open={createOpen} onOpenChange={setCreateOpen}>
          <DialogContent className="sm:max-w-[700px] bg-workflow-surface border-workflow-border">
            <DialogHeader>
              <DialogTitle className="text-workflow-text">
                Create Skill
              </DialogTitle>
              <DialogDescription className="text-workflow-text-muted">
                Create a reusable skill that can be assigned to Agent nodes
              </DialogDescription>
            </DialogHeader>
            {formContent}
            <DialogFooter>
              <Button
                variant="outline"
                onClick={() => {
                  resetForm();
                  setCreateOpen(false);
                }}
                className="border-workflow-border text-workflow-text"
              >
                Cancel
              </Button>
              <Button
                variant="outline"
                onClick={handleCreate}
                disabled={submitting || !newName || !newContent}
                className="border-workflow-border text-workflow-text"
              >
                {submitting && (
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                )}
                Create
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        {/* Edit Dialog */}
        <Dialog open={editOpen} onOpenChange={setEditOpen}>
          <DialogContent className="sm:max-w-[700px] bg-workflow-surface border-workflow-border">
            <DialogHeader>
              <DialogTitle className="text-workflow-text">
                Edit Skill
              </DialogTitle>
              <DialogDescription className="text-workflow-text-muted">
                Update skill configuration
              </DialogDescription>
            </DialogHeader>
            {formContent}
            <DialogFooter>
              <Button
                variant="outline"
                onClick={() => {
                  resetForm();
                  setEditOpen(false);
                  setEditingSkill(null);
                }}
                className="border-workflow-border text-workflow-text"
              >
                Cancel
              </Button>
              <Button
                variant="outline"
                onClick={handleUpdate}
                disabled={submitting || !newName}
                className="border-workflow-border text-workflow-text"
              >
                {submitting && (
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                )}
                Save Changes
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        {/* Delete Dialog */}
        <Dialog open={!!deleteId} onOpenChange={() => setDeleteId(null)}>
          <DialogContent className="sm:max-w-[400px] bg-workflow-surface border-workflow-border">
            <DialogHeader>
              <DialogTitle className="text-workflow-text">
                Delete Skill
              </DialogTitle>
              <DialogDescription className="text-workflow-text-muted">
                Are you sure you want to delete this skill? This action cannot
                be undone.
              </DialogDescription>
            </DialogHeader>
            <DialogFooter>
              <Button
                variant="outline"
                onClick={() => setDeleteId(null)}
                className="border-workflow-border text-workflow-text"
              >
                Cancel
              </Button>
              <Button
                onClick={handleDelete}
                variant="destructive"
                className="bg-red-600 hover:bg-red-700"
              >
                Delete
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        {/* Header */}
        <div className="mb-8 flex justify-between items-center gap-4">
          <div className="flex items-center gap-4 flex-1">
            <div className="relative max-w-sm flex-1">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-workflow-text-muted" />
              <Input
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                placeholder="Search skills..."
                className="pl-9"
              />
            </div>
          </div>
          {workspaceId && (
            <Button className="gap-2" onClick={() => setCreateOpen(true)}>
              <Plus className="w-4" />
              Create Skill
            </Button>
          )}
        </div>

        {/* Content */}
        {loading ? (
          <div className="flex items-center justify-center py-12">
            <Loader2 className="h-8 w-8 animate-spin text-workflow-primary" />
          </div>
        ) : filteredSkills.length === 0 ? (
          <div className="text-center py-12">
            <BookOpen className="h-12 w-12 mx-auto text-workflow-text-muted mb-4" />
            <h3 className="text-lg font-medium text-workflow-text">
              No skills found
            </h3>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {filteredSkills.map((sk) => (
              <Card
                key={sk.id}
                className="group hover:shadow-lg transition-all duration-200"
              >
                <CardContent className="pt-4">
                  <div className="flex items-start justify-between">
                    <div className="flex items-center gap-2 min-w-0 flex-1">
                      <div className="p-2 rounded-lg bg-blue-500/10 border border-blue-500/20 text-blue-400">
                        <BookOpen className="h-4 w-4" />
                      </div>
                      <div className="min-w-0 flex-1">
                        <h3 className="text-sm font-semibold text-workflow-text truncate group-hover:text-primary transition-colors">
                          {sk.name}
                        </h3>
                        <span
                          className={`inline-flex items-center gap-1 px-1.5 py-0.5 rounded-full text-[10px] font-mono ${
                            sk.enabled
                              ? "bg-green-500/10 text-green-400 border border-green-500/20"
                              : "bg-stone-500/10 text-stone-400 border border-stone-500/20"
                          }`}
                        >
                          {sk.enabled ? "Enabled" : "Disabled"}
                        </span>
                      </div>
                    </div>
                    <div className="flex gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleToggleEnabled(sk)}
                        className="h-7 w-7 p-0"
                        title={sk.enabled ? "Disable" : "Enable"}
                      >
                        {sk.enabled ? (
                          <PowerOff className="h-3 w-3" />
                        ) : (
                          <Power className="h-3 w-3" />
                        )}
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => startEdit(sk)}
                        className="h-7 w-7 p-0"
                      >
                        <Edit className="h-3 w-3" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setDeleteId(sk.id)}
                        className="h-7 w-7 p-0 text-workflow-text-muted hover:text-red-400"
                      >
                        <Trash2 className="h-3 w-3" />
                      </Button>
                    </div>
                  </div>

                  {sk.description && (
                    <CardDescription className="text-workflow-text-muted mt-2 line-clamp-2">
                      {sk.description}
                    </CardDescription>
                  )}

                  <div className="mt-3 p-2 rounded bg-workflow-bg/50 border border-workflow-border/50">
                    <p className="text-xs text-workflow-text-muted font-mono line-clamp-3">
                      {sk.content}
                    </p>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
