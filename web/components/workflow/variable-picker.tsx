"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import type { WorkflowNode } from "@/lib/workflow-types";
import type { Edge } from "@xyflow/react";
import { NODE_OUTPUT_VARIABLES } from "./node-edit-panel";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { ChevronDown, ChevronRight, Hash, Search, Braces } from "lucide-react";

export interface VariableTreeNode {
  nodeId: string;
  nodeType: string;
  nodeLabel: string;
  variables: {
    name: string;
    type: string;
    description: string;
  }[];
}

function buildVariableTree(
  nodes: WorkflowNode[],
  currentNodeId: string,
  edges: Edge[]
): VariableTreeNode[] {
  // Collect ALL upstream ancestor nodes (all hops, not just one)
  const upstreamIds = new Set<string>();
  const queue: string[] = [];

  // Seed with direct predecessors
  edges.forEach((edge) => {
    if (edge.target === currentNodeId) {
      queue.push(edge.source);
    }
  });

  // BFS to collect all ancestors
  while (queue.length > 0) {
    const nodeId = queue.shift()!;
    if (upstreamIds.has(nodeId)) continue;
    upstreamIds.add(nodeId);
    edges.forEach((edge) => {
      if (edge.target === nodeId && !upstreamIds.has(edge.source)) {
        queue.push(edge.source);
      }
    });
  }

  return nodes
    .filter((node) => upstreamIds.has(node.id))
    .map((node) => {
      const vars = NODE_OUTPUT_VARIABLES[node.type] || [];
      return {
        nodeId: node.id,
        nodeType: node.type,
        nodeLabel: getNodeLabel(node),
        variables: vars,
      };
    })
    .filter((n) => n.variables.length > 0);
}

function getNodeLabel(node: WorkflowNode): string {
  const label = node.data?.label as string;
  if (label) return label;
  switch (node.type) {
    case "LLM": return "LLM Node";
    case "Prometheus": return "Prometheus Query";
    case "Jaeger": return "Jaeger Trace";
    case "Elasticsearch": return "Elasticsearch";
    case "Loki": return "Loki Log";
    case "ClickHouse": return "ClickHouse Query";
    case "Condition": return "Condition";
    case "Output": return "Output";
    case "Start": return "Start";
    case "GoScript": return "GoScript";
    default: return node.type;
  }
}

interface VariablePickerProps {
  nodes: WorkflowNode[];
  edges: Edge[];
  currentNodeId: string;
  onInsert: (variableRef: string) => void;
  trigger?: React.ReactNode;
  align?: "start" | "center" | "end";
  side?: "left" | "right" | "top" | "bottom";
}

export function VariablePicker({
  nodes,
  edges,
  currentNodeId,
  onInsert,
  trigger,
  align = "start",
  side = "right",
}: VariablePickerProps) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set());
  const searchRef = useRef<HTMLInputElement>(null);

  const variableTree = buildVariableTree(nodes, currentNodeId, edges);

  const filteredTree = search.trim()
    ? variableTree
        .map((node) => ({
          ...node,
          variables: node.variables.filter(
            (v) =>
              v.name.toLowerCase().includes(search.toLowerCase()) ||
              v.description.toLowerCase().includes(search.toLowerCase())
          ),
        }))
        .filter((n) => n.variables.length > 0)
    : variableTree;

  useEffect(() => {
    if (open && searchRef.current) {
      searchRef.current.focus();
    }
  }, [open]);

  const toggleNode = (nodeId: string) => {
    setExpandedNodes((prev) => {
      const next = new Set(prev);
      if (next.has(nodeId)) next.delete(nodeId);
      else next.add(nodeId);
      return next;
    });
  };

  const handleSelect = (nodeId: string, varName: string) => {
    onInsert(`{{${nodeId}.${varName}}}`);
    setOpen(false);
    setSearch("");
  };

  const nodeTypeColors: Record<string, string> = {
    LLM: "text-blue-400 bg-blue-400/10 border-blue-400/30",
    Prometheus: "text-orange-400 bg-orange-400/10 border-orange-400/30",
    Jaeger: "text-purple-400 bg-purple-400/10 border-purple-400/30",
    Elasticsearch: "text-teal-400 bg-teal-400/10 border-teal-400/30",
    Loki: "text-yellow-400 bg-yellow-400/10 border-yellow-400/30",
    ClickHouse: "text-green-400 bg-green-400/10 border-green-400/30",
    Condition: "text-pink-400 bg-pink-400/10 border-pink-400/30",
    Output: "text-red-400 bg-red-400/10 border-red-400/30",
    Start: "text-gray-400 bg-gray-400/10 border-gray-400/30",
    GoScript: "text-cyan-400 bg-cyan-400/10 border-cyan-400/30",
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        {trigger || (
          <button
            type="button"
            className="flex items-center gap-1 px-2 py-1 text-[10px] font-mono text-workflow-text-muted border border-workflow-border rounded hover:bg-workflow-surface-hover transition-colors duration-200"
            title="Insert variable (Cmd+/)"
          >
            <Braces className="h-3 w-3" />
            Vars
          </button>
        )}
      </PopoverTrigger>
      <PopoverContent
        align={align}
        side={side}
        sideOffset={4}
        className="w-80 p-0 bg-workflow-node-bg border-workflow-border shadow-xl"
        style={{ zIndex: 9999 }}
      >
        <div className="p-2 border-b border-workflow-border/50">
          <div className="flex items-center gap-2 px-2 py-1.5 bg-workflow-surface/50 rounded-lg">
            <Search className="h-3.5 w-3.5 text-workflow-text-subtle shrink-0" />
            <input
              ref={searchRef}
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Filter variables..."
              className="flex-1 bg-transparent text-xs font-mono text-workflow-text placeholder:text-workflow-text-subtle focus:outline-none"
            />
          </div>
        </div>

        <div className="max-h-72 overflow-y-auto p-1.5">
          {filteredTree.length === 0 ? (
            <div className="text-xs text-workflow-text-subtle font-mono text-center py-4">
              No variables found
            </div>
          ) : (
            <div className="space-y-0.5">
              {filteredTree.map((treeNode) => {
                const isExpanded = expandedNodes.has(treeNode.nodeId);
                const colorClass = nodeTypeColors[treeNode.nodeType] || "text-gray-400 bg-gray-400/10 border-gray-400/30";

                return (
                  <div key={treeNode.nodeId}>
                    <button
                      type="button"
                      onClick={() => toggleNode(treeNode.nodeId)}
                      className="w-full flex items-center gap-1.5 px-2 py-1.5 rounded-lg hover:bg-workflow-surface/50 transition-colors duration-150 text-left group"
                    >
                      {isExpanded ? (
                        <ChevronDown className="h-3 w-3 text-workflow-text-subtle shrink-0" />
                      ) : (
                        <ChevronRight className="h-3 w-3 text-workflow-text-subtle shrink-0" />
                      )}
                      <span className={`text-[10px] font-mono px-1.5 py-0.5 rounded border shrink-0 ${colorClass}`}>
                        {treeNode.nodeType}
                      </span>
                      <Hash className="h-3 w-3 text-workflow-text-subtle shrink-0 ml-auto text-[10px]" />
                      <span className="text-[10px] font-mono text-workflow-text-subtle">
                        {treeNode.nodeId}
                      </span>
                    </button>

                    {isExpanded && (
                      <div className="ml-5 pl-2 border-l border-workflow-border/30 space-y-0.5 py-0.5">
                        {treeNode.variables.map((variable) => (
                          <button
                            key={variable.name}
                            type="button"
                            onClick={() => handleSelect(treeNode.nodeId, variable.name)}
                            className="w-full text-left px-2 py-1.5 rounded-lg hover:bg-workflow-surface/50 transition-colors duration-150 group"
                            title={variable.description}
                          >
                            <div className="flex items-center gap-2">
                              <code className="text-[11px] font-mono text-primary bg-primary/10 px-1.5 py-0.5 rounded shrink-0 group-hover:bg-primary/20 transition-colors">
                                {variable.name}
                              </code>
                              <span className="text-[10px] font-mono text-workflow-text-subtle uppercase shrink-0">
                                {variable.type}
                              </span>
                            </div>
                            <p className="text-[10px] text-workflow-text-subtle mt-0.5 leading-relaxed">
                              {variable.description}
                            </p>
                            <p className="text-[10px] font-mono text-primary/70 mt-0.5">
                              {`{{${treeNode.nodeId}.${variable.name}}}`}
                            </p>
                          </button>
                        ))}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </div>

        <div className="p-2 border-t border-workflow-border/50">
          <p className="text-[10px] font-mono text-workflow-text-subtle text-center">
            Click to Vars reference
          </p>
        </div>
      </PopoverContent>
    </Popover>
  );
}

// Hook to handle Cmd+/ shortcut on a textarea
export function useVariablePickerShortcuts(
  textareaRef: React.RefObject<HTMLTextAreaElement | null>,
  onTrigger: () => void
) {
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "/") {
        e.preventDefault();
        onTrigger();
      }
    };

    const textarea = textareaRef.current;
    if (textarea) {
      textarea.addEventListener("keydown", handleKeyDown);
      return () => textarea.removeEventListener("keydown", handleKeyDown);
    }
  }, [textareaRef, onTrigger]);
}
