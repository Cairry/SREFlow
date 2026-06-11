"use client";

import React, { useState } from "react"

import { cn } from "@/lib/utils";
import type { WorkflowNodeType } from "@/lib/workflow-types";
import { NODE_COLORS } from "@/lib/node-styles";
import {
  Sparkles,
  GitBranch,
  FileOutput,
  Activity,
  Search,
  Database,
  BarChart3,
  Code,
  Terminal,
  Globe,
  ChevronRight,
  Zap,
  Puzzle,
  Route,
  Bot,
} from "lucide-react";

interface NodePaletteProps {
  onDragStart: (event: React.DragEvent, nodeType: WorkflowNodeType) => void;
}

type NodeCategory = {
  id: string;
  label: string;
  icon: React.ReactNode;
};

const CATEGORIES: NodeCategory[] = [
  { id: "integration", label: "Integration", icon: <Database className="w-3 h-3" /> },
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
  // Integration
  { type: "Prometheus", label: "Prometheus", description: "Query metrics with PromQL", icon: <Activity className="w-4 h-4" />, category: "integration" },
  { type: "Jaeger", label: "Jaeger", description: "Search traces with tags", icon: <Search className="w-4 h-4" />, category: "integration" },
  { type: "Elasticsearch", label: "Elasticsearch", description: "Search and analyze logs", icon: <Search className="w-4 h-4" />, category: "integration" },
  { type: "Loki", label: "Loki", description: "Query logs with LogQL", icon: <Search className="w-4 h-4" />, category: "integration" },
  { type: "ClickHouse", label: "ClickHouse", description: "Execute SQL queries", icon: <BarChart3 className="w-4 h-4" />, category: "integration" },
  { type: "Kubernetes", label: "Kubernetes", description: "Query pod status and events", icon: <Activity className="w-4 h-4" />, category: "integration" },
  // Action
  { type: "SSH", label: "SSH", description: "Execute commands via SSH", icon: <Terminal className="w-4 h-4" />, category: "action" },
  { type: "GoScript", label: "GoScript", description: "Execute GoScript code", icon: <Code className="w-4 h-4" />, category: "action" },
  { type: "HTTP", label: "HTTP", description: "Send HTTP requests to external endpoints", icon: <Globe className="w-4 h-4" />, category: "action" },
  // AI
  { type: "LLM", label: "LLM", description: "Call LLM models", icon: <Sparkles className="w-4 h-4" />, category: "ai" },
  { type: "Agent", label: "Agent", description: "ReAct agent with tools and skills", icon: <Bot className="w-4 h-4" />, category: "ai" },
  // Logic
  { type: "Condition", label: "Condition", description: "Branch logic flow", icon: <GitBranch className="w-4 h-4" />, category: "logic" },
  { type: "SmartRouter", label: "SmartRouter", description: "AI-powered smart routing", icon: <Route className="w-4 h-4" />, category: "logic" },
  // Output
  { type: "Output", label: "Output", description: "Export Output", icon: <FileOutput className="w-4 h-4" />, category: "output" },
];

export function NodePalette({ onDragStart }: NodePaletteProps) {
  const [collapsed, setCollapsed] = useState<Set<string>>(new Set());

  const toggle = (id: string) => {
    setCollapsed((prev) => {
      const next = new Set(prev);
      next.has(id) ? next.delete(id) : next.add(id);
      return next;
    });
  };

  return (
    <div className="w-64 bg-workflow-bg border-r border-workflow-border flex flex-col h-full transition-colors duration-200">
      <div className="p-4 border-b border-workflow-border">
        <h2 className="text-sm font-semibold text-workflow-text font-mono">Nodes</h2>
        <p className="text-xs text-workflow-text-muted mt-1">Drag to canvas</p>
      </div>

      <div className="flex-1 overflow-y-auto p-3" >
        {CATEGORIES.map((cat) => {
          const nodes = NODE_DEFINITIONS.filter((n) => n.category === cat.id);
          const isCollapsed = collapsed.has(cat.id);
          return (
            <div key={cat.id} className="mb-2">
              <button
                type="button"
                onClick={() => toggle(cat.id)}
                className="w-full flex items-center gap-2 px-2 py-1.5 rounded hover:bg-workflow-surface-hover transition-colors duration-200 text-left"
              >
                <ChevronRight className={`w-3 h-3 text-workflow-text-muted transition-transform duration-200 ${isCollapsed ? "" : "rotate-90"}`} />
                <span className="text-workflow-text-muted">{cat.icon}</span>
                <span className="text-xs font-mono text-workflow-text-muted">{cat.label}</span>
                <span className="ml-auto text-[10px] text-workflow-text-subtle font-mono">{nodes.length}</span>
              </button>

              {!isCollapsed && (
                <div className="space-y-1.5 mt-1">
                  {nodes.map((node) => {
                    const colors = NODE_COLORS[node.type] || {
                      bg: "bg-workflow-node-bg",
                      border: "border-gray-500/50",
                      accent: "text-gray-600",
                      handle: "bg-gray-500 border-gray-600",
                    };
                    return (
                      <div
                        key={node.type}
                        draggable
                        onDragStart={(e) => onDragStart(e, node.type)}
                        className={cn(
                          "p-2.5 rounded-lg border-2 cursor-grab active:cursor-grabbing transition-all duration-200",
                          "hover:scale-[1.02] hover:shadow-md",
                          colors.bg,
                          colors.border,
                          "group ml-2"
                        )}
                      >
                        <div className="flex items-center gap-2">
                          <div className={cn(colors.accent)}>{node.icon}</div>
                          <div>
                            <div className={cn("text-sm font-medium font-mono", colors.accent)}>
                              {node.label}
                            </div>
                            <div className="text-xs text-workflow-text-muted">{node.description}</div>
                          </div>
                        </div>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          );
        })}
      </div>

      <div className="p-3 border-t border-workflow-border text-xs text-workflow-text-subtle font-mono">
        <div className="flex items-center gap-2">
          <kbd className="px-1.5 py-0.5 bg-workflow-node-input rounded transition-colors duration-200">Del</kbd>
          <span>Delete selected</span>
        </div>
      </div>
    </div>
  );
}
