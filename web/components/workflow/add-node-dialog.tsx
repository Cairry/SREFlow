"use client";

import React, { useState, useEffect } from "react";

import type { WorkflowNode, WorkflowNodeType } from "@/lib/workflow-types";
import {
  Sparkles,
  GitBranch,
  FileOutput,
  Activity,
  Search,
  BarChart3,
  Code,
  X,
  Terminal,
  Globe,
  ChevronRight,
  Database,
  Zap,
  Puzzle,
  Wrench,
  Loader2,
} from "lucide-react";
import { MenuDialog } from "./ui/dialog";
import { mcpApi, MCPServer, MCPServerTool } from "@/lib/api/mcp";

type NodeCategory = {
  id: string;
  label: string;
  icon: React.ReactNode;
};

const CATEGORIES: NodeCategory[] = [
  { id: "datasource", label: "Data Source", icon: <Database className="w-3 h-3" /> },
  { id: "action", label: "Action", icon: <Zap className="w-3 h-3" /> },
  { id: "ai", label: "AI", icon: <Sparkles className="w-3 h-3" /> },
  { id: "logic", label: "Logic", icon: <Puzzle className="w-3 h-3" /> },
  { id: "output", label: "Output", icon: <FileOutput className="w-3 h-3" /> },
];

type NodeDef = {
  type: WorkflowNodeType;
  label: string;
  description: string;
  icon: React.ReactNode;
  category: string;
};

const NODE_DEFINITIONS: NodeDef[] = [
  // Data Source
  { type: "Prometheus", label: "Prometheus", description: "Query metrics with PromQL", icon: <Activity className="w-4 h-4" />, category: "datasource" },
  { type: "Jaeger", label: "Jaeger", description: "Query distributed traces", icon: <Search className="w-4 h-4" />, category: "datasource" },
  { type: "Elasticsearch", label: "Elasticsearch", description: "Search and analyze logs", icon: <Search className="w-4 h-4" />, category: "datasource" },
  { type: "Loki", label: "Loki", description: "Query logs with LogQL", icon: <Search className="w-4 h-4" />, category: "datasource" },
  { type: "ClickHouse", label: "ClickHouse", description: "Execute SQL queries", icon: <BarChart3 className="w-4 h-4" />, category: "datasource" },
  { type: "Kubernetes", label: "Kubernetes", description: "Query pod status and events", icon: <Activity className="w-4 h-4" />, category: "datasource" },
  // Action
  { type: "SSH", label: "SSH", description: "Execute commands via SSH", icon: <Terminal className="w-4 h-4" />, category: "action" },
  { type: "GoScript", label: "GoScript", description: "Execute GoScript code", icon: <Code className="w-4 h-4" />, category: "action" },
  { type: "HTTP", label: "HTTP", description: "Send HTTP requests to external endpoints", icon: <Globe className="w-4 h-4" />, category: "action" },
  // AI
  { type: "LLM", label: "LLM", description: "Call LLM models", icon: <Sparkles className="w-4 h-4" />, category: "ai" },
  // Logic
  { type: "Condition", label: "Condition", description: "Branch logic flow", icon: <GitBranch className="w-4 h-4" />, category: "logic" },
  // Output
  { type: "Output", label: "Output", description: "Export Output", icon: <FileOutput className="w-4 h-4" />, category: "output" },
];

interface AddNodeDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onAddNode: (nodeType: WorkflowNodeType, mcpData?: { serverId: string; serverName: string; toolName: string; description: string }) => void;
}

export function AddNodeDialog({ isOpen, onClose, onAddNode }: AddNodeDialogProps) {
  const [collapsed, setCollapsed] = useState<Set<string>>(new Set());
  const [activeTab, setActiveTab] = useState<"nodes" | "mcp">("nodes");
  const [mcpServers, setMcpServers] = useState<MCPServer[]>([]);
  const [mcpTools, setMcpTools] = useState<Record<string, MCPServerTool[]>>({});
  const [loadingServers, setLoadingServers] = useState(false);
  const [loadingTools, setLoadingTools] = useState<Record<string, boolean>>({});
  const [expandedServer, setExpandedServer] = useState<string | null>(null);

  useEffect(() => {
    if (isOpen && activeTab === "mcp") {
      loadMCPServers();
    }
  }, [isOpen, activeTab]);

  const loadMCPServers = async () => {
    setLoadingServers(true);
    try {
      const data = await mcpApi.list();
      const servers = (data.servers || []).filter((s: MCPServer) => s.enabled);
      setMcpServers(servers);
      // Auto-load tools for all enabled servers
      const toolsMap: Record<string, MCPServerTool[]> = {};
      for (const server of servers) {
        try {
          const toolData = await mcpApi.listTools(server.id);
          toolsMap[server.id] = toolData.tools || [];
        } catch {
          toolsMap[server.id] = [];
        }
      }
      setMcpTools(toolsMap);
    } catch (error) {
      console.error("Failed to load MCP servers:", error);
      setMcpServers([]);
    } finally {
      setLoadingServers(false);
    }
  };

  const toggle = (id: string) => {
    setCollapsed((prev) => {
      const next = new Set(prev);
      next.has(id) ? next.delete(id) : next.add(id);
      return next;
    });
  };

  const toggleServerExpand = (serverId: string) => {
    setExpandedServer(prev => prev === serverId ? null : serverId);
  };

  const handleMCPToolClick = (server: MCPServer, tool: MCPServerTool) => {
    onAddNode("MCP", {
      serverId: server.id,
      serverName: server.name,
      toolName: tool.name,
      description: tool.description,
    });
    onClose();
  };

  return (
    <MenuDialog
      isOpen={isOpen}
      onClose={onClose}
      className="absolute top-full left-2 mt-2 w-[320px] h-[680px]"
    >
      <div className="flex items-center justify-between px-4 py-2 border-b border-workflow-border">
        <span className="text-[10px] font-mono uppercase tracking-wider text-workflow-text-muted">
          Add Node
        </span>
        <button
          type="button"
          onClick={onClose}
          className="p-1 hover:bg-workflow-surface-hover rounded-lg transition-colors duration-200 text-workflow-text-muted hover:text-workflow-text"
        >
          <X className="w-3 h-3" />
        </button>
      </div>

      {/* Tabs */}
      <div className="flex border-b border-workflow-border">
        <button
          type="button"
          onClick={() => setActiveTab("nodes")}
          className={`flex-1 px-4 py-2 text-xs font-mono uppercase tracking-wider transition-colors duration-200 ${
            activeTab === "nodes"
              ? "text-workflow-text border-b-2 border-primary bg-workflow-surface-hover"
              : "text-workflow-text-muted hover:text-workflow-text"
          }`}
        >
          Nodes
        </button>
        <button
          type="button"
          onClick={() => setActiveTab("mcp")}
          className={`flex-1 px-4 py-2 text-xs font-mono uppercase tracking-wider transition-colors duration-200 ${
            activeTab === "mcp"
              ? "text-workflow-text border-b-2 border-purple-500 bg-workflow-surface-hover"
              : "text-workflow-text-muted hover:text-workflow-text"
          }`}
        >
          <Puzzle className="w-3 h-3 inline mr-1" />
          MCP
        </button>
      </div>

      <div className="py-1 max-h-[600px] overflow-y-auto">
        {activeTab === "nodes" ? (
          // Nodes tab - existing content
          CATEGORIES.map((cat) => {
            const nodes = NODE_DEFINITIONS.filter((n) => n.category === cat.id);
            const isCollapsed = collapsed.has(cat.id);
            return (
              <div key={cat.id}>
                <button
                  type="button"
                  onClick={() => toggle(cat.id)}
                  className="w-full flex items-center gap-2 px-4 py-1.5 hover:bg-workflow-surface-hover transition-colors duration-200 text-left"
                >
                  <ChevronRight className={`w-3 h-3 text-workflow-text-muted transition-transform duration-200 ${isCollapsed ? "" : "rotate-90"}`} />
                  <span className="text-workflow-text-muted mt-0.5">{cat.icon}</span>
                  <span className="text-xs font-mono text-workflow-text-muted">{cat.label}</span>
                  <span className="ml-auto text-[10px] text-workflow-text-subtle font-mono">{nodes.length}</span>
                </button>

                {!isCollapsed && nodes.map((node) => (
                  <button
                    key={node.type}
                    type="button"
                    onClick={() => {
                      onAddNode(node.type);
                      onClose();
                    }}
                    className="w-full flex items-start gap-3 pl-10 pr-4 py-2 hover:bg-workflow-surface-hover transition-colors duration-200 text-left"
                  >
                    <div className="text-workflow-text-muted mt-0.5">{node.icon}</div>
                    <div className="flex-1 min-w-0">
                      <div className="text-sm font-medium text-workflow-text">
                        {node.label}
                      </div>
                      <div className="text-xs text-workflow-text-muted mt-0.5">{node.description}</div>
                    </div>
                  </button>
                ))}
              </div>
            );
          })
        ) : (
          // MCP tab
          loadingServers ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-5 w-5 animate-spin text-workflow-text-muted" />
              <span className="ml-2 text-xs text-workflow-text-muted">Loading MCP servers...</span>
            </div>
          ) : mcpServers.length === 0 ? (
            <div className="px-4 py-8 text-center">
              <Puzzle className="w-8 h-8 mx-auto text-workflow-text-muted mb-2" />
              <p className="text-xs text-workflow-text-muted">No MCP servers configured</p>
              <a href="/mcp-tools" className="text-xs text-primary hover:underline mt-1 inline-block">
                Add MCP Server
              </a>
            </div>
          ) : (
            mcpServers.map((server) => {
              const tools = mcpTools[server.id] || [];
              const isExpanded = expandedServer === server.id;
              return (
                <div key={server.id}>
                  <button
                    type="button"
                    onClick={() => toggleServerExpand(server.id)}
                    className="w-full flex items-center gap-2 px-4 py-1.5 hover:bg-workflow-surface-hover transition-colors duration-200 text-left"
                  >
                    <ChevronRight className={`w-3 h-3 text-workflow-text-muted transition-transform duration-200 ${isExpanded ? "rotate-90" : ""}`} />
                    <Puzzle className="w-3 h-3 text-purple-400" />
                    <span className="text-xs font-mono text-workflow-text-muted">{server.name}</span>
                    <span className="ml-auto text-[10px] text-workflow-text-subtle font-mono">{tools.length}</span>
                  </button>

                  {isExpanded && tools.length === 0 && (
                    <div className="pl-10 pr-4 py-2 text-xs text-workflow-text-subtle">
                      No tools available
                    </div>
                  )}

                  {isExpanded && tools.map((tool) => (
                    <button
                      key={tool.name}
                      type="button"
                      onClick={() => handleMCPToolClick(server, tool)}
                      className="w-full flex items-start gap-3 pl-10 pr-4 py-2 hover:bg-workflow-surface-hover transition-colors duration-200 text-left"
                    >
                      <Wrench className="w-4 h-4 text-purple-400 mt-0.5 shrink-0" />
                      <div className="flex-1 min-w-0">
                        <div className="text-sm font-medium text-workflow-text">
                          {tool.name}
                        </div>
                        {tool.description && (
                          <div className="text-xs text-workflow-text-muted mt-0.5 line-clamp-2">{tool.description}</div>
                        )}
                      </div>
                    </button>
                  ))}
                </div>
              );
            })
          )
        )}
      </div>
    </MenuDialog>
  );
}
