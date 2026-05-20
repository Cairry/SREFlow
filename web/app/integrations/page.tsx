"use client";

import { useEffect, useState } from "react";
import {
  Database,
  Search,
  Loader2,
  Server,
  Bot,
  Activity,
  Logs,
  Edit,
  LayoutGrid,
  Terminal,
  Key,
  Lock,
} from "lucide-react";
import { toast } from "@/hooks/use-toast";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
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
import { integrationApi, Integration } from "@/lib/api/integration";

const INTEGRATION_TYPES = [
  { value: "Prometheus", label: "Prometheus", icon: Activity, color: "text-orange-400" },
  { value: "Jaeger", label: "Jaeger", icon: Server, color: "text-teal-400" },
  { value: "Clickhouse", label: "ClickHouse", icon: Database, color: "text-yellow-400" },
  { value: "Elasticsearch", label: "Elasticsearch", icon: Search, color: "text-blue-400" },
  { value: "Loki", label: "Loki", icon: Logs, color: "text-green-400" },
  { value: "LLM", label: "LLM", icon: Bot, color: "text-purple-400" },
  { value: "SSHAuth", label: "SSH Auth", icon: Terminal, color: "text-emerald-400" },
  { value: "Kubernetes", label: "Kubernetes", icon: Database, color: "text-cyan-400" },
];

const AUTH_TYPES = [
  { value: "none", label: "None" },
  { value: "basic", label: "Basic Auth" },
  { value: "bearer", label: "Bearer Token" },
  { value: "apikey", label: "API Key" },
];

const SSH_AUTH_TYPES = [
  { value: "ssh_password", label: "Password", icon: Lock },
  { value: "ssh_key", label: "Private Key", icon: Key },
];

export default function IntegrationsPage() {
  const [integrations, setIntegrations] = useState<Integration[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState("");
  const [filterType, setFilterType] = useState<string>("all");
  const [editingIntegration, setEditingIntegration] = useState<Integration | null>(null);
  const [editOpen, setEditOpen] = useState(false);
  const [workspaceId, setWorkspaceId] = useState<string>("");
  const [noWorkspace, setNoWorkspace] = useState(false);

  // Edit form state
  const [newName, setNewName] = useState("");
  const [newType, setNewType] = useState<string>("Prometheus");
  const [newEndpoint, setNewEndpoint] = useState("");
  const [newAuthType, setNewAuthType] = useState<string>("none");
  const [newToken, setNewToken] = useState("");
  const [newUsername, setNewUsername] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [newAIAPI, setNewAIAPI] = useState("");
  const [newAIKey, setNewAIKey] = useState("");
  const [newHeaders, setNewHeaders] = useState("");
  // SSHAuth specific state
  const [newSSHUsername, setNewSSHUsername] = useState("");
  const [newSSHAuthType, setNewSSHAuthType] = useState<string>("ssh_password");
  const [newSSHPassword, setNewSSHPassword] = useState("");
  const [newSSHPrivateKey, setNewSSHPrivateKey] = useState("");
  // Kubernetes specific state
  const [newKubeconfig, setNewKubeconfig] = useState("");
  const [testingK8s, setTestingK8s] = useState(false);
  const [k8sTestResult, setK8sTestResult] = useState<{ connected: boolean; serverVersion: string; namespaceCount: number } | null>(null);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    const stored = localStorage.getItem("flowforge-current-workspace-id");
    if (stored) {
      setWorkspaceId(stored);
    } else {
      setNoWorkspace(true);
    }
  }, []);

  const loadIntegrations = async () => {
    if (!workspaceId) {
      setNoWorkspace(true);
      return;
    }
    setNoWorkspace(false);
    setLoading(true);
    try {
      const data = await integrationApi.list();
      setIntegrations(data.integrations || []);
    } catch (error) {
      console.error("Failed to load integrations:", error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadIntegrations();
  }, [workspaceId]);

  const filteredIntegrations = integrations.filter((ds) => {
    const matchesSearch =
	      ds.name.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesType = filterType === "all" || ds.type === filterType;
    return matchesSearch && matchesType;
  });

  const getTypeInfo = (type: string) => {
    return INTEGRATION_TYPES.find((t) => t.value === type) || INTEGRATION_TYPES[0];
  };

  const resetForm = () => {
    setNewName("");
    setNewType("Prometheus");
    setNewEndpoint("");
    setNewAuthType("none");
    setNewToken("");
    setNewUsername("");
    setNewPassword("");
    setNewAIAPI("");
    setNewAIKey("");
    setNewHeaders("");
    setNewSSHUsername("");
    setNewSSHAuthType("ssh_password");
    setNewSSHPassword("");
    setNewSSHPrivateKey("");
    setNewKubeconfig("");
    setK8sTestResult(null);
  };

  const handleUpdate = async () => {
    if (!editingIntegration || !newName) return;

    const currentWorkspaceId = localStorage.getItem("flowforge-current-workspace-id");
    if (!currentWorkspaceId) {
      toast({ title: "Error", description: "Please select a workspace first", variant: "destructive" });
      setSubmitting(false);
      return;
    }

    setSubmitting(true);

    const isHTTPType = ["Prometheus", "Jaeger", "Clickhouse", "Elasticsearch", "Loki"].includes(newType);
    const body: Record<string, unknown> = {
      name: newName,
    };

    if (isHTTPType) {
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

      body.httpConfig = {
        endpoint: newEndpoint,
        headers: Object.keys(headersObj).length > 0 ? headersObj : undefined,
        auth: {
          type: newAuthType,
          token: newAuthType === "bearer" ? newToken : undefined,
          username: newAuthType === "basic" ? newUsername : undefined,
          password: newAuthType === "basic" ? newPassword : undefined,
        },
      };
    } else if (newType === "LLM") {
      body.aiConfig = {
        api: newAIAPI || "https://api.openai.com/v1",
        apiKey: newAIKey,
      };
    } else if (newType === "SSHAuth") {
      body.httpConfig = {
        auth: {
          type: newSSHAuthType,
          username: newSSHUsername,
          password: newSSHAuthType === "ssh_password" ? newSSHPassword : undefined,
          privateKey: newSSHAuthType === "ssh_key" ? newSSHPrivateKey : undefined,
        },
      };
    } else if (newType === "Kubernetes") {
      body.kubeConfig = {
        kubeconfig: newKubeconfig,
      };
    }

    try {
      await integrationApi.update(editingIntegration.id, body as Parameters<typeof integrationApi.update>[1]);
      resetForm();
      setEditOpen(false);
      setEditingIntegration(null);
      loadIntegrations();
    } catch (error) {
      toast({ title: "Failed to update integration", description: error instanceof Error ? error.message : 'Unknown error', variant: "destructive" });
    } finally {
      setSubmitting(false);
    }
  };

  const showHTTPConfig = () => ["Prometheus", "Jaeger", "Clickhouse", "Elasticsearch", "Loki"].includes(newType);
  const showAIConfig = () => newType === "LLM";
  const showSSHAuthConfig = () => newType === "SSHAuth";
  const showKubeConfig = () => newType === "Kubernetes";

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
                Please select or create a workspace from the navigation bar to start managing your integrations.
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
    <div className="min-h-screen bg-workflow-bg p-6">
      <div className="max-w-7xl mx-auto">
        {/* Header with Search */}
        <div className="mb-8 flex justify-between items-center gap-4">
          <div className="flex items-center gap-4 flex-1">
            <div className="relative max-w-sm flex-1">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-workflow-text-muted" />
              <Input
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                placeholder="Search integrations..."
                className="pl-9"
              />
            </div>
          </div>
        </div>

        {/* Content */}
        {loading ? (
          <div className="flex items-center justify-center py-12">
            <Loader2 className="h-8 w-8 animate-spin text-workflow-primary" />
          </div>
        ) : filteredIntegrations.length === 0 ? (
          <div className="text-center py-12">
            <Database className="h-12 w-12 mx-auto text-workflow-text-muted mb-4" />
            <h3 className="text-lg font-medium text-workflow-text">No integrations found</h3>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
            {filteredIntegrations.map((ds) => {
              const typeInfo = getTypeInfo(ds.type);
              const TypeIcon = typeInfo.icon;
              return (
                <Card key={ds.id} className="group hover:shadow-lg transition-all duration-200">
                  <CardContent className="pt-5">
                    <div className="flex items-start justify-between">
                      <div className="flex items-center gap-3 flex-1">
                        <div className={`p-2 rounded-lg bg-workflow-surface border border-workflow-border ${typeInfo.color}`}>
                          <TypeIcon className="h-5 w-5" />
                        </div>
                        <div className="min-w-0 flex-1">
                          <h3 className="text-lg font-semibold text-workflow-text truncate group-hover:text-primary transition-colors">
                            {ds.name}
                          </h3>
                        </div>
                      </div>
                      <div className="flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => {
                            setEditingIntegration(ds);
                            setNewName(ds.name);
                            setNewType(ds.type);
                            setNewEndpoint(ds.httpConfig?.endpoint || "");
                            setNewAuthType(ds.httpConfig?.auth?.type || "none");
                            setNewToken(ds.httpConfig?.auth?.token || "");
                            setNewUsername(ds.httpConfig?.auth?.username || "");
                            setNewHeaders(ds.httpConfig?.headers ? Object.entries(ds.httpConfig.headers).map(([k, v]) => `${k}: ${v}`).join('\n') : "");
                            setNewPassword(ds.httpConfig?.auth?.password || "");
                            setNewAIAPI(ds.aiConfig?.api || "");
                            setNewAIKey(ds.aiConfig?.apiKey || "");
                            // SSHAuth fields
                            setNewSSHUsername(ds.httpConfig?.auth?.username || "");
                            setNewSSHAuthType(ds.httpConfig?.auth?.type === "ssh_key" ? "ssh_key" : "ssh_password");
                            setNewSSHPassword(ds.httpConfig?.auth?.password || "");
                            setNewSSHPrivateKey(ds.httpConfig?.auth?.privateKey || "");
                            // Kubernetes fields
                            setNewKubeconfig(ds.kubeConfig?.kubeconfig || "");
                            setEditOpen(true);
                          }}
                          className="h-8 w-8 p-0"
                        >
                          <Edit className="h-4 w-4" />
                        </Button>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              );
            })}
          </div>
        )}
      </div>

      {/* Edit Dialog */}
      <Dialog open={editOpen} onOpenChange={setEditOpen}>
        <DialogContent className="sm:max-w-[600px] bg-workflow-surface border-workflow-border">
          <DialogHeader>
            <DialogTitle className="text-workflow-text">Edit Integration</DialogTitle>
            <DialogDescription className="text-workflow-text-muted">
              Update integration configuration
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4 max-h-[60vh] overflow-y-auto">
            <div className="space-y-2">
              <Label className="text-workflow-text">Name</Label>
              <Input
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                placeholder="My Name"
                className="bg-workflow-bg border-workflow-border text-workflow-text"
              />
            </div>
            <div className="space-y-2">
              {showHTTPConfig() && (
                <>
                  <div className="space-y-2">
                    <Label className="text-workflow-text">Endpoint URL</Label>
                    <Input
                      value={newEndpoint}
                      onChange={(e) => setNewEndpoint(e.target.value)}
                      placeholder="http://prometheus:9090"
                      className="bg-workflow-bg border-workflow-border text-workflow-text"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label className="text-workflow-text">Auth Type</Label>
                    <Select value={newAuthType} onValueChange={setNewAuthType}>
                      <SelectTrigger className="bg-workflow-bg border-workflow-border text-workflow-text">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent className="bg-workflow-surface border-workflow-border">
                        {AUTH_TYPES.map((auth) => (
                          <SelectItem key={auth.value} value={auth.value}>
                            {auth.label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                  {newAuthType === "bearer" && (
                    <div className="space-y-2">
                      <Label className="text-workflow-text">Token</Label>
                      <Input
                        value={newToken}
                        onChange={(e) => setNewToken(e.target.value)}
                        placeholder="Bearer token..."
                        type="password"
                        className="bg-workflow-bg border-workflow-border text-workflow-text"
                      />
                    </div>
                  )}
                  {newAuthType === "basic" && (
                    <>
                      <div className="space-y-2">
                        <Label className="text-workflow-text">Username</Label>
                        <Input
                          value={newUsername}
                          onChange={(e) => setNewUsername(e.target.value)}
                          placeholder="Username"
                          className="bg-workflow-bg border-workflow-border text-workflow-text"
                        />
                      </div>
                      <div className="space-y-2">
                        <Label className="text-workflow-text">Password</Label>
                        <Input
                          value={newPassword}
                          onChange={(e) => setNewPassword(e.target.value)}
                          placeholder="Password"
                          type="password"
                          className="bg-workflow-bg border-workflow-border text-workflow-text"
                        />
                      </div>
                    </>
                  )}
                </>
              )}
              
              {showHTTPConfig() && (
                <div className="space-y-2">
                  <Label className="text-workflow-text">Headers (Key: Value, one per line)</Label>
                  <Textarea
                    value={newHeaders}
                    onChange={(e) => setNewHeaders(e.target.value)}
                    placeholder={`Authorization: Bearer token123\nX-Custom-Header: value`}
                    className="bg-workflow-bg border-workflow-border text-workflow-text resize-none"
                    rows={3}
                  />
                  <p className="text-xs text-workflow-text-subtle mt-1">
                    Format: Key: Value (one header per line)
                  </p>
                </div>
              )}

              {showAIConfig() && (
                <>
                  <div className="space-y-2">
                    <Label className="text-workflow-text">API Endpoint</Label>
                    <Input
                      value={newAIAPI}
                      onChange={(e) => setNewAIAPI(e.target.value)}
                      placeholder="https://api.openai.com/v1"
                      className="bg-workflow-bg border-workflow-border text-workflow-text"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label className="text-workflow-text">API Key</Label>
                    <Input
                      value={newAIKey}
                      onChange={(e) => setNewAIKey(e.target.value)}
                      placeholder="sk-..."
                      type="password"
                      className="bg-workflow-bg border-workflow-border text-workflow-text"
                  />
                </div>
              </>
              )}

              {showSSHAuthConfig() && (
                <>
                  <div className="space-y-2">
                    <Label className="text-workflow-text">Username *</Label>
                    <Input
                      value={newSSHUsername}
                      onChange={(e) => setNewSSHUsername(e.target.value)}
                      placeholder="root"
                      className="bg-workflow-bg border-workflow-border text-workflow-text"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label className="text-workflow-text">Auth Method</Label>
                    <Select value={newSSHAuthType} onValueChange={setNewSSHAuthType}>
                      <SelectTrigger className="bg-workflow-bg border-workflow-border text-workflow-text">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent className="bg-workflow-surface border-workflow-border">
                        {SSH_AUTH_TYPES.map((auth) => {
                          const Icon = auth.icon;
                          return (
                            <SelectItem key={auth.value} value={auth.value}>
                              <div className="flex items-center gap-2">
                                <Icon className="h-4 w-4" />
                                <span>{auth.label}</span>
                              </div>
                            </SelectItem>
                          );
                        })}
                      </SelectContent>
                    </Select>
                  </div>
                  {newSSHAuthType === "ssh_password" && (
                    <div className="space-y-2">
                      <Label className="text-workflow-text">Password *</Label>
                      <Input
                        value={newSSHPassword}
                        onChange={(e) => setNewSSHPassword(e.target.value)}
                        placeholder="Enter password"
                        type="password"
                        className="bg-workflow-bg border-workflow-border text-workflow-text"
                      />
                    </div>
                  )}
                  {newSSHAuthType === "ssh_key" && (
                    <div className="space-y-2">
                      <Label className="text-workflow-text">Private Key *</Label>
                      <Textarea
                        value={newSSHPrivateKey}
                        onChange={(e) => setNewSSHPrivateKey(e.target.value)}
                        placeholder="-----BEGIN RSA PRIVATE KEY-----&#10;...&#10;-----END RSA PRIVATE KEY-----"
                        className="bg-workflow-bg border-workflow-border text-workflow-text resize-none font-mono text-sm"
                        rows={6}
                      />
                    </div>
                  )}
                </>
              )}

              {showKubeConfig() && (
                <div className="space-y-2">
                  <Label className="text-workflow-text">Kubeconfig *</Label>
                  <Textarea
                    value={newKubeconfig}
                    onChange={(e) => { setNewKubeconfig(e.target.value); setK8sTestResult(null); }}
                    placeholder="Paste your kubeconfig YAML content here..."
                    className="bg-workflow-bg border-workflow-border text-workflow-text resize-none font-mono text-sm"
                    rows={8}
                  />
                  <div className="flex items-center gap-3">
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      disabled={!newKubeconfig || testingK8s}
                      onClick={async () => {
                        setTestingK8s(true);
                        setK8sTestResult(null);
                        try {
                          const result = await integrationApi.testK8sConnection(newKubeconfig);
                          setK8sTestResult(result);
                          toast({ title: "Connection Successful", description: `K8s v${result.serverVersion}, ${result.namespaceCount} namespaces accessible` });
                        } catch (error) {
                          setK8sTestResult(null);
                          toast({ title: "Connection Failed", description: error instanceof Error ? error.message : "Unknown error", variant: "destructive" });
                        } finally {
                          setTestingK8s(false);
                        }
                      }}
                      className="border-workflow-border text-workflow-text"
                    >
                      {testingK8s ? <Loader2 className="h-4 w-4 mr-2 animate-spin" /> : null}
                      Test Connection
                    </Button>
                    {k8sTestResult && (
                      <span className="text-xs text-green-400 font-mono">
                        Connected (v{k8sTestResult.serverVersion})
                      </span>
                    )}
                  </div>
                </div>
              )}
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                resetForm();
                setEditOpen(false);
              }}
              className="border-workflow-border text-workflow-text"
            >
              Cancel
            </Button>
            <Button
              variant="outline"
              onClick={handleUpdate}
              disabled={submitting || !newName || (showHTTPConfig() && !newEndpoint) || (showAIConfig() && !newAIKey) || (showSSHAuthConfig() && (!newSSHUsername || (newSSHAuthType === "ssh_password" && !newSSHPassword) || (newSSHAuthType === "ssh_key" && !newSSHPrivateKey)))}
              className="border-workflow-border text-workflow-text"
            >
              {submitting && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
              Save Changes
            </Button>
          </DialogFooter>
          
        </DialogContent>
      </Dialog>
    </div>
  );
}
