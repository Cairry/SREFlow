"use client";

import { useEffect, useState } from "react";
import {
  Puzzle,
  Plus,
  Trash2,
  Search,
  Loader2,
  Globe,
  ChevronRight,
  ChevronDown,
  RefreshCw,
  X,
  Edit,
  LayoutGrid,
  Power,
  PowerOff,
  Wrench,
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import { mcpApi, MCPServer, MCPServerTool } from "@/lib/api/mcp";

const SERVER_TYPES = [
  { value: "http", label: "HTTP" },
  { value: "streamableHttp", label: "Streamable HTTP" },
  { value: "sse", label: "SSE" },
];

export default function MCPToolsPage() {
  const [servers, setServers] = useState<MCPServer[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState("");
  const [createOpen, setCreateOpen] = useState(false);
  const [deleteId, setDeleteId] = useState<string | null>(null);
  const [editingServer, setEditingServer] = useState<MCPServer | null>(null);
  const [editOpen, setEditOpen] = useState(false);
  const [workspaceId, setWorkspaceId] = useState<string>("");
  const [noWorkspace, setNoWorkspace] = useState(false);
  const [expandedServer, setExpandedServer] = useState<string | null>(null);
  const [serverTools, setServerTools] = useState<Record<string, MCPServerTool[]>>({});
  const [loadingTools, setLoadingTools] = useState<Record<string, boolean>>({});
  const [serverStatus, setServerStatus] = useState<Record<string, boolean | null>>({});

  // Create/Edit form state
  const [newName, setNewName] = useState("");
  const [newType, setNewType] = useState<string>("http");
  const [newURL, setNewURL] = useState("");
  const [newAPIKey, setNewAPIKey] = useState("");
  const [newHeaders, setNewHeaders] = useState("");
  const [newTimeout, setNewTimeout] = useState(30);
  const [newEnabled, setNewEnabled] = useState(true);
  const [newDescription, setNewDescription] = useState("");
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    const stored = localStorage.getItem("flowforge-current-workspace-id");
    if (stored) {
      setWorkspaceId(stored);
    } else {
      setNoWorkspace(true);
    }
  }, []);

  const loadServers = async () => {
    if (!workspaceId) {
      setNoWorkspace(true);
      return;
    }
    setNoWorkspace(false);
    setLoading(true);
    try {
      const data = await mcpApi.list();
      setServers(data.servers || []);
    } catch (error) {
      console.error("Failed to load MCP servers:", error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadServers();
  }, [workspaceId]);

  const filteredServers = servers.filter((srv) =>
    srv.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    srv.description?.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const resetForm = () => {
    setNewName("");
    setNewType("http");
    setNewURL("");
    setNewAPIKey("");
    setNewHeaders("");
    setNewTimeout(30);
    setNewEnabled(true);
    setNewDescription("");
  };

  const handleCreate = async () => {
    if (!newName || !newURL) return;
    setSubmitting(true);

    const headersObj: Record<string, string> = {};
    if (newHeaders.trim()) {
      newHeaders.split('\n').forEach(line => {
        const colonIndex = line.indexOf(':');
        if (colonIndex > 0) {
          const key = line.substring(0, colonIndex).trim();
          const val = line.substring(colonIndex + 1).trim();
          if (key && val) {
            headersObj[key] = val;
          }
        }
      });
    }

    try {
      await mcpApi.create({
        name: newName,
        type: newType,
        url: newURL,
        headers: Object.keys(headersObj).length > 0 ? headersObj : undefined,
        apiKey: newAPIKey || undefined,
        timeout: newTimeout,
        enabled: newEnabled,
        description: newDescription || undefined,
      });
      resetForm();
      setCreateOpen(false);
      loadServers();
    } catch (error) {
      toast({ title: "Failed to create MCP server", description: error instanceof Error ? error.message : 'Unknown error', variant: "destructive" });
    } finally {
      setSubmitting(false);
    }
  };

  const handleUpdate = async () => {
    if (!editingServer || !newName) return;
    setSubmitting(true);

    const headersObj: Record<string, string> = {};
    if (newHeaders.trim()) {
      newHeaders.split('\n').forEach(line => {
        const colonIndex = line.indexOf(':');
        if (colonIndex > 0) {
          const key = line.substring(0, colonIndex).trim();
          const val = line.substring(colonIndex + 1).trim();
          if (key && val) {
            headersObj[key] = val;
          }
        }
      });
    }

    try {
      await mcpApi.update(editingServer.id, {
        name: newName,
        type: newType,
        url: newURL,
        headers: Object.keys(headersObj).length > 0 ? headersObj : undefined,
        apiKey: newAPIKey || undefined,
        timeout: newTimeout,
        enabled: newEnabled,
        description: newDescription || undefined,
      });
      resetForm();
      setEditOpen(false);
      setEditingServer(null);
      loadServers();
    } catch (error) {
      toast({ title: "Failed to update MCP server", description: error instanceof Error ? error.message : 'Unknown error', variant: "destructive" });
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteId) return;
    try {
      await mcpApi.delete(deleteId);
      setDeleteId(null);
      loadServers();
    } catch (error) {
      toast({ title: "Failed to delete MCP server", description: error instanceof Error ? error.message : 'Unknown error', variant: "destructive" });
    }
  };

  const handleToggleEnabled = async (server: MCPServer) => {
    try {
      await mcpApi.update(server.id, { enabled: !server.enabled });
      loadServers();
    } catch (error) {
      toast({ title: "Failed to update server", description: error instanceof Error ? error.message : 'Unknown error', variant: "destructive" });
    }
  };

  const loadTools = async (serverId: string) => {
    setLoadingTools(prev => ({ ...prev, [serverId]: true }));
    try {
      const data = await mcpApi.listTools(serverId);
      setServerTools(prev => ({ ...prev, [serverId]: data.tools || [] }));
    } catch (error) {
      setServerTools(prev => ({ ...prev, [serverId]: [] }));
      toast({ title: "Failed to load tools", description: error instanceof Error ? error.message : 'Unknown error', variant: "destructive" });
    } finally {
      setLoadingTools(prev => ({ ...prev, [serverId]: false }));
    }
  };

  const checkStatus = async (serverId: string) => {
    try {
      const data = await mcpApi.getStatus(serverId);
      setServerStatus(prev => ({ ...prev, [serverId]: data.connected }));
    } catch {
      setServerStatus(prev => ({ ...prev, [serverId]: false }));
    }
  };

  const toggleExpand = (serverId: string) => {
    if (expandedServer === serverId) {
      setExpandedServer(null);
    } else {
      setExpandedServer(serverId);
      if (!serverTools[serverId]) {
        loadTools(serverId);
      }
    }
  };

  const startEdit = (server: MCPServer) => {
    setEditingServer(server);
    setNewName(server.name);
    setNewType(server.type);
    setNewURL(server.url);
    setNewAPIKey(server.apiKey || "");
    setNewHeaders(server.headers ? Object.entries(server.headers).map(([k, v]) => `${k}: ${v}`).join('\n') : "");
    setNewTimeout(server.timeout || 30);
    setNewEnabled(server.enabled);
    setNewDescription(server.description || "");
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
              <h2 className="text-2xl font-semibold mb-2 text-workflow-text">No Workspace Selected</h2>
              <p className="text-workflow-text-muted mb-6 max-w-md">
                Please select or create a workspace from the navigation bar to manage MCP tools.
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
          placeholder="My MCP Server"
          className="bg-workflow-bg border-workflow-border text-workflow-text"
        />
      </div>
      <div className="space-y-2">
        <Label className="text-workflow-text">Type</Label>
        <Select value={newType} onValueChange={setNewType}>
          <SelectTrigger className="bg-workflow-bg border-workflow-border text-workflow-text">
            <SelectValue />
          </SelectTrigger>
          <SelectContent className="bg-workflow-surface border-workflow-border">
            {SERVER_TYPES.map((t) => (
              <SelectItem key={t.value} value={t.value}>
                {t.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className="space-y-2">
        <Label className="text-workflow-text">URL *</Label>
        <Input
          value={newURL}
          onChange={(e) => setNewURL(e.target.value)}
          placeholder="http://localhost:3001"
          className="bg-workflow-bg border-workflow-border text-workflow-text"
        />
      </div>
      <div className="space-y-2">
        <Label className="text-workflow-text">API Key</Label>
        <Input
          value={newAPIKey}
          onChange={(e) => setNewAPIKey(e.target.value)}
          placeholder="Optional API key"
          type="password"
          className="bg-workflow-bg border-workflow-border text-workflow-text"
        />
      </div>
      <div className="space-y-2">
        <Label className="text-workflow-text">Headers (Key: Value, one per line)</Label>
        <Textarea
          value={newHeaders}
          onChange={(e) => setNewHeaders(e.target.value)}
          placeholder={`Authorization: Bearer token123\nX-Custom-Header: value`}
          className="bg-workflow-bg border-workflow-border text-workflow-text resize-none"
          rows={3}
        />
      </div>
      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-2">
          <Label className="text-workflow-text">Timeout (s)</Label>
          <Input
            type="number"
            value={newTimeout}
            onChange={(e) => setNewTimeout(parseInt(e.target.value) || 30)}
            min={1}
            max={300}
            className="bg-workflow-bg border-workflow-border text-workflow-text"
          />
        </div>
        <div className="space-y-2">
          <Label className="text-workflow-text">Enabled</Label>
          <div className="flex items-center h-10">
            <Switch
              checked={newEnabled}
              onCheckedChange={setNewEnabled}
            />
          </div>
        </div>
      </div>
      <div className="space-y-2">
        <Label className="text-workflow-text">Description</Label>
        <Textarea
          value={newDescription}
          onChange={(e) => setNewDescription(e.target.value)}
          placeholder="Optional description..."
          className="bg-workflow-bg border-workflow-border text-workflow-text resize-none"
          rows={2}
        />
      </div>
    </div>
  );

  return (
    <div className="min-h-screen bg-workflow-bg p-6">
      <div className="max-w-7xl mx-auto">
        {/* Create Dialog */}
        <Dialog open={createOpen} onOpenChange={setCreateOpen}>
          <DialogContent className="sm:max-w-[600px] bg-workflow-surface border-workflow-border">
            <DialogHeader>
              <DialogTitle className="text-workflow-text">Add MCP Server</DialogTitle>
              <DialogDescription className="text-workflow-text-muted">
                Configure a new MCP server connection
              </DialogDescription>
            </DialogHeader>
            {formContent}
            <DialogFooter>
              <Button
                variant="outline"
                onClick={() => { resetForm(); setCreateOpen(false); }}
                className="border-workflow-border text-workflow-text"
              >
                Cancel
              </Button>
              <Button
                variant="outline"
                onClick={handleCreate}
                disabled={submitting || !newName || !newURL}
                className="border-workflow-border text-workflow-text"
              >
                {submitting && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
                Create
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        {/* Edit Dialog */}
        <Dialog open={editOpen} onOpenChange={setEditOpen}>
          <DialogContent className="sm:max-w-[600px] bg-workflow-surface border-workflow-border">
            <DialogHeader>
              <DialogTitle className="text-workflow-text">Edit MCP Server</DialogTitle>
              <DialogDescription className="text-workflow-text-muted">
                Update MCP server configuration
              </DialogDescription>
            </DialogHeader>
            {formContent}
            <DialogFooter>
              <Button
                variant="outline"
                onClick={() => { resetForm(); setEditOpen(false); setEditingServer(null); }}
                className="border-workflow-border text-workflow-text"
              >
                Cancel
              </Button>
              <Button
                variant="outline"
                onClick={handleUpdate}
                disabled={submitting || !newName || !newURL}
                className="border-workflow-border text-workflow-text"
              >
                {submitting && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
                Save Changes
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        {/* Delete Dialog */}
        <Dialog open={!!deleteId} onOpenChange={() => setDeleteId(null)}>
          <DialogContent className="sm:max-w-[400px] bg-workflow-surface border-workflow-border">
            <DialogHeader>
              <DialogTitle className="text-workflow-text">Delete MCP Server</DialogTitle>
              <DialogDescription className="text-workflow-text-muted">
                Are you sure you want to delete this MCP server? This action cannot be undone.
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
                placeholder="Search MCP servers..."
                className="pl-9"
              />
            </div>
          </div>
          {workspaceId && (
            <Button
              className="gap-2"
              onClick={() => setCreateOpen(true)}
            >
              <Plus className="w-4" />
              Add Server
            </Button>
          )}
        </div>

        {/* Content */}
        {loading ? (
          <div className="flex items-center justify-center py-12">
            <Loader2 className="h-8 w-8 animate-spin text-workflow-primary" />
          </div>
        ) : filteredServers.length === 0 ? (
          <div className="text-center py-12">
            <Puzzle className="h-12 w-12 mx-auto text-workflow-text-muted mb-4" />
            <h3 className="text-lg font-medium text-workflow-text">No MCP servers found</h3>
          </div>
        ) : (
          <div className="space-y-4">
            {filteredServers.map((srv) => (
              <Card key={srv.id} className="group hover:shadow-lg transition-all duration-200">
                <CardContent className="pt-4">
                  <div className="flex items-start justify-between">
                    <div className="flex items-center gap-3 flex-1">
                      <button
                        type="button"
                        onClick={() => toggleExpand(srv.id)}
                        className="p-2 rounded-lg bg-workflow-surface border border-workflow-border text-purple-400 hover:bg-workflow-surface-hover transition-colors"
                      >
                        {expandedServer === srv.id ? (
                          <ChevronDown className="h-5 w-5" />
                        ) : (
                          <ChevronRight className="h-5 w-5" />
                        )}
                      </button>
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-2">
                          <h3 className="text-lg font-semibold text-workflow-text truncate group-hover:text-primary transition-colors">
                            {srv.name}
                          </h3>
                          <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-mono ${
                            srv.enabled
                              ? "bg-green-500/10 text-green-400 border border-green-500/20"
                              : "bg-stone-500/10 text-stone-400 border border-stone-500/20"
                          }`}>
                            {srv.enabled ? "Enabled" : "Disabled"}
                          </span>
                          <span className="px-2 py-0.5 rounded-full text-xs font-mono bg-purple-500/10 text-purple-400 border border-purple-500/20">
                            {srv.type}
                          </span>
                        </div>
                        <p className="text-xs text-workflow-text-muted font-mono mt-0.5 truncate">
                          {srv.url}
                        </p>
                      </div>
                    </div>
                    <div className="flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => checkStatus(srv.id)}
                        className="h-8 w-8 p-0"
                        title="Check status"
                      >
                        <RefreshCw className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleToggleEnabled(srv)}
                        className="h-8 w-8 p-0"
                        title={srv.enabled ? "Disable" : "Enable"}
                      >
                        {srv.enabled ? <PowerOff className="h-4 w-4" /> : <Power className="h-4 w-4" />}
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => startEdit(srv)}
                        className="h-8 w-8 p-0"
                      >
                        <Edit className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setDeleteId(srv.id)}
                        className="h-8 w-8 p-0 text-workflow-text-muted hover:text-red-400"
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  </div>

                  {srv.description && (
                    <CardDescription className="text-workflow-text-muted mt-2 line-clamp-2 ml-12">
                      {srv.description}
                    </CardDescription>
                  )}

                  {serverStatus[srv.id] !== undefined && (
                    <div className={`mt-2 ml-12 text-xs font-mono ${serverStatus[srv.id] ? "text-green-400" : "text-red-400"}`}>
                      {serverStatus[srv.id] ? "Connected" : "Unreachable"}
                    </div>
                  )}

                  {/* Expanded tools list */}
                  {expandedServer === srv.id && (
                    <div className="mt-4 ml-12 border-t border-workflow-border pt-4">
                      <div className="flex items-center justify-between mb-3">
                        <h4 className="text-sm font-mono font-semibold text-workflow-text">
                          <Wrench className="w-3 h-3 inline mr-1" />
                          Tools
                        </h4>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => loadTools(srv.id)}
                          disabled={loadingTools[srv.id]}
                          className="h-6 text-xs"
                        >
                          {loadingTools[srv.id] ? <Loader2 className="h-3 w-3 animate-spin" /> : <RefreshCw className="h-3 w-3" />}
                        </Button>
                      </div>
                      {loadingTools[srv.id] ? (
                        <div className="flex items-center gap-2 text-workflow-text-muted text-sm py-4 justify-center">
                          <Loader2 className="h-4 w-4 animate-spin" />
                          Loading tools...
                        </div>
                      ) : (serverTools[srv.id] || []).length === 0 ? (
                        <p className="text-sm text-workflow-text-muted py-2">
                          No tools available. Make sure the server is running and enabled.
                        </p>
                      ) : (
                        <div className="space-y-2">
                          {(serverTools[srv.id] || []).map((tool) => (
                            <div
                              key={tool.name}
                              className="flex items-start gap-3 p-3 rounded-lg bg-workflow-bg/50 border border-workflow-border/50"
                            >
                              <Wrench className="w-4 h-4 text-purple-400 mt-0.5 shrink-0" />
                              <div className="min-w-0 flex-1">
                                <div className="text-sm font-mono font-medium text-workflow-text">
                                  {tool.name}
                                </div>
                                {tool.description && (
                                  <p className="text-xs text-workflow-text-muted mt-0.5">
                                    {tool.description}
                                  </p>
                                )}
                              </div>
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  )}
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
