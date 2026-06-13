"use client";

import { useState, useEffect, useRef } from "react";
import { loader } from "@monaco-editor/react";
import Editor from "@monaco-editor/react";
import type { WorkflowNode } from "@/lib/workflow-types";

loader.config({
  paths: {
    vs: "https://cdn.jsdelivr.net/npm/monaco-editor@0.52.0/min/vs",
  },
});
import { SidebarDialog, ModalDialog } from "./ui/dialog";
import { useToast } from "@/hooks/use-toast";
import { VariablePicker, useVariablePickerShortcuts } from "./variable-picker";
import { ChevronDown, ChevronRight, Variable, Sparkles, Braces, Maximize2, Minimize2 } from "lucide-react";
import { Loader2 } from "lucide-react";
import { mcpApi, type MCPServer, type MCPServerTool } from "@/lib/api/mcp";
import { integrationApi } from "@/lib/api/integration";
import { skillApi, type Skill } from "@/lib/api/skill";

// Node output variables definition (aligned with backend node return structures)
export const NODE_OUTPUT_VARIABLES: Record<string, { name: string; type: string; description: string }[]> = {
  LLM: [
    { name: "output", type: "string", description: "LLM generated text response" },
    { name: "latency", type: "float", description: "API response time in seconds" },
    { name: "usage.maxTokens", type: "int", description: "Max tokens configured for request" },
    { name: "usage.inputTokens", type: "int", description: "Tokens used in prompt" },
    { name: "usage.outputTokens", type: "int", description: "Tokens in completion" },
    { name: "usage.totalTokens", type: "int", description: "Total tokens used" },
    { name: "parsed", type: "object", description: "Parsed JSON from AI response (when response is valid JSON)" },
  ],
  Prometheus: [
    { name: "promQL", type: "string", description: "The PromQL query that was executed" },
    { name: "series", type: "array", description: "Time series data returned from query" },
  ],
  Jaeger: [
    { name: "traces", type: "array", description: "Array of processed trace objects" },
    { name: "trace_count", type: "int", description: "Number of traces returned" },
    { name: "raw_response", type: "object", description: "Raw Jaeger API response" },
  ],
  Elasticsearch: [
    { name: "data", type: "object", description: "Elasticsearch query response data" },
  ],
  Loki: [
    { name: "data.result_type", type: "string", description: "Result type: streams/logs/vector" },
    { name: "data.series", type: "array", description: "Processed log series data" },
    { name: "data.data", type: "object", description: "Raw Loki API response data" },
    { name: "query", type: "string", description: "The LogQL query that was executed" },
  ],
  ClickHouse: [
    { name: "data", type: "array/object", description: "ClickHouse SQL query result data" },
  ],
  SSH: [
    { name: "output", type: "string", description: "SSH command execution output" },
  ],
  Kubernetes: [
    { name: "pods", type: "array", description: "Pod list (when resourceType=status, pod=all)" },
    { name: "pod", type: "object", description: "Single pod status (when resourceType=status, pod=specific)" },
    { name: "events", type: "array", description: "Event list (when resourceType=event)" },
    { name: "count", type: "int", description: "Number of pods or events returned" },
    { name: "namespace", type: "string", description: "Namespace queried" },
  ],
  Condition: [
    { name: "branch", type: "string", description: "Condition result: 'true' or 'false', used for routing" },
  ],
  SmartRouter: [
    { name: "output", type: "string", description: "Selected branch ID" },
    { name: "selectedBranch", type: "string", description: "Selected branch ID (same as output)" },
    { name: "confidence", type: "float", description: "Confidence score (0.0-1.0)" },
    { name: "reasoning", type: "string", description: "LLM reasoning for the routing decision" },
    { name: "latency", type: "float", description: "LLM API response time in seconds" },
    { name: "usage.inputTokens", type: "int", description: "Tokens used in prompt" },
    { name: "usage.outputTokens", type: "int", description: "Tokens in completion" },
    { name: "usage.totalTokens", type: "int", description: "Total tokens used" },
  ],
  Start: [
    { name: "text", type: "string", description: "完整文本内容（当输入为纯字符串时使用 start.text 引用）" },
    { name: "xxx", type: "any", description: "当输入为 JSON 时，各字段直接展开，如 start.key1、start.key2" },
  ],
  GoScript: [
    { name: "output", type: "object", description: "User-defined output variable from script" },
    { name: "error", type: "string", description: "Error message (when execution fails)" },
    { name: "error_type", type: "string", description: "Error type classification (when execution fails)" },
    { name: "success", type: "bool", description: "Whether script executed successfully" },
  ],
  HTTP: [
    { name: "url", type: "string", description: "Target URL that was requested" },
    { name: "method", type: "string", description: "HTTP method used (GET/POST/etc.)" },
    { name: "requestBody", type: "string", description: "The resolved request body sent" },
    { name: "statusCode", type: "int", description: "HTTP response status code" },
    { name: "responseBody", type: "string", description: "Response body from the endpoint" },
    { name: "responseHeaders", type: "object", description: "HTTP response headers" },
  ],
  MCP: [
    { name: "isError", type: "bool", description: "Whether the MCP tool returned an error" },
    { name: "output", type: "array", description: "MCP tool execution result content list" },
  ],
  Agent: [
    { name: "output", type: "string", description: "Agent final response text" },
    { name: "iterations", type: "int", description: "Number of ReAct iterations (tool calls + 1)" },
    { name: "latency", type: "float", description: "Total execution time in seconds" },
    { name: "toolCalls", type: "array", description: "Array of tool call records (when tools were used)" },
  ],
};

interface NodeEditPanelProps {
  node: WorkflowNode | null;
  nodes: WorkflowNode[];
  edges: import("@xyflow/react").Edge[];
  onUpdate: (nodeId: string, data: Partial<WorkflowNode["data"]>) => void;
  onClose: () => void;
}

export function NodeEditPanel({ node, nodes, edges, onUpdate, onClose }: NodeEditPanelProps) {
  const { toast } = useToast();
  const [beautifyingSystem, setBeautifyingSystem] = useState(false);
  const [beautifyingPrompt, setBeautifyingPrompt] = useState(false);
  const [activePicker, setActivePicker] = useState<string | null>(null);
  const [mcpInputSchema, setMcpInputSchema] = useState<Record<string, unknown> | null>(null);
  const [loadingMcpSchema, setLoadingMcpSchema] = useState(false);
  const [activeTextPicker, setActiveTextPicker] = useState<string | null>(null);
  const [goscriptModalOpen, setGoscriptModalOpen] = useState(false);
  const [goscriptModalCode, setGoscriptModalCode] = useState("");
  const [availableSkills, setAvailableSkills] = useState<Skill[]>([]);
  const [mcpServers, setMcpServers] = useState<MCPServer[]>([]);
  const [mcpServerTools, setMcpServerTools] = useState<Record<string, MCPServerTool[]>>({});
  const [loadingMcpTools, setLoadingMcpTools] = useState<Record<string, boolean>>({});
  const [expandedMcpServer, setExpandedMcpServer] = useState<string | null>(null);

  const systemPromptRef = useRef<HTMLTextAreaElement>(null);
  const promptRef = useRef<HTMLTextAreaElement>(null);
  const goscriptRef = useRef<HTMLTextAreaElement>(null);
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const goscriptEditorRef = useRef<any>(null);
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const goscriptInlineEditorRef = useRef<any>(null);
  const promqlRef = useRef<HTMLTextAreaElement>(null);
  const esQueryRef = useRef<HTMLTextAreaElement>(null);
  const lokiQueryRef = useRef<HTMLTextAreaElement>(null);
  const chQueryRef = useRef<HTMLTextAreaElement>(null);
  const webhookBodyRef = useRef<HTMLTextAreaElement>(null);
  const conditionRef = useRef<HTMLInputElement>(null);
  const sshCommandRef = useRef<HTMLTextAreaElement>(null);
  const sshHostRef = useRef<HTMLInputElement>(null);
  const sshPortRef = useRef<HTMLInputElement>(null);
  const jaegerServiceRef = useRef<HTMLInputElement>(null);
  const jaegerTagsRef = useRef<HTMLInputElement>(null);
  const esIndexRef = useRef<HTMLInputElement>(null);
  const smartRouterInputRef = useRef<HTMLTextAreaElement>(null);

  useVariablePickerShortcuts(promptRef, () => setActivePicker(activePicker === "prompt" ? null : "prompt"));
  useVariablePickerShortcuts(systemPromptRef, () => setActivePicker(activePicker === "systemPrompt" ? null : "systemPrompt"));
  useVariablePickerShortcuts(goscriptRef, () => setActivePicker(activePicker === "goscript" ? null : "goscript"));
  useVariablePickerShortcuts(promqlRef, () => setActivePicker(activePicker === "promql" ? null : "promql"));
  useVariablePickerShortcuts(esQueryRef, () => setActivePicker(activePicker === "esQuery" ? null : "esQuery"));
  useVariablePickerShortcuts(lokiQueryRef, () => setActivePicker(activePicker === "lokiQuery" ? null : "lokiQuery"));
  useVariablePickerShortcuts(chQueryRef, () => setActivePicker(activePicker === "chQuery" ? null : "chQuery"));
  useVariablePickerShortcuts(webhookBodyRef, () => setActivePicker(activePicker === "webhookBody" ? null : "webhookBody"));
  useVariablePickerShortcuts(conditionRef, () => setActivePicker(activePicker === "condition" ? null : "condition"));
  useVariablePickerShortcuts(sshCommandRef, () => setActivePicker(activePicker === "sshCommand" ? null : "sshCommand"));
  useVariablePickerShortcuts(sshHostRef, () => setActiveTextPicker(activeTextPicker === "sshHost" ? null : "sshHost"));
  useVariablePickerShortcuts(sshPortRef, () => setActiveTextPicker(activeTextPicker === "sshPort" ? null : "sshPort"));
  useVariablePickerShortcuts(jaegerServiceRef, () => setActiveTextPicker(activeTextPicker === "jaegerService" ? null : "jaegerService"));
  useVariablePickerShortcuts(jaegerTagsRef, () => setActiveTextPicker(activeTextPicker === "jaegerTags" ? null : "jaegerTags"));
  useVariablePickerShortcuts(esIndexRef, () => setActiveTextPicker(activeTextPicker === "esIndex" ? null : "esIndex"));
  useVariablePickerShortcuts(smartRouterInputRef, () => setActivePicker(activePicker === "smartRouterInput" ? null : "smartRouterInput"));

  const insertVariable = (pickerKey: string, variableRef: string) => {
    if (pickerKey === "condition") {
      const input = conditionRef.current;
      if (!input) return;
      const start = input.selectionStart;
      const end = input.selectionEnd;
      const currentValue = input.value;
      input.value = currentValue.slice(0, start) + variableRef + currentValue.slice(end);
      onUpdate(node!.id, { condition: input.value });
      setActivePicker(null);
      setTimeout(() => {
        input.focus();
        input.selectionStart = input.selectionEnd = start + variableRef.length;
      }, 0);
      return;
    }

    const refMap: Record<string, React.RefObject<HTMLTextAreaElement | null>> = {
      prompt: promptRef,
      systemPrompt: systemPromptRef,
      goscript: goscriptRef,
      promql: promqlRef,
      esQuery: esQueryRef,
      lokiQuery: lokiQueryRef,
      chQuery: chQueryRef,
      webhookBody: webhookBodyRef,
      sshCommand: sshCommandRef,
      smartRouterInput: smartRouterInputRef,
    };
    const textareaRef = refMap[pickerKey];

    // GoScript 内联 Monaco Editor：用 Monaco API 插入变量
    if (pickerKey === "goscript" && goscriptInlineEditorRef.current) {
      const editor = goscriptInlineEditorRef.current;
      const selection = editor.getSelection();
      const model = editor.getModel();
      if (selection && model) {
        editor.executeEdits("", [{
          range: selection,
          text: variableRef,
          forceMoveMarkers: true,
        }]);
        editor.focus();
      }
      onUpdate(node!.id, { code: model.getValue() });
      setActivePicker(null);
      return;
    }

    if (!textareaRef?.current) return;
    const textarea = textareaRef.current;
    const start = textarea.selectionStart;
    const end = textarea.selectionEnd;
    const currentValue = textarea.value;
    const newValue = currentValue.slice(0, start) + variableRef + currentValue.slice(end);
    const fieldMap: Record<string, string> = {
      prompt: "prompt", systemPrompt: "systemPrompt",
      goscript: "code", promql: "query", esQuery: "query",
      lokiQuery: "query", chQuery: "query", webhookBody: "webhook",
      sshCommand: "command", smartRouterInput: "input",
    };
    const field = fieldMap[pickerKey];
    if (field === "webhook") {
      if (node?.type === "HTTP") {
        onUpdate(node!.id, { body: newValue });
      } else {
        const webhook = { ...((node?.data as any)?.webhook || {}), body: currentValue.slice(0, start) + variableRef + currentValue.slice(end) };
        onUpdate(node!.id, { webhook });
      }
    } else {
      onUpdate(node!.id, { [field]: newValue });
    }
    setActivePicker(null);
    setTimeout(() => {
      textarea.focus();
      textarea.selectionStart = textarea.selectionEnd = start + variableRef.length;
    }, 0);
  };

  // Handle variable insertion into plain <input> text fields (host, port)
  const insertTextVariable = (pickerKey: string, variableRef: string) => {
    const refMap: Record<string, React.RefObject<HTMLInputElement | null>> = {
      sshHost: sshHostRef,
      sshPort: sshPortRef,
      jaegerService: jaegerServiceRef,
      jaegerTags: jaegerTagsRef,
      esIndex: esIndexRef,
    };
    const inputRef = refMap[pickerKey];
    if (!inputRef?.current) return;
    const input = inputRef.current;
    const start = input.selectionStart;
    const end = input.selectionEnd;
    const currentValue = input.value;
    input.value = currentValue.slice(0, start) + variableRef + currentValue.slice(end);
    const fieldMap: Record<string, string> = {
      sshHost: "host",
      sshPort: "port",
      jaegerService: "service",
      jaegerTags: "tags",
      esIndex: "index",
    };
    onUpdate(node!.id, { [fieldMap[pickerKey]]: input.value });
    setActiveTextPicker(null);
    setTimeout(() => {
      input.focus();
      input.selectionStart = input.selectionEnd = start + variableRef.length;
    }, 0);
  };

  useEffect(() => {
    if (node?.type !== "MCP") {
      setMcpInputSchema(null);
      return;
    }
    const nodeData = node.data as Record<string, unknown>;
    const serverId = nodeData?.serverId as string;
    const toolName = nodeData?.toolName as string;
    if (!serverId || !toolName) {
      setMcpInputSchema(null);
      return;
    }
    setLoadingMcpSchema(true);
    mcpApi.listTools(serverId)
      .then((data) => {
        const tool = (data.tools || []).find((t: { name: string }) => t.name === toolName);
        setMcpInputSchema((tool?.input_schema as Record<string, unknown>) || null);
      })
      .catch(() => setMcpInputSchema(null))
      .finally(() => setLoadingMcpSchema(false));
  }, [node?.id, node?.type, (node?.data as Record<string, unknown>)?.serverId, (node?.data as Record<string, unknown>)?.toolName]);

  // Load available skills for Agent node
  useEffect(() => {
    skillApi.list()
      .then((data) => setAvailableSkills((data.skills || []).filter((s: Skill) => s.enabled)))
      .catch(() => setAvailableSkills([]));
  }, []);

  // Load MCP servers for Agent node
  useEffect(() => {
    mcpApi.list()
      .then((data) => setMcpServers((data.servers || []).filter((s: MCPServer) => s.enabled)))
      .catch(() => setMcpServers([]));
  }, []);

  // Load tools for expanded MCP server (Agent node)
  useEffect(() => {
    if (!expandedMcpServer || mcpServerTools[expandedMcpServer]) return;
    setLoadingMcpTools(prev => ({ ...prev, [expandedMcpServer]: true }));
    mcpApi.listTools(expandedMcpServer)
      .then((data) => setMcpServerTools(prev => ({ ...prev, [expandedMcpServer]: data.tools || [] })))
      .catch(() => {})
      .finally(() => setLoadingMcpTools(prev => ({ ...prev, [expandedMcpServer]: false })));
  }, [expandedMcpServer, mcpServerTools]);

  const beautifyPrompt = async (field: "systemPrompt" | "prompt", currentValue: string) => {
    if (!currentValue.trim()) return;

    if (field === "systemPrompt") setBeautifyingSystem(true);
    else setBeautifyingPrompt(true);

    try {
      // Auto-resolve LLM integration from workspace
      const data = await integrationApi.list("LLM");
      const llmIntegration = (data.integrations || [])[0];
      if (!llmIntegration?.aiConfig?.api || !llmIntegration?.aiConfig?.apiKey) {
        console.error("No LLM integration configured");
        return;
      }
      const aiConfig = llmIntegration.aiConfig;

      const response = await fetch(`${aiConfig.api}/chat/completions`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Authorization": `Bearer ${aiConfig.apiKey}`,
        },
        body: JSON.stringify({
          model: (node.data as Record<string, unknown>)?.model || "gpt-4o",
          messages: [
            {
              role: "user",
              content: `Please improve the following prompt. Keep the improved version concise, clear, and well-structured. Only return the improved prompt, nothing else.\n\n${currentValue}`
            }
          ],
        }),
      });

      if (!response.ok) {
        throw new Error(`API request failed: ${response.status}`);
      }

      const result = await response.json();
      const improved = result.choices?.[0]?.message?.content;

      if (improved) {
        handleChange(field, improved.trim());
      }
    } catch (error) {
      console.error("Beautify failed:", error);
    } finally {
      if (field === "systemPrompt") setBeautifyingSystem(false);
      else setBeautifyingPrompt(false);
    }
  };

  if (!node) return null;

  if (!node.data) {
    (node as any).data = {};
  }

  const handleChange = (key: string, value: unknown) => {
    const currentData = node.data || {};
    onUpdate(node.id, { ...currentData, [key]: value });
  };

  const safeGetValue = (field: string, defaultValue: any = '') => {
    const data = node.data || {};
    return data[field] !== undefined ? data[field] : defaultValue;
  };

  const inputClass = "w-full bg-workflow-surface/50 border border-workflow-border rounded-lg px-3 py-2 text-sm text-workflow-text font-mono focus:outline-none focus:ring-1 focus:ring-primary/50 transition-colors duration-200";
  const selectClass = "w-full bg-workflow-surface/50 border border-workflow-border rounded-lg px-3 py-2 text-sm text-workflow-text font-mono focus:outline-none focus:ring-1 focus:ring-primary/50 transition-colors duration-200 cursor-pointer";
  const textareaClass = "w-full bg-workflow-surface/50 border border-workflow-border rounded-lg px-3 py-2 text-sm text-workflow-text font-mono focus:outline-none focus:ring-1 focus:ring-primary/50 transition-colors duration-200 resize-none";
  const labelClass = "block text-xs text-workflow-text-muted font-mono mb-1.5";

  return (
    <SidebarDialog
      isOpen={true}
      onClose={onClose}
      title={node.type}
      nodeId={node.id}
      onCopyId={(id) => toast({ title: "Copied", description: `Node ID "${id}" copied to clipboard` })}
    >
      {/* LLM Node */}
      {node.type === "LLM" && (
        <>

          <div>
            <label className={labelClass}>Model</label>
            <ModelSelector value={safeGetValue("model", "") as string} onChange={(v) => handleChange("model", v)} />
          </div>

          <div>
            <div className="flex items-center justify-between mb-1.5">
              <label className={labelClass + " mb-0"}>System Prompt</label>
              <div className="flex items-center gap-1">
                <VariablePicker
                  nodes={nodes}
                  edges={edges}
                  currentNodeId={node.id}
                  onInsert={(ref) => insertVariable("systemPrompt", ref)}
                  trigger={
                    <button type="button" className="flex items-center gap-1 px-2 py-1 text-[10px] font-mono text-workflow-text-muted border border-workflow-border rounded hover:bg-workflow-surface-hover transition-colors duration-200">
                      <Braces className="h-3 w-3" />
                      Vars
                    </button>
                  }
                />
                <button
                  type="button"
                  onClick={() => beautifyPrompt("systemPrompt", safeGetValue("systemPrompt", ""))}
                  disabled={beautifyingSystem}
                  className="flex items-center gap-1 px-2 py-1 text-[10px] font-mono text-workflow-text-muted border border-workflow-border rounded hover:bg-workflow-surface-hover transition-colors duration-200"
                  title="Beautify with AI"
                >
                  {beautifyingSystem ? (
                    <Loader2 className="h-3 w-3 animate-spin" />
                  ) : (
                    <Sparkles className="h-3 w-3" />
                  )}
                  Beautify
                </button>
              </div>
            </div>
            <textarea
              ref={systemPromptRef}
              value={safeGetValue('systemPrompt', '')}
              onChange={(e) => handleChange("systemPrompt", e.target.value)}
              placeholder="Optional system instructions..."
              className={`${textareaClass} h-20`}
            />
          </div>
          <div>
            <div className="flex items-center justify-between mb-1.5">
              <label className={labelClass + " mb-0"}>Prompt</label>
              <div className="flex items-center gap-1">
                <VariablePicker
                  nodes={nodes}
                  edges={edges}
                  currentNodeId={node.id}
                  onInsert={(ref) => insertVariable("prompt", ref)}
                  trigger={
                    <button type="button" className="flex items-center gap-1 px-2 py-1 text-[10px] font-mono text-workflow-text-muted border border-workflow-border rounded hover:bg-workflow-surface-hover transition-colors duration-200">
                      <Braces className="h-3 w-3" />
                      Vars
                    </button>
                  }
                />
                <button
                  type="button"
                  onClick={() => beautifyPrompt("prompt", safeGetValue("prompt", ""))}
                  disabled={beautifyingPrompt}
                  className="flex items-center gap-1 px-2 py-1 text-[10px] font-mono text-workflow-text-muted border border-workflow-border rounded hover:bg-workflow-surface-hover transition-colors duration-200"
                  title="Beautify with AI"
                >
                  {beautifyingPrompt ? (
                    <Loader2 className="h-3 w-3 animate-spin" />
                  ) : (
                    <Sparkles className="h-3 w-3" />
                  )}
                  Beautify
                </button>
              </div>
            </div>
            <textarea
              ref={promptRef}
              value={safeGetValue('prompt', '')}
              onChange={(e) => handleChange("prompt", e.target.value)}
              placeholder="Enter prompt..."
              className={`${textareaClass} h-28`}
            />
          </div>
        </>
      )}

      {/* Prometheus Node */}
      {node.type === "Prometheus" && (
        <>

          <div>
            <div className="flex items-center justify-between mb-1.5">
              <label className={labelClass + " mb-0"}>PromQL Query</label>
              <VariablePicker
                nodes={nodes}
                edges={edges}
                currentNodeId={node.id}
                onInsert={(ref) => insertVariable("promql", ref)}
                trigger={
                  <button type="button" className="flex items-center gap-1 px-2 py-1 text-[10px] font-mono text-workflow-text-muted border border-workflow-border rounded hover:bg-workflow-surface-hover transition-colors duration-200">
                    <Braces className="h-3 w-3" />
                    Vars
                  </button>
                }
              />
            </div>
            <textarea
              ref={promqlRef}
              value={safeGetValue('query', 'up')}
              onChange={(e) => handleChange("query", e.target.value)}
              placeholder="up"
              className={`${textareaClass} h-20`}
            />
          </div>
          <div>
            <label className={labelClass}>Timeout (seconds)</label>
            <input
              type="number"
              value={safeGetValue('timeout', 30)}
              onChange={(e) => handleChange("timeout", parseInt(e.target.value) || 30)}
              placeholder="30"
              min="1"
              max="300"
              className={inputClass}
            />
          </div>
        </>
      )}

      {/* Jaeger Node */}
      {node.type === "Jaeger" && (
        <>
          <div>
            <div className="flex items-center justify-between mb-1.5">
              <label className={labelClass + " mb-0"}>Service</label>
              <VariablePicker
                nodes={nodes}
                edges={edges}
                currentNodeId={node.id}
                onInsert={(ref) => insertTextVariable("jaegerService", ref)}
                trigger={
                  <button type="button" className="flex items-center gap-1 px-2 py-1 text-[10px] font-mono text-workflow-text-muted border border-workflow-border rounded hover:bg-workflow-surface-hover transition-colors duration-200">
                    <Braces className="h-3 w-3" />
                    Vars
                  </button>
                }
              />
            </div>
            <input
              ref={jaegerServiceRef}
              type="text"
              value={(node.data as any).service || ""}
              onChange={(e) => handleChange("service", e.target.value)}
              placeholder="service-name"
              className={inputClass}
            />
          </div>
          <div>
            <div className="flex items-center justify-between mb-1.5">
              <label className={labelClass + " mb-0"}>Tags</label>
              <VariablePicker
                nodes={nodes}
                edges={edges}
                currentNodeId={node.id}
                onInsert={(ref) => insertTextVariable("jaegerTags", ref)}
                trigger={
                  <button type="button" className="flex items-center gap-1 px-2 py-1 text-[10px] font-mono text-workflow-text-muted border border-workflow-border rounded hover:bg-workflow-surface-hover transition-colors duration-200">
                    <Braces className="h-3 w-3" />
                    Vars
                  </button>
                }
              />
            </div>
            <input
              ref={jaegerTagsRef}
              type="text"
              value={(node.data as any).tags || ""}
              onChange={(e) => handleChange("tags", e.target.value)}
              placeholder="http.status_code=5.*"
              className={inputClass}
            />
          </div>
        </>
      )}

      {/* Elasticsearch Node */}
      {node.type === "Elasticsearch" && (
        <>

          <div>
            <div className="flex items-center justify-between mb-1.5">
              <label className={labelClass + " mb-0"}>Index</label>
              <VariablePicker
                nodes={nodes}
                edges={edges}
                currentNodeId={node.id}
                onInsert={(ref) => insertTextVariable("esIndex", ref)}
                trigger={
                  <button type="button" className="flex items-center gap-1 px-2 py-1 text-[10px] font-mono text-workflow-text-muted border border-workflow-border rounded hover:bg-workflow-surface-hover transition-colors duration-200">
                    <Braces className="h-3 w-3" />
                    Vars
                  </button>
                }
              />
            </div>
            <input
              ref={esIndexRef}
              type="text"
              value={(node.data as any).index || ''}
              onChange={(e) => handleChange("index", e.target.value)}
              placeholder="log-*"
              className={inputClass}
            />
          </div>
          <div>
            <div className="flex items-center justify-between mb-1.5">
              <label className={labelClass + " mb-0"}>Query</label>
              <VariablePicker
                nodes={nodes}
                edges={edges}
                currentNodeId={node.id}
                onInsert={(ref) => insertVariable("esQuery", ref)}
                trigger={
                  <button type="button" className="flex items-center gap-1 px-2 py-1 text-[10px] font-mono text-workflow-text-muted border border-workflow-border rounded hover:bg-workflow-surface-hover transition-colors duration-200">
                    <Braces className="h-3 w-3" />
                    Vars
                  </button>
                }
              />
            </div>
            <textarea
              ref={esQueryRef}
              value={(node.data as any).query || ''}
              onChange={(e) => handleChange("query", e.target.value)}
              placeholder='{"match": {"message": "error"}}'
              className={`${textareaClass} h-20`}
            />
          </div>
        </>
      )}

      {/* Loki Node */}
      {node.type === "Loki" && (
        <>

          <div>
            <div className="flex items-center justify-between mb-1.5">
              <label className={labelClass + " mb-0"}>LogQL Query</label>
              <VariablePicker
                nodes={nodes}
                edges={edges}
                currentNodeId={node.id}
                onInsert={(ref) => insertVariable("lokiQuery", ref)}
                trigger={
                  <button type="button" className="flex items-center gap-1 px-2 py-1 text-[10px] font-mono text-workflow-text-muted border border-workflow-border rounded hover:bg-workflow-surface-hover transition-colors duration-200">
                    <Braces className="h-3 w-3" />
                    Vars
                  </button>
                }
              />
            </div>
            <textarea
              ref={lokiQueryRef}
              value={(node.data as any).query || ''}
              onChange={(e) => handleChange("query", e.target.value)}
              placeholder='{job="varlogs"}'
              className={`${textareaClass} h-20`}
            />
          </div>
        </>
      )}

      {/* ClickHouse Node */}
      {node.type === "ClickHouse" && (
        <>
          <div>
            <label className={labelClass}>Database</label>
            <input
              type="text"
              value={(node.data as any).database || ''}
              onChange={(e) => handleChange("database", e.target.value)}
              placeholder="default"
              className={inputClass}
            />
          </div>
          <div>
            <div className="flex items-center justify-between mb-1.5">
              <label className={labelClass + " mb-0"}>SQL Query</label>
              <VariablePicker
                nodes={nodes}
                edges={edges}
                currentNodeId={node.id}
                onInsert={(ref) => insertVariable("chQuery", ref)}
                trigger={
                  <button type="button" className="flex items-center gap-1 px-2 py-1 text-[10px] font-mono text-workflow-text-muted border border-workflow-border rounded hover:bg-workflow-surface-hover transition-colors duration-200">
                    <Braces className="h-3 w-3" />
                    Vars
                  </button>
                }
              />
            </div>
            <textarea
              ref={chQueryRef}
              value={(node.data as any).query || ''}
              onChange={(e) => handleChange("query", e.target.value)}
              placeholder="SELECT * FROM table_name"
              className={`${textareaClass} h-24`}
            />
          </div>
        </>
      )}

      {/* SSH Node */}
      {node.type === "SSH" && (
        <>
          <div>
            <div className="flex items-center justify-between mb-1.5">
              <label className={labelClass + " mb-0"}>Host</label>
              <VariablePicker
                nodes={nodes}
                edges={edges}
                currentNodeId={node.id}
                onInsert={(ref) => insertTextVariable("sshHost", ref)}
                trigger={
                  <button type="button" className="flex items-center gap-1 px-2 py-1 text-[10px] font-mono text-workflow-text-muted border border-workflow-border rounded hover:bg-workflow-surface-hover transition-colors duration-200">
                    <Braces className="h-3 w-3" />
                    Vars
                  </button>
                }
              />
            </div>
            <input
              ref={sshHostRef}
              type="text"
              value={safeGetValue("host", "")}
              onChange={(e) => handleChange("host", e.target.value)}
              placeholder="192.168.1.1"
              className={inputClass}
            />
          </div>

          <div>
            <div className="flex items-center justify-between mb-1.5">
              <label className={labelClass + " mb-0"}>Port</label>
              <VariablePicker
                nodes={nodes}
                edges={edges}
                currentNodeId={node.id}
                onInsert={(ref) => insertTextVariable("sshPort", ref)}
                trigger={
                  <button type="button" className="flex items-center gap-1 px-2 py-1 text-[10px] font-mono text-workflow-text-muted border border-workflow-border rounded hover:bg-workflow-surface-hover transition-colors duration-200">
                    <Braces className="h-3 w-3" />
                    Vars
                  </button>
                }
              />
            </div>
            <input
              ref={sshPortRef}
              type="text"
              value={safeGetValue("port", 22)}
              onChange={(e) => handleChange("port", e.target.value)}
              placeholder="22"
              className={inputClass}
            />
          </div>

          <div>
            <div className="flex items-center justify-between mb-1.5">
              <label className={labelClass + " mb-0"}>Command</label>
              <VariablePicker
                nodes={nodes}
                edges={edges}
                currentNodeId={node.id}
                onInsert={(ref) => insertVariable("sshCommand", ref)}
                trigger={
                  <button type="button" className="flex items-center gap-1 px-2 py-1 text-[10px] font-mono text-workflow-text-muted border border-workflow-border rounded hover:bg-workflow-surface-hover transition-colors duration-200">
                    <Braces className="h-3 w-3" />
                    Vars
                  </button>
                }
              />
            </div>
            <textarea
              ref={sshCommandRef}
              value={safeGetValue("command", "")}
              onChange={(e) => handleChange("command", e.target.value)}
              placeholder="ls -la /home"
              className={`${textareaClass} h-24`}
            />
          </div>
          <div>
            <label className={labelClass}>Timeout (seconds)</label>
            <input
              type="number"
              value={safeGetValue("timeout", 30)}
              onChange={(e) => handleChange("timeout", parseInt(e.target.value) || 30)}
              placeholder="30"
              min="1"
              max="300"
              className={inputClass}
            />
          </div>
        </>
      )}

      {/* Kubernetes Node */}
      {node.type === "Kubernetes" && (
        <>
          <K8sNamespaceSelector
            namespace={safeGetValue("namespace", "")}
            onNamespaceChange={(val) => handleChange("namespace", val)}
          />

          <K8sPodSelector
            namespace={safeGetValue("namespace", "")}
            pod={safeGetValue("pod", "all")}
            onPodChange={(val) => handleChange("pod", val)}
          />

          <div>
            <label className={labelClass}>Resource Type</label>
            <select
              value={safeGetValue("resourceType", "status")}
              onChange={(e) => handleChange("resourceType", e.target.value)}
              className={selectClass}
            >
              <option value="status">Pod Status</option>
              <option value="event">Events</option>
            </select>
          </div>
        </>
      )}

      {/* Condition Node */}
      {node.type === "Condition" && (
        <>
          <div>
            <div className="flex items-center justify-between mb-1.5">
              <label className={labelClass + " mb-0"}>Variable</label>
              <VariablePicker
                nodes={nodes}
                edges={edges}
                currentNodeId={node.id}
                onInsert={(ref) => insertVariable("condition", ref)}
                trigger={
                  <button type="button" className="flex items-center gap-1 px-2 py-1 text-[10px] font-mono text-workflow-text-muted border border-workflow-border rounded hover:bg-workflow-surface-hover transition-colors duration-200">
                    <Braces className="h-3 w-3" />
                    Vars
                  </button>
                }
              />
            </div>
            <input
              ref={conditionRef}
              type="text"
              value={(node.data as any).condition || ''}
              onChange={(e) => handleChange("condition", e.target.value)}
              placeholder="{{input}}"
              className={inputClass}
            />
          </div>
          <div>
            <label className={labelClass}>Operator</label>
            <select
              value={(node.data as any).operator || 'contains'}
              onChange={(e) => handleChange("operator", e.target.value)}
              className={selectClass}
            >
              <option value="contains">contains</option>
              <option value="equals">equals</option>
              <option value="notEquals">not equals</option>
              <option value="greaterThan">greater than</option>
              <option value="lessThan">less than</option>
              <option value="isEmpty">is empty</option>
              <option value="isNotEmpty">is not empty</option>
            </select>
          </div>
          {((node.data as any).operator || 'contains') !== 'isEmpty' && ((node.data as any).operator || 'contains') !== 'isNotEmpty' && (
            <div>
              <label className={labelClass}>Value</label>
              <input
                type="text"
                value={(node.data as any).value || ''}
                onChange={(e) => handleChange("value", e.target.value)}
                placeholder="compare value"
                className={inputClass}
              />
            </div>
          )}
        </>
      )}

      {/* SmartRouter Node */}
      {node.type === "SmartRouter" && (
        <>
          <div className="bg-amber-500/5 border border-amber-500/20 rounded-lg px-3 py-2">
            <p className="text-[11px] text-amber-600 dark:text-amber-400 font-mono">
              AI analyzes upstream data and automatically selects the best branch.
              Each branch maps to an output port on the right side of the node.
            </p>
          </div>
          <div>
            <label className={labelClass}>Model</label>
            <ModelSelector value={safeGetValue("model", "") as string} onChange={(v) => handleChange("model", v)} />
          </div>
          <div>
            <div className="flex items-center justify-between mb-1.5">
              <label className={labelClass + " mb-0"}>Input</label>
              <VariablePicker
                nodes={nodes}
                edges={edges}
                currentNodeId={node.id}
                onInsert={(ref) => insertVariable("smartRouterInput", ref)}
                trigger={
                  <button type="button" className="flex items-center gap-1 px-2 py-1 text-[10px] font-mono text-workflow-text-muted border border-workflow-border rounded hover:bg-workflow-surface-hover transition-colors duration-200">
                    <Braces className="h-3 w-3" />
                    Vars
                  </button>
                }
              />
            </div>
            <textarea
              ref={smartRouterInputRef}
              value={safeGetValue('input', '')}
              onChange={(e) => handleChange("input", e.target.value)}
              placeholder="Optional: specify input data for LLM routing, e.g. {{prometheus_1.series}}. Leave empty to use all upstream outputs."
              className={`${textareaClass} h-20`}
            />
          </div>
          <div>
            <label className={labelClass}>Routing Branches</label>
            <SmartRouterBranchEditor
              branches={safeGetValue("branches", "[]")}
              onChange={(val) => handleChange("branches", val)}
            />
          </div>
        </>
      )}

      {/* Output Node */}
      {node.type === "Output" && (
        <>
          <div>
            <label className={labelClass}>Output Type</label>
            <select
              value={(node.data as any).outputType || 'webhook'}
              onChange={(e) => handleChange("outputType", e.target.value)}
              className={selectClass}
            >
              <option value="file">File Output</option>
              <option value="webhook">Webhook</option>
            </select>
          </div>

          {/* Webhook Configuration */}
          {(node.data as any).outputType === 'webhook' && (
            <div className="space-y-3 border-t border-workflow-border pt-3 mt-3">
              <p className="text-xs text-workflow-text-muted font-mono font-medium">Webhook Configuration</p>

              <div>
                <label className={labelClass}>URL</label>
                <input
                  type="text"
                  value={(node.data as any).webhook?.url || ''}
                  onChange={(e) => handleChange("webhook", {
                    ...(node.data as any).webhook || {},
                    url: e.target.value
                  })}
                  placeholder="https://example.com/webhook"
                  className={inputClass}
                />
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className={labelClass}>Method</label>
                  <select
                    value={(node.data as any).webhook?.method || 'POST'}
                    onChange={(e) => handleChange("webhook", {
                      ...(node.data as any).webhook || {},
                      method: e.target.value
                    })}
                    className={selectClass}
                  >
                    <option value="GET">GET</option>
                    <option value="POST">POST</option>
                    <option value="PUT">PUT</option>
                    <option value="PATCH">PATCH</option>
                  </select>
                </div>
                <div>
                  <label className={labelClass}>Timeout (s)</label>
                  <input
                    type="number"
                    value={(node.data as any).webhook?.timeout || 30}
                    onChange={(e) => handleChange("webhook", {
                      ...(node.data as any).webhook || {},
                      timeout: parseInt(e.target.value) || 30
                    })}
                    min="1"
                    max="300"
                    className={inputClass}
                  />
                </div>
              </div>

              <div>
                <label className={labelClass}>Headers (JSON)</label>
                <textarea
                  value={JSON.stringify((node.data as any).webhook?.headers || { "Content-Type": "application/json" }, null, 2)}
                  onChange={(e) => {
                    try {
                      const headers = JSON.parse(e.target.value);
                      handleChange("webhook", {
                        ...(node.data as any).webhook || {},
                        headers
                      });
                    } catch {
                      // Invalid JSON, don't update
                    }
                  }}
                  placeholder='{"Content-Type": "application/json"}'
                  className={`${textareaClass} h-16`}
                />
              </div>

              <div>
                <div className="flex items-center justify-between mb-1.5">
                  <label className={labelClass + " mb-0"}>Body</label>
                  <VariablePicker
                    nodes={nodes}
                    edges={edges}
                    currentNodeId={node.id}
                    onInsert={(ref) => insertVariable("webhookBody", ref)}
                    trigger={
                      <button type="button" className="flex items-center gap-1 px-2 py-1 text-[10px] font-mono text-workflow-text-muted border border-workflow-border rounded hover:bg-workflow-surface-hover transition-colors duration-200">
                        <Braces className="h-3 w-3" />
                        Vars
                      </button>
                    }
                  />
                </div>
                <textarea
                  ref={webhookBodyRef}
                  value={(node.data as any).webhook?.body || '{\n  "data": {{input}}\n}'}
                  onChange={(e) => handleChange("webhook", {
                    ...(node.data as any).webhook || {},
                    body: e.target.value
                  })}
                  placeholder='{"data": {{input}}}'
                  className={`${textareaClass} h-28`}
                />
              </div>
            </div>
          )}
        </>
      )}

      {/* Start Node */}
      {node.type === "Start" && (
        <div>
          <label className={labelClass}>Text Content</label>
          <textarea
            value={safeGetValue('text', '')}
            onChange={(e) => handleChange("text", e.target.value)}
            placeholder={"纯文本示例: service-a is down\nJSON 示例: {\"service\": \"a\", \"error\": \"timeout\"}"}
            className={`${textareaClass} h-40`}
          />
        </div>
      )}

      {/* GoScript Node */}
      {node.type === "GoScript" && (
        <>
          <div>
            <label className={labelClass}>Timeout (ms)</label>
            <input
              type="number"
              min="1000"
              step="1000"
              value={safeGetValue('timeout', 5000)}
              onChange={(e) => handleChange("timeout", parseInt(e.target.value) || 5000)}
              className={inputClass}
            />
          </div>
          <div>
            <div className="flex items-center justify-between mb-1.5">
              <label className={labelClass + " mb-0"}>GoScript Code</label>
              <div className="flex items-center gap-1">
                <VariablePicker
                  nodes={nodes}
                  edges={edges}
                  currentNodeId={node.id}
                  onInsert={(ref) => insertVariable("goscript", ref)}
                  trigger={
                    <button type="button" 
                      className="flex items-center gap-1 px-2 py-1 text-[10px] font-mono text-workflow-text-muted border border-workflow-border rounded hover:bg-workflow-surface-hover transition-colors duration-200"
                    >
                      <Braces className="h-3 w-3" />
                      Vars
                    </button>
                  }
                />
                <button
                  type="button"
                  onClick={() => {
                    setGoscriptModalCode(safeGetValue("code", ""));
                    setGoscriptModalOpen(true);
                  }}
                  className="flex items-center gap-1 px-2 py-1 text-[10px] font-mono text-workflow-text-muted border border-workflow-border rounded hover:bg-workflow-surface-hover transition-colors duration-200"
                  title="Expand to full editor"
                >
                  <Maximize2 className="h-3 w-3" />
                  Expand
                </button>
              </div>
            </div>
            <div className="rounded-lg overflow-hidden border border-workflow-border">
              <Editor
                height="240px"
                defaultLanguage="go"
                value={safeGetValue("code", "")}
                onChange={(value) => handleChange("code", value || "")}
                onMount={(editor) => { goscriptInlineEditorRef.current = editor; }}
                theme="vs-dark"
                options={{
                  fontSize: 13,
                  minimap: { enabled: false },
                  scrollBeyondLastLine: false,
                  lineNumbers: "on",
                  glyphMargin: false,
                  folding: true,
                  lineDecorationsWidth: 0,
                  lineNumbersMinChars: 3,
                  padding: { top: 8, bottom: 8 },
                  wordWrap: "on",
                  automaticLayout: true,
                }}
              />
            </div>
          </div>

          {/* Full-screen editor modal */}
          <ModalDialog
            isOpen={goscriptModalOpen}
            onClose={() => setGoscriptModalOpen(false)}
            title="GoScript Code Editor"
            description={
            <div className="flex-1 flex">
              <span style={{marginTop: "5px"}}>Full-screen Go code editor and click </span>
              <VariablePicker
                nodes={nodes}
                edges={edges}
                currentNodeId={node.id}
                onInsert={(ref) => {
                  if (goscriptEditorRef.current) {
                    const editor = goscriptEditorRef.current;
                    const selection = editor.getSelection();
                    const model = editor.getModel();
                    if (selection && model) {
                      editor.executeEdits("", [{
                        range: selection,
                        text: ref,
                        forceMoveMarkers: true,
                      }]);
                      editor.focus();
                    }
                  }
                  setActivePicker(null);
                }}
                trigger={
                  <button type="button" className="flex items-center gap-1 px-2 py-1 text-[10px] font-mono text-workflow-text-muted border border-workflow-border rounded hover:bg-workflow-surface-hover transition-colors duration-200">
                    <Braces className="h-3 w-3" />
                    Vars
                  </button>
                }
              />
              <span style={{marginTop: "5px"}}>to add variable</span>
            </div>
            }
            width="full"
          >
            <div className="h-full flex flex-col">
              <div className="flex-1 rounded-lg overflow-hidden border border-workflow-border">

                <Editor
                  height="100%"
                  defaultLanguage="go"
                  value={goscriptModalCode}
                  onChange={(value) => setGoscriptModalCode(value || "")}
                  onMount={(editor) => { goscriptEditorRef.current = editor; }}
                  theme="vs-dark"
                  options={{
                    fontSize: 14,
                    minimap: { enabled: true },
                    scrollBeyondLastLine: false,
                    lineNumbers: "on",
                    glyphMargin: false,
                    folding: true,
                    lineDecorationsWidth: 0,
                    lineNumbersMinChars: 3,
                    padding: { top: 12, bottom: 12 },
                    wordWrap: "off",
                    automaticLayout: true,
                  }}
                />
              </div>
              <div className="flex justify-end gap-2 pt-3 border-t border-workflow-border shrink-0">
                <button
                  type="button"
                  onClick={() => setGoscriptModalOpen(false)}
                  className="px-4 py-2 text-sm font-mono text-workflow-text-muted hover:text-workflow-text transition-colors"
                >
                  Cancel
                </button>
                <button
                  type="button"
                  onClick={() => {
                    handleChange("code", goscriptModalCode);
                    setGoscriptModalOpen(false);
                  }}
                  className="px-4 py-2 text-sm font-mono text-white hover:text-workflow-text transition-colors"
                >
                  Apply
                </button>
              </div>
            </div>
          </ModalDialog>
        </>
      )}

      {/* HTTP Node */}
      {node.type === "HTTP" && (
        <>
          <div>
            <label className={labelClass}>URL</label>
            <input
              type="text"
              value={safeGetValue("url", "")}
              onChange={(e) => handleChange("url", e.target.value)}
              placeholder="https://example.com/webhook"
              className={inputClass}
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className={labelClass}>Method</label>
              <select
                value={safeGetValue("method", "POST")}
                onChange={(e) => handleChange("method", e.target.value)}
                className={selectClass}
              >
                <option value="GET">GET</option>
                <option value="POST">POST</option>
                <option value="PUT">PUT</option>
                <option value="PATCH">PATCH</option>
              </select>
            </div>
            <div>
              <label className={labelClass}>Timeout (s)</label>
              <input
                type="number"
                value={safeGetValue("timeout", 30)}
                onChange={(e) => handleChange("timeout", parseInt(e.target.value) || 30)}
                min="1"
                max="300"
                className={inputClass}
              />
            </div>
          </div>
          <div>
            <label className={labelClass}>Headers (JSON)</label>
            <textarea
              value={JSON.stringify(safeGetValue("headers", { "Content-Type": "application/json" }), null, 2)}
              onChange={(e) => {
                try {
                  handleChange("headers", JSON.parse(e.target.value));
                } catch {
                  // invalid JSON, skip
                }
              }}
              placeholder='{"Content-Type": "application/json"}'
              className={`${textareaClass} h-16`}
            />
          </div>
          <div>
            <div className="flex items-center justify-between mb-1.5">
              <label className={labelClass + " mb-0"}>Body</label>
              <VariablePicker
                nodes={nodes}
                edges={edges}
                currentNodeId={node.id}
                onInsert={(ref) => insertVariable("webhookBody", ref)}
                trigger={
                  <button type="button" className="flex items-center gap-1 px-2 py-1 text-[10px] font-mono text-workflow-text-muted border border-workflow-border rounded hover:bg-workflow-surface-hover transition-colors duration-200">
                    <Braces className="h-3 w-3" />
                    Vars
                  </button>
                }
              />
            </div>
            <textarea
              ref={webhookBodyRef}
              value={safeGetValue("body", "{\n  \"data\": {{input}}\n}")}
              onChange={(e) => handleChange("body", e.target.value)}
              placeholder='{"data": {{input}}}'
              className={`${textareaClass} h-28`}
            />
          </div>
        </>
      )}

      {/* MCP Node */}
      {node.type === "MCP" && (
        <>
          <div>
            <label className={labelClass}>Server</label>
            <input
              type="text"
              value={safeGetValue("serverName", "")}
              readOnly
              className={`${inputClass} opacity-60 cursor-not-allowed`}
            />
          </div>
          <div>
            <label className={labelClass}>Tool</label>
            <input
              type="text"
              value={safeGetValue("toolName", "")}
              readOnly
              className={`${inputClass} opacity-60 cursor-not-allowed`}
            />
          </div>
          {safeGetValue("description", "") && (
            <div>
              <label className={labelClass}>Description</label>
              <p className="text-xs text-workflow-text font-mono bg-workflow-surface/30 rounded-lg px-3 py-2 border border-workflow-border">
                {safeGetValue("description", "")}
              </p>
            </div>
          )}
          {(() => {
            if (loadingMcpSchema) {
              return (
                <div>
                  <label className={labelClass}>Parameters</label>
                  <div className="flex items-center gap-2 text-workflow-text-muted text-xs py-2">
                    <Loader2 className="h-3 w-3 animate-spin" />
                    Loading parameters...
                  </div>
                </div>
              );
            }
            const schema = mcpInputSchema;
            if (!schema || typeof schema !== "object") return null;
            const properties = schema.properties as Record<string, { type?: string; description?: string; [key: string]: unknown }> | undefined;
            const required = (schema.required as string[]) || [];
            if (!properties || Object.keys(properties).length === 0) return null;
            return (
              <div>
                <label className={labelClass}>Parameters</label>
                <div className="bg-workflow-surface/30 rounded-lg border border-workflow-border overflow-hidden">
                  <div className="overflow-x-auto">
                    <table className="w-full text-xs font-mono whitespace-nowrap">
                      <thead>
                        <tr className="border-b border-workflow-border text-workflow-text-muted text-left">
                          <th className="px-3 py-1.5 font-normal">Name</th>
                          <th className="px-3 py-1.5 font-normal">Type</th>
                          <th className="px-3 py-1.5 font-normal">Required</th>
                          <th className="px-3 py-1.5 font-normal">Description</th>
                        </tr>
                      </thead>
                      <tbody>
                        {Object.entries(properties).map(([name, prop]) => (
                          <tr key={name} className="border-b border-workflow-border/50 last:border-b-0">
                            <td className="px-3 py-1.5">
                              <code className="text-primary bg-primary/10 px-1 py-0.5 rounded text-[11px]">{name}</code>
                            </td>
                            <td className="px-3 py-1.5 text-workflow-text-subtle">{prop.type || "any"}</td>
                            <td className="px-3 py-1.5">
                              {required.includes(name) ? (
                                <span className="text-amber-400 text-[10px]">Required</span>
                              ) : (
                                <span className="text-workflow-text-subtle text-[10px]">Optional</span>
                              )}
                            </td>
                            <td className={`px-3 text-workflow-text-muted ${(prop.description || "").length > 50 ? "max-w-[50ch] whitespace-normal" : "whitespace-nowrap"}`}>{prop.description || "-"}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>
              </div>
            );
          })()}
          <div>
            <label className={labelClass}>Arguments (JSON)</label>
            <textarea
              value={(() => {
                const args = safeGetValue("arguments", {});
                if (typeof args === "string") return args;
                return JSON.stringify(args, null, 2);
              })()}
              onChange={(e) => {
                try {
                  const parsed = JSON.parse(e.target.value);
                  handleChange("arguments", parsed);
                } catch {
                  handleChange("arguments", e.target.value);
                }
              }}
              placeholder='{"key": "value"}'
              className={`${textareaClass} h-32 font-mono`}
            />
          </div>
        </>
      )}

      {/* Agent Node */}
      {node.type === "Agent" && (
        <>
          <div>
            <label className={labelClass}>Model</label>
            <ModelSelector value={safeGetValue("model", "") as string} onChange={(v) => handleChange("model", v)} />
          </div>

          <div>
            <label className={labelClass}>Max Iterations</label>
            <input
              type="number"
              min={1}
              max={50}
              value={safeGetValue("maxIterations", 10) as number}
              onChange={(e) => handleChange("maxIterations", parseInt(e.target.value) || 10)}
              className={inputClass}
            />
          </div>

          <div>
            <label className={labelClass}>Skills</label>
            <SkillSelector
              selectedIds={(safeGetValue("skillIds", []) as string[]) || []}
              onChange={(ids) => handleChange("skillIds", ids)}
              skills={availableSkills}
            />
          </div>

          <div>
            <label className={labelClass}>MCP Tools</label>
            <MCPToolSelector
              selectedTools={(safeGetValue("selectedTools", []) as string[]) || []}
              onChange={(tools) => handleChange("selectedTools", tools)}
              servers={mcpServers}
              serverTools={mcpServerTools}
              loadingTools={loadingMcpTools}
              expandedServer={expandedMcpServer}
              onExpandServer={setExpandedMcpServer}
            />
          </div>

          <div>
            <div className="flex items-center justify-between mb-1.5">
              <label className={labelClass + " mb-0"}>Prompt</label>
              <div className="flex items-center gap-1">
                <VariablePicker
                  nodes={nodes}
                  edges={edges}
                  currentNodeId={node.id}
                  onInsert={(ref) => insertVariable("prompt", ref)}
                  trigger={
                    <button type="button" className="flex items-center gap-1 px-2 py-1 text-[10px] font-mono text-workflow-text-muted border border-workflow-border rounded hover:bg-workflow-surface-hover transition-colors duration-200">
                      <Braces className="h-3 w-3" />
                      Vars
                    </button>
                  }
                />
              </div>
            </div>
            <textarea
              value={safeGetValue('prompt', '')}
              onChange={(e) => handleChange("prompt", e.target.value)}
              placeholder="User prompt for the agent..."
              className={`${textareaClass} h-20`}
            />
          </div>
        </>
      )}

      {/* Output Variables Section */}
      {node.type !== "Output" && (
        <div className="border-t border-workflow-border pt-4 mt-4">
          <OutputVariables nodeType={node.type} />
        </div>
      )}
    </SidebarDialog>
  );
}

// ModelSelector — dynamically loads available LLM models from integration
function ModelSelector({ value, onChange }: {
  value: string;
  onChange: (val: string) => void;
}) {
  const [models, setModels] = useState<{ id: string; name: string }[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    setLoading(true);
    setError("");
    integrationApi.listModels()
      .then((data) => setModels(data.models || []))
      .catch((err) => {
        setError(err instanceof Error ? err.message : "Failed to load models");
        setModels([]);
      })
      .finally(() => setLoading(false));
  }, []);

  const selectClass = "w-full bg-workflow-surface/50 border border-workflow-border rounded-lg px-3 py-2 text-sm text-workflow-text font-mono focus:outline-none focus:ring-1 focus:ring-primary/50 transition-colors duration-200 cursor-pointer";

  if (loading) {
    return (
      <div className="flex items-center gap-2 text-workflow-text-muted text-sm">
        <Loader2 className="h-4 w-4 animate-spin" />
        Loading models...
      </div>
    );
  }

  if (error) {
    return (
      <div className="text-xs text-red-400 font-mono">
        {error}
      </div>
    );
  }

  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className={selectClass}
    >
      <option value="">-- Select Model --</option>
      {models.map((m) => (
        <option key={m.id} value={m.id}>{m.name}</option>
      ))}
    </select>
  );
}

// K8sNamespaceSelector — dynamically loads namespaces from K8s integration
function K8sNamespaceSelector({ namespace, onNamespaceChange }: {
  namespace: string;
  onNamespaceChange: (val: string) => void;
}) {
  const [namespaces, setNamespaces] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);
  const [resolvedId, setResolvedId] = useState("");

  useEffect(() => {
    setLoading(true);
    integrationApi.list("Kubernetes")
      .then((data) => {
        const k8s = (data.integrations || [])[0];
        if (!k8s) { setNamespaces([]); return; }
        setResolvedId(k8s.id);
        return integrationApi.getNamespaces(k8s.id);
      })
      .then((data) => setNamespaces(data.namespaces || []))
      .catch(() => setNamespaces([]))
      .finally(() => setLoading(false));
  }, []);

  const selectClass = "w-full bg-workflow-surface/50 border border-workflow-border rounded-lg px-3 py-2 text-sm text-workflow-text font-mono focus:outline-none focus:ring-1 focus:ring-primary/50 transition-colors duration-200 cursor-pointer";
  const labelClass = "block text-xs text-workflow-text-muted font-mono mb-1.5";

  return (
    <div>
      <label className={labelClass}>Namespace</label>
      {loading ? (
        <div className="flex items-center gap-2 text-workflow-text-muted text-sm">
          <Loader2 className="h-4 w-4 animate-spin" />
          Loading namespaces...
        </div>
      ) : (
        <select
          value={namespace}
          onChange={(e) => onNamespaceChange(e.target.value)}
          className={selectClass}
        >
          <option value="">-- Select Namespace --</option>
          {namespaces.map((ns) => (
            <option key={ns} value={ns}>{ns}</option>
          ))}
        </select>
      )}
    </div>
  );
}

// K8sPodSelector — dynamically loads pods from K8s integration
function K8sPodSelector({ namespace, pod, onPodChange }: {
  namespace: string;
  pod: string;
  onPodChange: (val: string) => void;
}) {
  const [pods, setPods] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);
  const [resolvedId, setResolvedId] = useState("");

  useEffect(() => {
    if (!namespace) {
      setPods([]);
      return;
    }
    setLoading(true);
    integrationApi.list("Kubernetes")
      .then((data) => {
        const k8s = (data.integrations || [])[0];
        if (!k8s) { setPods([]); return; }
        setResolvedId(k8s.id);
        return integrationApi.getPods(k8s.id, namespace);
      })
      .then((data) => setPods(data.pods || []))
      .catch(() => setPods([]))
      .finally(() => setLoading(false));
  }, [namespace]);

  const selectClass = "w-full bg-workflow-surface/50 border border-workflow-border rounded-lg px-3 py-2 text-sm text-workflow-text font-mono focus:outline-none focus:ring-1 focus:ring-primary/50 transition-colors duration-200 cursor-pointer";
  const labelClass = "block text-xs text-workflow-text-muted font-mono mb-1.5";

  return (
    <div>
      <label className={labelClass}>Pod</label>
      {loading ? (
        <div className="flex items-center gap-2 text-workflow-text-muted text-sm">
          <Loader2 className="h-4 w-4 animate-spin" />
          Loading pods...
        </div>
      ) : (
        <select
          value={pod}
          onChange={(e) => onPodChange(e.target.value)}
          className={selectClass}
        >
          <option value="all">All Pods</option>
          {pods.map((p) => (
            <option key={p} value={p}>{p}</option>
          ))}
        </select>
      )}
    </div>
  );
}

// Output Variables Component - shows output variables only
function OutputVariables({ nodeType }: { nodeType: string }) {
  const [isOpen, setIsOpen] = useState(false);
  const outputs = NODE_OUTPUT_VARIABLES[nodeType] || [];

  if (outputs.length === 0) {
    return null;
  }

  return (
    <div className="mt-2">
      <button
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center gap-1.5 text-xs text-workflow-text-muted hover:text-workflow-text transition-colors w-full"
      >
        <span className="font-mono uppercase tracking-wider">Output Variables ({outputs.length})</span>
        {isOpen ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
      </button>

      {isOpen && (
        <div className="mt-2 space-y-1.5">
          {outputs.map((output) => (
            <>
              <div key={output.name} className="flex items-start gap-2">
                <code className="text-xs font-mono text-primary bg-primary/10 px-1.5 py-0.5 rounded shrink-0">
                  {output.name}
                </code>
                <span className="text-[10px] font-mono text-workflow-text-subtle uppercase min-w-0 mt-2">
                  {output.type}
                </span>
              </div>
              <p className="text-[10px] text-workflow-text-muted">
                {output.description}
              </p>
            </>
          ))}

          <div className="pt-2 border-t border-workflow-border/50 mt-2">
            <p className="text-[10px] text-workflow-text-subtle font-mono">
              Reference: <code className="text-primary">{"{{nodeId.variableName}}"}</code>
            </p>
          </div>
        </div>
      )}
    </div>
  );
}

// SmartRouter Branch Editor Component
interface BranchItem {
  id: string;
  label: string;
  description: string;
}

// Generate a branch ID from label: lowercase, replace spaces/special chars with underscores
function generateBranchId(label: string): string {
  return label
    .toLowerCase()
    .replace(/[^a-z0-9一-鿿]+/g, "_")
    .replace(/^_|_$/g, "")
    .slice(0, 32);
}

const PRESET_TEMPLATES: { name: string; branches: Omit<BranchItem, "id">[] }[] = [
  {
    name: "SRE Incident Triage",
    branches: [
      { label: "downstream", description: "Downstream service errors: timeouts, 5xx, connection refused" },
      { label: "infrastructure", description: "Infrastructure issues: CPU/Memory/Disk saturation, node failures" },
      { label: "network", description: "Network issues: DNS failures, packet loss, high latency" },
    ],
  },
  {
    name: "Application Error",
    branches: [
      { label: "app_error", description: "Application-level errors: exceptions, crash loops, OOMKilled" },
      { label: "middleware", description: "Middleware issues: database, cache, message queue problems" },
      { label: "config", description: "Configuration issues: bad deployments, env vars, feature flags" },
    ],
  },
];

function SmartRouterBranchEditor({ branches, onChange }: {
  branches: string;
  onChange: (val: string) => void;
}) {
  const [items, setItems] = useState<BranchItem[]>(() => {
    try {
      const parsed = JSON.parse(branches || "[]");
      return Array.isArray(parsed) ? parsed : [];
    } catch {
      return [];
    }
  });

  const [newLabel, setNewLabel] = useState("");
  const [newDesc, setNewDesc] = useState("");

  const syncToParent = (updated: BranchItem[]) => {
    setItems(updated);
    onChange(JSON.stringify(updated));
  };

  const addBranch = () => {
    const label = newLabel.trim();
    if (!label) return;
    const id = generateBranchId(label);
    // Avoid ID collision
    let finalId = id;
    let suffix = 1;
    while (items.some(b => b.id === finalId)) {
      finalId = `${id}_${suffix++}`;
    }
    syncToParent([...items, { id: finalId, label, description: newDesc.trim() }]);
    setNewLabel("");
    setNewDesc("");
  };

  const removeBranch = (idx: number) => {
    syncToParent(items.filter((_, i) => i !== idx));
  };

  const updateBranch = (idx: number, field: keyof BranchItem, value: string) => {
    const updated = [...items];
    const oldItem = updated[idx];
    const newItem = { ...oldItem, [field]: value };
    // Auto-regenerate id when label changes
    if (field === "label" && value.trim()) {
      const newId = generateBranchId(value);
      // Only auto-update id if the old id was auto-generated from the old label
      if (generateBranchId(oldItem.label) === oldItem.id) {
        let finalId = newId;
        let suffix = 1;
        while (items.some((b, i) => i !== idx && b.id === finalId)) {
          finalId = `${newId}_${suffix++}`;
        }
        newItem.id = finalId;
      }
    }
    updated[idx] = newItem;
    syncToParent(updated);
  };

  const applyTemplate = (template: typeof PRESET_TEMPLATES[number]) => {
    const newBranches: BranchItem[] = template.branches.map(b => ({
      id: generateBranchId(b.label),
      label: b.label,
      description: b.description,
    }));
    syncToParent([...items, ...newBranches]);
  };

  const inputClass = "w-full bg-workflow-surface/50 border border-workflow-border rounded-lg px-3 py-2 text-sm text-workflow-text font-mono focus:outline-none focus:ring-1 focus:ring-primary/50 transition-colors duration-200";

  return (
    <div className="space-y-2">
      {/* Preset templates */}
      {items.length === 0 && (
        <div className="space-y-1.5">
          <p className="text-[10px] text-workflow-text-subtle font-mono">Quick start — choose a template:</p>
          <div className="flex flex-wrap gap-1.5">
            {PRESET_TEMPLATES.map((t) => (
              <button
                key={t.name}
                type="button"
                onClick={() => applyTemplate(t)}
                className="text-[10px] font-mono px-2 py-1 rounded-lg border border-amber-500/30 text-amber-600 dark:text-amber-400 hover:bg-amber-500/10 transition-colors"
              >
                {t.name}
              </button>
            ))}
          </div>
        </div>
      )}

      {/* Existing branches */}
      {items.map((branch, idx) => (
        <div key={branch.id} className="bg-workflow-surface/30 rounded-lg border border-workflow-border p-2 space-y-1.5">
          <div className="flex items-center justify-between">
            <code className="text-[10px] font-mono text-amber-600 dark:text-amber-400 bg-amber-500/10 px-1.5 py-0.5 rounded">
              {branch.id}
            </code>
            <button
              type="button"
              onClick={() => removeBranch(idx)}
              className="text-workflow-text-muted hover:text-red-400 transition-colors text-xs"
            >
              Remove
            </button>
          </div>
          <input
            type="text"
            value={branch.label}
            onChange={(e) => updateBranch(idx, "label", e.target.value)}
            placeholder="Branch name (e.g. downstream)"
            className={`${inputClass} text-xs py-1`}
          />
          <input
            type="text"
            value={branch.description}
            onChange={(e) => updateBranch(idx, "description", e.target.value)}
            placeholder="Description (helps LLM choose, e.g. connection timeout to downstream services)"
            className={`${inputClass} text-xs py-1`}
          />
        </div>
      ))}

      {/* Add new branch */}
      <div className="border border-dashed border-workflow-border rounded-lg p-2 space-y-1.5">
        <p className="text-[10px] text-workflow-text-subtle font-mono">Add Branch</p>
        <input
          type="text"
          value={newLabel}
          onChange={(e) => setNewLabel(e.target.value)}
          placeholder="Branch name (e.g. downstream)"
          className={`${inputClass} text-xs py-1`}
        />
        <input
          type="text"
          value={newDesc}
          onChange={(e) => setNewDesc(e.target.value)}
          placeholder="Description (helps LLM choose, e.g. connection timeout to downstream services)"
          className={`${inputClass} text-xs py-1`}
        />
        <button
          type="button"
          onClick={addBranch}
          disabled={!newLabel.trim()}
          className="w-full text-xs font-mono px-2 py-1 rounded-lg border border-amber-500/30 text-amber-600 dark:text-amber-400 hover:bg-amber-500/10 transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
        >
          + Add Branch
        </button>
      </div>

      {items.length > 0 && (
        <p className="text-[10px] text-workflow-text-subtle font-mono">
          Drag from the branch ports on the right side of the node to connect downstream nodes.
          Only the selected branch will execute; others will be skipped.
        </p>
      )}
    </div>
  );
}

// SkillSelector — multi-select skills for Agent node
function SkillSelector({ selectedIds, onChange, skills }: {
  selectedIds: string[];
  onChange: (ids: string[]) => void;
  skills: Skill[];
}) {
  const toggleSkill = (id: string) => {
    if (selectedIds.includes(id)) {
      onChange(selectedIds.filter((sid) => sid !== id));
    } else {
      onChange([...selectedIds, id]);
    }
  };

  if (skills.length === 0) {
    return (
      <div className="text-xs text-workflow-text-muted font-mono py-2">
        No skills available. Create skills in the Skills page first.
      </div>
    );
  }

  return (
    <div className="space-y-1.5 max-h-32 overflow-y-auto">
      {skills.map((skill) => (
        <label
          key={skill.id}
          className="flex items-center gap-2 px-2 py-1.5 rounded-lg border border-workflow-border hover:bg-workflow-surface-hover cursor-pointer transition-colors duration-200"
        >
          <input
            type="checkbox"
            checked={selectedIds.includes(skill.id)}
            onChange={() => toggleSkill(skill.id)}
            className="rounded border-workflow-border text-primary focus:ring-primary/30"
          />
          <div className="min-w-0 flex-1">
            <div className="text-xs font-mono text-workflow-text truncate">{skill.name}</div>
            {skill.description && (
              <div className="text-[10px] text-workflow-text-muted truncate">{skill.description}</div>
            )}
          </div>
        </label>
      ))}
    </div>
  );
}

// MCPToolSelector — multi-select MCP tools for Agent node
function MCPToolSelector({
  selectedTools,
  onChange,
  servers,
  serverTools,
  loadingTools,
  expandedServer,
  onExpandServer,
}: {
  selectedTools: string[];
  onChange: (tools: string[]) => void;
  servers: MCPServer[];
  serverTools: Record<string, MCPServerTool[]>;
  loadingTools: Record<string, boolean>;
  expandedServer: string | null;
  onExpandServer: (id: string | null) => void;
}) {
  const toggleTool = (serverId: string, toolName: string) => {
    const key = `${serverId}::${toolName}`;
    if (selectedTools.includes(key)) {
      onChange(selectedTools.filter((t) => t !== key));
    } else {
      onChange([...selectedTools, key]);
    }
  };

  const toggleServer = (serverId: string) => {
    onExpandServer(expandedServer === serverId ? null : serverId);
  };

  if (servers.length === 0) {
    return (
      <div className="text-xs text-workflow-text-muted font-mono py-2">
        No MCP servers available. Add MCP servers in the MCP Tools page first.
      </div>
    );
  }

  return (
    <div className="space-y-1.5 max-h-40 overflow-y-auto">
      {servers.map((server) => {
        const tools = serverTools[server.id];
        const isLoading = loadingTools[server.id];
        const isExpanded = expandedServer === server.id;
        const selectedInServer = selectedTools.filter((t) => t.startsWith(`${server.id}::`)).length;

        return (
          <div key={server.id} className="border border-workflow-border rounded-lg overflow-hidden">
            <button
              type="button"
              onClick={() => toggleServer(server.id)}
              className="w-full flex items-center gap-2 px-2.5 py-1.5 hover:bg-workflow-surface-hover transition-colors text-left"
            >
              <span className={`transition-transform duration-150 ${isExpanded ? "rotate-90" : ""}`}>
                <ChevronRight className="w-3 h-3 text-workflow-text-muted" />
              </span>
              <span className="text-xs font-mono text-workflow-text truncate flex-1">{server.name}</span>
              {selectedInServer > 0 && (
                <span className="text-[10px] bg-blue-500/20 text-blue-400 px-1.5 py-0.5 rounded-full shrink-0">
                  {selectedInServer}
                </span>
              )}
            </button>

            {isExpanded && (
              <div className="border-t border-workflow-border bg-workflow-bg/30">
                {isLoading ? (
                  <div className="flex items-center gap-2 text-xs text-workflow-text-muted py-2 px-3">
                    <Loader2 className="w-3 h-3 animate-spin" />
                    Loading tools...
                  </div>
                ) : !tools || tools.length === 0 ? (
                  <div className="text-xs text-workflow-text-muted py-2 px-3">No tools available</div>
                ) : (
                  <div className="py-1">
                    {tools.map((tool) => {
                      const key = `${server.id}::${tool.name}`;
                      return (
                        <label
                          key={tool.name}
                          className="flex items-start gap-2 px-3 py-1.5 hover:bg-workflow-surface-hover cursor-pointer transition-colors ml-4"
                        >
                          <input
                            type="checkbox"
                            checked={selectedTools.includes(key)}
                            onChange={() => toggleTool(server.id, tool.name)}
                            className="mt-0.5 rounded border-workflow-border text-primary focus:ring-primary/30"
                          />
                          <div className="min-w-0 flex-1">
                            <div className="text-xs font-mono text-workflow-text">{tool.name}</div>
                            {tool.description && (
                              <div className="text-[10px] text-workflow-text-muted line-clamp-2">{tool.description}</div>
                            )}
                          </div>
                        </label>
                      );
                    })}
                  </div>
                )}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
