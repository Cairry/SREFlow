"use client";

import { useEffect, useState, useCallback } from "react";
import {
  BookOpen,
  Plus,
  Trash2,
  Search,
  Loader2,
  ArrowLeft,
  Save,
  Power,
  PowerOff,
  PanelTop,
  PanelTopClose,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { toast } from "@/hooks/use-toast";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { skillApi, Skill } from "@/lib/api/skill";
import { isNavHidden, toggleNav, setDetailMode } from "@/lib/nav-toggle";

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

export default function SkillsPage() {
  const [skills, setSkills] = useState<Skill[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState("");
  const [workspaceId, setWorkspaceId] = useState<string>("");
  const [noWorkspace, setNoWorkspace] = useState(false);

  // Editor state: null = list view, Skill = editing, "new" = creating
  const [editorMode, setEditorMode] = useState<"list" | "create" | "edit">("list");
  const [editingSkill, setEditingSkill] = useState<Skill | null>(null);

  // Form fields
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [content, setContent] = useState("");
  const [enabled, setEnabled] = useState(true);
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

  const openCreate = () => {
    setName("");
    setDescription("");
    setContent("");
    setEnabled(true);
    setEditingSkill(null);
    setEditorMode("create");
    setDetailMode(true);
  };

  const openEdit = (skill: Skill) => {
    setName(skill.name);
    setDescription(skill.description || "");
    setContent(skill.content || "");
    setEnabled(skill.enabled);
    setEditingSkill(skill);
    setEditorMode("edit");
    setDetailMode(true);
  };

  const backToList = () => {
    setEditorMode("list");
    setEditingSkill(null);
    setDetailMode(false);
  };

  const handleSave = async () => {
    if (!name.trim() || !description.trim() || !content.trim()) return;
    setSubmitting(true);
    try {
      if (editorMode === "create") {
        await skillApi.create({
          name: name.trim(),
          description: description.trim(),
          content,
          enabled,
        });
        toast({ title: "Skill created" });
      } else if (editingSkill) {
        await skillApi.update(editingSkill.id, {
          name: name.trim(),
          description: description.trim(),
          content,
          enabled,
        });
        toast({ title: "Skill updated" });
      }
      backToList();
      loadSkills();
    } catch (error) {
      toast({
        title: editorMode === "create" ? "Failed to create skill" : "Failed to update skill",
        description: error instanceof Error ? error.message : "Unknown error",
        variant: "destructive",
      });
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = useCallback(async () => {
    const target = editingSkill;
    if (!target) return;
    if (!confirm(`Delete skill "${target.name}"? This cannot be undone.`)) return;
    try {
      await skillApi.delete(target.id);
      toast({ title: "Skill deleted" });
      backToList();
      loadSkills();
    } catch (error) {
      toast({
        title: "Failed to delete skill",
        description: error instanceof Error ? error.message : "Unknown error",
        variant: "destructive",
      });
    }
  }, [editingSkill]);

  const handleToggleEnabled = useCallback(async (skill: Skill) => {
    try {
      await skillApi.update(skill.id, { enabled: !skill.enabled });
      // If editing this skill, update local state too
      if (editingSkill?.id === skill.id) {
        setEnabled(!skill.enabled);
        setEditingSkill({ ...skill, enabled: !skill.enabled });
      }
      loadSkills();
    } catch (error) {
      toast({
        title: "Failed to update skill",
        description: error instanceof Error ? error.message : "Unknown error",
        variant: "destructive",
      });
    }
  }, [editingSkill]);

  // Keyboard shortcut: Ctrl/Cmd+S to save
  useEffect(() => {
    if (editorMode === "list") return;
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "s") {
        e.preventDefault();
        handleSave();
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [editorMode, name, description, content, enabled, editingSkill]);

  // --- No Workspace ---
  if (noWorkspace) {
    return (
      <div className="min-h-screen bg-workflow-bg p-6">
        <div className="max-w-7xl mx-auto">
          <div className="flex items-center justify-center min-h-[60vh]">
            <div className="text-center">
              <div className="flex justify-center mb-4">
                <div className="p-4 rounded-full bg-muted">
                  <BookOpen className="w-12 h-12 text-muted-foreground" />
                </div>
              </div>
              <h2 className="text-2xl font-semibold mb-2 text-workflow-text">
                No Workspace Selected
              </h2>
              <p className="text-workflow-text-muted mb-6 max-w-md">
                Please select or create a workspace from the navigation bar to manage skills.
              </p>
              <Button onClick={() => window.location.reload()}>Refresh Page</Button>
            </div>
          </div>
        </div>
      </div>
    );
  }

  // --- Editor View ---
  if (editorMode === "create" || editorMode === "edit") {
    return (
      <div className="min-h-screen bg-workflow-bg flex flex-col">
        {/* Toolbar - matching workflow toolbar style */}
        <div className="px-4 pt-3 pb-1 flex justify-center">
          <div className="bg-workflow-surface border border-workflow-border rounded-xl px-4 h-14 flex items-center shadow-sm w-fit">
            <div className="flex items-center gap-6">
              {/* Back button */}
              <button
                type="button"
                onClick={backToList}
                className="flex items-center gap-1.5 text-workflow-text-muted hover:text-workflow-text transition-colors duration-200 shrink-0"
                title="Back to Skills"
              >
                <ArrowLeft className="w-4 h-4" />
                <span className="font-mono text-xs">Back</span>
              </button>

              <div className="h-5 w-px bg-workflow-border shrink-0" />

              {/* Skill name display */}
              <div className="flex items-center gap-2 shrink-0">
                <BookOpen className="w-4 h-4 text-workflow-text-muted" />
                <span className="text-workflow-text font-mono text-sm font-medium truncate max-w-48">
                  {name || (editorMode === "create" ? "New Skill" : editingSkill?.name)}
                </span>
              </div>

              {/* Action button groups */}
              <div className="flex items-center gap-2">
                {/* Enable/Disable toggle */}
                <ButtonGroup>
                  <IconButton
                    icon={enabled ? <PowerOff className="w-4 h-4" /> : <Power className="w-4 h-4" />}
                    onClick={() => {
                      if (editorMode === "edit" && editingSkill) {
                        handleToggleEnabled(editingSkill);
                      } else {
                        setEnabled(!enabled);
                      }
                    }}
                    title={enabled ? "Disable skill" : "Enable skill"}
                    active={enabled}
                  />
                </ButtonGroup>

                {/* Delete */}
                {editorMode === "edit" && (
                  <ButtonGroup>
                    <IconButton
                      icon={<Trash2 className="w-4 h-4" />}
                      onClick={handleDelete}
                      title="Delete skill"
                      variant="danger"
                    />
                  </ButtonGroup>
                )}

                {/* Save */}
                <ButtonGroup>
                  <IconButton
                    icon={submitting ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
                    onClick={handleSave}
                    title="Save (Ctrl+S)"
                    disabled={submitting || !name.trim() || !description.trim() || !content.trim()}
                  />
                </ButtonGroup>

                {/* Nav Toggle */}
                <ButtonGroup>
                  <NavToggleButton />
                </ButtonGroup>
              </div>
            </div>
          </div>
        </div>

        {/* Document body */}
        <div className="flex-1 overflow-y-auto">
          <div className="max-w-4xl mx-auto px-6 py-8">
            {/* Name - Document title */}
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="* Skill name..."
              className="w-full text-3xl font-bold bg-transparent border-none outline-none text-workflow-text placeholder:text-workflow-text-muted/40 mb-3"
              autoFocus
            />

            {/* Description - Subtitle */}
            <input
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="* Brief description..."
              className="w-full text-base bg-transparent border-none outline-none text-workflow-text-muted placeholder:text-workflow-text-muted/40 mb-8 pb-6 border-b border-workflow-border"
            />

            {/* Content - Document body */}
            <textarea
              value={content}
              onChange={(e) => setContent(e.target.value)}
              placeholder="Write your skill content here...&#10;&#10;This content will be loaded by the Agent as domain knowledge when the skill is activated.&#10;&#10;You can include instructions, reference documentation, troubleshooting steps, etc."
              className="w-full flex-1 min-h-[calc(100vh-340px)] bg-transparent border-none outline-none text-workflow-text placeholder:text-workflow-text-muted/40 resize-none font-mono text-sm leading-relaxed"
            />
          </div>
        </div>
      </div>
    );
  }

  // --- List View ---
  return (
    <div className="min-h-screen p-6">
      <div className="max-w-7xl mx-auto">
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
            <Button className="gap-2" onClick={openCreate}>
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
              {searchTerm ? "No skills match your search" : "No skills yet"}
            </h3>
            <p className="text-workflow-text-muted mt-2">
              {searchTerm ? "Try a different search term" : "Click Create Skill to get started"}
            </p>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {filteredSkills.map((sk) => (
              <div
                key={sk.id}
                onClick={() => openEdit(sk)}
                className="group cursor-pointer rounded-xl border border-workflow-border bg-workflow-surface p-4 hover:shadow-lg hover:border-workflow-text-muted/30 transition-all duration-200"
              >
                <div className="flex items-start justify-between mb-2">
                  <div className="flex items-center gap-2 min-w-0 flex-1">
                    <div className="p-2 rounded-lg bg-blue-500/10 border border-blue-500/20 text-blue-400">
                      <BookOpen className="h-4 w-4" />
                    </div>
                    <h3 className="text-sm font-semibold text-workflow-text truncate group-hover:text-primary transition-colors">
                      {sk.name}
                    </h3>
                  </div>
                  <span
                    className={`shrink-0 inline-flex items-center px-1.5 py-0.5 rounded-full text-[10px] font-mono ${
                      sk.enabled
                        ? "bg-green-500/10 text-green-400 border border-green-500/20"
                        : "bg-stone-500/10 text-stone-400 border border-stone-500/20"
                    }`}
                  >
                    {sk.enabled ? "Enabled" : "Disabled"}
                  </span>
                </div>

                {sk.description && (
                  <p className="text-xs text-workflow-text-muted line-clamp-2 mb-2">
                    {sk.description}
                  </p>
                )}

                <div className="p-2 rounded bg-workflow-bg/50 border border-workflow-border/50">
                  <p className="text-[11px] text-workflow-text-muted font-mono line-clamp-3">
                    {sk.content}
                  </p>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
