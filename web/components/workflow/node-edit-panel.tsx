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
import { mcpApi } from "@/lib/api/mcp";
import { integrationApi } from "@/lib/api/integration";

// Node output variables definition
export const NODE_OUTPUT_VARIABLES: Record<string, { name: string; type: string; description: string }[]> = {
  LLM: [
    { name: "output", type: "string", description: "LLM generated text response" },
    { name: "latency", type: "float", description: "API response time in seconds" },
    { name: "usage.inputTokens", type: "int", description: "Tokens used in prompt" },
    { name: "usage.outputTokens", type: "int", description: "Tokens in completion" },
    { name: "usage.totalTokens", type: "int", description: "Total tokens used" },
  ],
  Prometheus: [
    { name: "series", type: "array", description: "Time series data" },
    { name: "raw_data", type: "object", description: "Raw API response" },
  ],
  Jaeger: [
    { name: "traces", type: "array", description: "Array of trace objects" },
    { name: "traceCount", type: "int", description: "Number of traces" },
    { name: "raw_response", type: "object", description: "Raw API response" },
  ],
  Elasticsearch: [
    { name: "data", type: "object", description: "Elasticsearch response data" },
  ],
  Loki: [
    { name: "result_type", type: "string", description: "Result type: streams/logs/etc" },
    { name: "series", type: "array", description: "Log series" },
    { name: "data", type: "object", description: "Raw API response" },
  ],
  ClickHouse: [
    { name: "data", type: "array/object", description: "Query result data" },
  ],
  SSH: [
    { name: "output", type: "string", description: "Command output" },
  ],
  Kubernetes: [
    { name: "pods", type: "array", description: "Pod list (when resourceType=status, pod=all)" },
    { name: "pod", type: "object", description: "Single pod status (when resourceType=status, pod=specific)" },
    { name: "events", type: "array", description: "Event list (when resourceType=event)" },
    { name: "count", type: "int", description: "Number of pods or events returned" },
    { name: "namespace", type: "string", description: "Namespace queried" },
  ],
  Condition: [
    { name: "result", type: "bool", description: "Condition check result (true/false)" },
    { name: "input", type: "string", description: "Original input value" },
    { name: "condition", type: "string", description: "Variable used in check" },
    { name: "operator", type: "string", description: "Operator used" },
    { name: "value", type: "string", description: "Value compared against" },
  ],
  Start: [
    { name: "text", type: "string", description: "Text content from start node" },
  ],
  GoScript: [
    { name: "output", type: "object", description: "User-defined output variable from script" },
    { name: "error", type: "string", description: "Error message (if execution failed)" },
    { name: "success", type: "bool", description: "Whether script executed successfully" },
  ],
  HTTP: [
    { name: "statusCode", type: "int", description: "HTTP response status code" },
    { name: "responseBody", type: "string", description: "Response body from the endpoint" },
    { name: "responseHeaders", type: "object", description: "Response headers" },
    { name: "requestBody", type: "string", description: "The resolved request body sent" },
    { name: "url", type: "string", description: "Target URL" },
    { name: "method", type: "string", description: "HTTP method used" },
  ],
  MCP: [
    { name: "output", type: "string", description: "MCP tool execution result" },
    { name: "error", type: "string", description: "Error message if execution failed" },
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
  const [showOutputVars, setShowOutputVars] = useState(false);
  const [beautifyingSystem, setBeautifyingSystem] = useState(false);
  const [beautifyingPrompt, setBeautifyingPrompt] = useState(false);
  const [activePicker, setActivePicker] = useState<string | null>(null);
  const [mcpInputSchema, setMcpInputSchema] = useState<Record<string, unknown> | null>(null);
  const [loadingMcpSchema, setLoadingMcpSchema] = useState(false);
  const [activeTextPicker, setActiveTextPicker] = useState<string | null>(null);
  const [goscriptModalOpen, setGoscriptModalOpen] = useState(false);
  const [goscriptModalCode, setGoscriptModalCode] = useState("");

  const systemPromptRef = useRef<HTMLTextAreaElement>(null);
  const promptRef = useRef<HTMLTextAreaElement>(null);
  const goscriptRef = useRef<HTMLTextAreaElement>(null);
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const goscriptEditorRef = useRef<any>(null);
  const promqlRef = useRef<HTMLTextAreaElement>(null);
  const esQueryRef = useRef<HTMLTextAreaElement>(null);
  const lokiQueryRef = useRef<HTMLTextAreaElement>(null);
  const chQueryRef = useRef<HTMLTextAreaElement>(null);
  const webhookBodyRef = useRef<HTMLTextAreaElement>(null);
  const conditionRef = useRef<HTMLInputElement>(null);
  const sshCommandRef = useRef<HTMLTextAreaElement>(null);
  const sshHostRef = useRef<HTMLInputElement>(null);
  const sshPortRef = useRef<HTMLInputElement>(null);

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
    };
    const textareaRef = refMap[pickerKey];
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
      sshCommand: "command",
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

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className={labelClass}>Service</label>
              <input
                type="text"
                value={(node.data as any).service || ""}
                onChange={(e) => handleChange("service", e.target.value)}
                placeholder="service-name"
                className={inputClass}
              />
            </div>
            <div>
              <label className={labelClass}>Operation</label>
              <input
                type="text"
                value={(node.data as any).operation || ""}
                onChange={(e) => handleChange("operation", e.target.value)}
                placeholder="operation-name"
                className={inputClass}
              />
            </div>
          </div>
          <div>
            <label className={labelClass}>Trace ID</label>
            <input
              type="text"
              value={(node.data as any).traceId || ""}
              onChange={(e) => handleChange("traceId", e.target.value)}
              placeholder="specific-trace-id"
              className={inputClass}
            />
          </div>
        </>
      )}

      {/* Elasticsearch Node */}
      {node.type === "Elasticsearch" && (
        <>

          <div>
            <label className={labelClass}>Index</label>
            <input
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
            placeholder="Enter your text content..."
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
