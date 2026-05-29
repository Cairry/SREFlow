"use client";

import React, { useState } from "react";
import { API_BASE_URL } from "@/lib/api/workflow";
import { ModalDialog } from "./ui/dialog";
import { Copy, Check, Terminal } from "lucide-react";

interface ApiCallDialogProps {
  isOpen: boolean;
  onClose: () => void;
  workflowId: string | null;
  workflowName?: string;
}

function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    await navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <button
      type="button"
      onClick={handleCopy}
      title="Copy"
      className="p-1 hover:bg-workflow-surface-hover rounded transition-colors duration-200 text-workflow-text-muted hover:text-workflow-text"
    >
      {copied ? (
        <Check className="w-3.5 h-3.5 text-emerald-400" />
      ) : (
        <Copy className="w-3.5 h-3.5" />
      )}
    </button>
  );
}

function CodeBlock({ children }: { children: React.ReactNode }) {
  return (
    <div className="bg-black/50 rounded-lg p-3 font-mono text-xs overflow-x-auto">
      <pre className="text-workflow-text">{children}</pre>
    </div>
  );
}

// 获取执行状态 curl 示例
function getStatusCurl(execId: string, baseUrl: string, wsId: string) {
  return `curl -X GET "${baseUrl}/api/workflow/executions/${execId}/status" \\
  -H "X-Workspace-Id: ${wsId}"`;
}

// SSE stream curl 示例
function getSseCurl(execId: string, baseUrl: string, wsId: string) {
  return `curl -N "${baseUrl}/api/workflow/executions/${execId}/stream?workspaceId=${wsId}"`;
}

// 获取执行结果 curl
function getResultCurl(execId: string, baseUrl: string, wsId: string) {
  return `curl -X GET "${baseUrl}/api/workflow/executions/${execId}" \\
  -H "X-Workspace-Id: ${wsId}"`;
}

export function ApiCallDialog({
  isOpen,
  onClose,
  workflowId,
  workflowName,
}: ApiCallDialogProps) {
  if (!isOpen) return null;

  const baseUrl = API_BASE_URL;
  const wsId = "YOUR_WORKSPACE_ID";
  const execId = "exec-xxxxxxxxxxxx"; // 执行 ID，从 execute 接口获取

  const executeCurl = workflowId
    ? `curl -X POST "${baseUrl}/api/workflow/${workflowId}/execute?mode=text" \\
  -H "Content-Type: application/json" \\
  -H "X-Workspace-Id: ${wsId}" \\
  -d '{"input": "your prompt text here"}'`
    : `# Please save the workflow first to get the API endpoint`;

  const executeCurlJson = workflowId
    ? `curl -X POST "${baseUrl}/api/workflow/${workflowId}/execute?mode=json" \\
  -H "Content-Type: application/json" \\
  -H "X-Workspace-Id: ${wsId}" \\
  -d '{"key1": "value1", "key2": "value2"}'`
    : "";

  const statusCurl = getStatusCurl(execId, baseUrl, wsId);
  const resultCurl = getResultCurl(execId, baseUrl, wsId);
  const sseCurl = getSseCurl(execId, baseUrl, wsId);

  return (
    <ModalDialog
      isOpen={isOpen}
      onClose={onClose}
      title="API 调用"
      description={`通过 API 外部调用 Workflow: ${workflowName || "Untitled"}`}
      width="xl"
      maxHeight="max-h-[80vh]"
    >
      <div className="space-y-6">
        {/* Intro */}
        <div className="text-xs text-workflow-text-muted leading-relaxed">
          <p>
            工作流保存后，可以通过以下 API 从外部系统触发执行。执行是异步的，
            立即返回 <code className="text-violet-400">executionId</code>，通过 SSE 或轮询获取实时状态。
          </p>
        </div>

        {/* ── 1. Execute ─────────────────────────────── */}
        <div className="space-y-3">
          <div className="flex items-center gap-2">
            <span className="inline-flex items-center px-2 py-0.5 rounded bg-amber-500/20 text-amber-400 font-mono text-xs">
              POST
            </span>
            <code className="text-xs text-workflow-text-muted font-mono">
              {baseUrl}/api/workflow/&#123;workflowId&#125;/execute<code className="text-violet-400">?mode=text|json</code>
            </code>
          </div>
          <div className="border border-workflow-border rounded-lg overflow-hidden">
            <table className="w-full text-xs font-mono">
              <thead>
                <tr className="border-b border-workflow-border bg-black/30">
                  <th className="text-left px-3 py-2 text-workflow-text-muted font-medium">参数</th>
                  <th className="text-left px-3 py-2 text-workflow-text-muted font-medium">位置</th>
                  <th className="text-left px-3 py-2 text-workflow-text-muted font-medium">类型</th>
                  <th className="text-left px-3 py-2 text-workflow-text-muted font-medium">说明</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-workflow-border/50">
                <tr>
                  <td className="px-3 py-2 text-violet-400">workflowId</td>
                  <td className="px-3 py-2 text-workflow-text-muted">URL path</td>
                  <td className="px-3 py-2 text-emerald-400">string</td>
                  <td className="px-3 py-2 text-workflow-text-muted">
                    {workflowId ? (
                      <span className="text-workflow-text">{workflowId}</span>
                    ) : (
                      <span className="text-amber-400">save first</span>
                    )}
                  </td>
                </tr>
                <tr>
                  <td className="px-3 py-2 text-violet-400">X-Workspace-Id</td>
                  <td className="px-3 py-2 text-workflow-text-muted">header</td>
                  <td className="px-3 py-2 text-emerald-400">string</td>
                  <td className="px-3 py-2 text-workflow-text-muted">工作空间 ID (必填)</td>
                </tr>
                <tr>
                  <td className="px-3 py-2 text-violet-400">input</td>
                  <td className="px-3 py-2 text-workflow-text-muted">body</td>
                  <td className="px-3 py-2 text-emerald-400">string</td>
                  <td className="px-3 py-2 text-workflow-text-muted">
                    Start 节点 text 参数 (mode=text 时使用，如 Start 节点已配置则可省略)
                  </td>
                </tr>
                <tr>
                  <td className="px-3 py-2 text-violet-400">mode</td>
                  <td className="px-3 py-2 text-workflow-text-muted">query</td>
                  <td className="px-3 py-2 text-emerald-400">string</td>
                  <td className="px-3 py-2 text-workflow-text-muted">
                    执行模式：<code className="text-violet-400">text</code>（默认）从 body 的 input 字段读取；<code className="text-violet-400">json</code> 将整个 body 作为 Start 节点输出
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
          <div className="space-y-1.5">
            <div className="flex items-center gap-2">
              <Terminal className="w-3.5 h-3.5 text-workflow-text-muted" />
              <span className="text-xs font-mono font-medium text-workflow-text">cURL (mode=text)</span>
              <div className="ml-auto">
                <CopyButton text={executeCurl} />
              </div>
            </div>
            <CodeBlock>
              <pre className="whitespace-pre-wrap break-all">{executeCurl}</pre>
            </CodeBlock>
          </div>
          {executeCurlJson && (
            <div className="space-y-1.5">
              <div className="flex items-center gap-2">
                <Terminal className="w-3.5 h-3.5 text-workflow-text-muted" />
                <span className="text-xs font-mono font-medium text-workflow-text">cURL (mode=json)</span>
                <span className="text-xs text-workflow-text-muted">— 整个 body 作为 Start 节点输出，下游通过 start.key 引用</span>
                <div className="ml-auto">
                  <CopyButton text={executeCurlJson} />
                </div>
              </div>
              <CodeBlock>
                <pre className="whitespace-pre-wrap break-all">{executeCurlJson}</pre>
              </CodeBlock>
            </div>
          )}
        </div>

        {/* ── 2. Status ──────────────────────────────── */}
        <div className="space-y-3">
          <div className="flex items-center gap-2">
            <span className="inline-flex items-center px-2 py-0.5 rounded bg-emerald-500/20 text-emerald-400 font-mono text-xs">
              GET
            </span>
            <code className="text-xs text-workflow-text-muted font-mono">
              {baseUrl}/api/workflow/executions/&#123;executionId&#125;/status
            </code>
          </div>
          <div className="border border-workflow-border rounded-lg p-3 font-mono text-xs bg-black/30 space-y-1 text-workflow-text-muted">
            <p>&#123;</p>
            <p className="pl-4">executionId, workflowId, workflowName,</p>
            <p className="pl-4">status: "pending" | "running" | "completed" | "failed",</p>
            <p className="pl-4">runningNodes: string[],</p>
            <p className="pl-4">completedNodes: string[],</p>
            <p className="pl-4">failedNodes: string[],</p>
            <p className="pl-4">pendingNodes: string[],</p>
            <p className="pl-4">totalNodes: number,</p>
            <p className="pl-4">startedAt, finishedAt</p>
            <p>&#125;</p>
          </div>
          <div className="space-y-1.5">
            <div className="flex items-center gap-2">
              <Terminal className="w-3.5 h-3.5 text-workflow-text-muted" />
              <span className="text-xs font-mono font-medium text-workflow-text">cURL</span>
              <div className="ml-auto">
                <CopyButton text={statusCurl} />
              </div>
            </div>
            <CodeBlock>
              <pre className="whitespace-pre-wrap break-all">{statusCurl}</pre>
            </CodeBlock>
          </div>
        </div>

        {/* ── 3. Result ──────────────────────────────── */}
        <div className="space-y-3">
          <div className="flex items-center gap-2">
            <span className="inline-flex items-center px-2 py-0.5 rounded bg-emerald-500/20 text-emerald-400 font-mono text-xs">
              GET
            </span>
            <code className="text-xs text-workflow-text-muted font-mono">
              {baseUrl}/api/workflow/executions/&#123;executionId&#125;
            </code>
          </div>
          <div className="border border-workflow-border rounded-lg p-3 font-mono text-xs bg-black/30 space-y-1 text-workflow-text-muted">
            <p>&#123;</p>
            <p className="pl-4">id, workflowId, workflowName, status,</p>
            <p className="pl-4">nodeResults: &#123; [nodeId]: &#123; outputs, error &#125; &#125;,</p>
            <p className="pl-4">variables: &#123; [key]: value &#125;,</p>
            <p className="pl-4">restructuredOutput: &#123; [nodeId]: output &#125;,</p>
            <p className="pl-4">startedAt, finishedAt</p>
            <p>&#125;</p>
          </div>
          <div className="space-y-1.5">
            <div className="flex items-center gap-2">
              <Terminal className="w-3.5 h-3.5 text-workflow-text-muted" />
              <span className="text-xs font-mono font-medium text-workflow-text">cURL</span>
              <div className="ml-auto">
                <CopyButton text={resultCurl} />
              </div>
            </div>
            <CodeBlock>
              <pre className="whitespace-pre-wrap break-all">{resultCurl}</pre>
            </CodeBlock>
          </div>
        </div>

        {/* ── 4. SSE Stream ──────────────────────────── */}
        <div className="space-y-3">
          <div className="flex items-center gap-2">
            <span className="inline-flex items-center px-2 py-0.5 rounded bg-blue-500/20 text-blue-400 font-mono text-xs">
              GET
            </span>
            <code className="text-xs text-workflow-text-muted font-mono">
              {baseUrl}/api/workflow/executions/&#123;executionId&#125;/stream
            </code>
          </div>
          <div className="border border-workflow-border rounded-lg overflow-hidden">
            <table className="w-full text-xs font-mono">
              <thead>
                <tr className="border-b border-workflow-border bg-black/30">
                  <th className="text-left px-3 py-2 text-workflow-text-muted font-medium">Event</th>
                  <th className="text-left px-3 py-2 text-workflow-text-muted font-medium">说明</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-workflow-border/50">
                <tr><td className="px-3 py-2 text-violet-400">test_connection</td><td className="px-3 py-2 text-workflow-text-muted">连接测试</td></tr>
                <tr><td className="px-3 py-2 text-violet-400">heartbeat</td><td className="px-3 py-2 text-workflow-text-muted">心跳保活 (每 10s)</td></tr>
                <tr><td className="px-3 py-2 text-violet-400">running</td><td className="px-3 py-2 text-workflow-text-muted">工作流开始运行</td></tr>
                <tr><td className="px-3 py-2 text-violet-400">node_started</td><td className="px-3 py-2 text-workflow-text-muted">节点开始执行</td></tr>
                <tr><td className="px-3 py-2 text-violet-400">node_completed</td><td className="px-3 py-2 text-workflow-text-muted">节点执行完成</td></tr>
                <tr><td className="px-3 py-2 text-violet-400">node_failed</td><td className="px-3 py-2 text-workflow-text-muted">节点执行失败</td></tr>
                <tr><td className="px-3 py-2 text-violet-400">workflow_completed</td><td className="px-3 py-2 text-workflow-text-muted">工作流执行完成</td></tr>
                <tr><td className="px-3 py-2 text-violet-400">workflow_failed</td><td className="px-3 py-2 text-workflow-text-muted">工作流执行失败</td></tr>
                <tr><td className="px-3 py-2 text-violet-400">workflow_cancelled</td><td className="px-3 py-2 text-workflow-text-muted">工作流已取消</td></tr>
              </tbody>
            </table>
          </div>
          <div className="space-y-1.5">
            <div className="flex items-center gap-2">
              <Terminal className="w-3.5 h-3.5 text-workflow-text-muted" />
              <span className="text-xs font-mono font-medium text-workflow-text">cURL</span>
              <div className="ml-auto">
                <CopyButton text={sseCurl} />
              </div>
            </div>
            <CodeBlock>
              <pre className="whitespace-pre-wrap break-all">{sseCurl}</pre>
            </CodeBlock>
          </div>
        </div>

        {/* Notes */}
        <div className="rounded-lg border border-workflow-border/50 bg-black/30 p-3 space-y-1.5">
          <p className="text-xs font-mono text-workflow-text font-medium">注意事项</p>
          <ul className="text-xs font-mono text-workflow-text-muted space-y-1 list-disc list-inside">
            <li>执行是异步的，立即返回 executionId，不会阻塞 HTTP 连接</li>
            <li>推荐使用 SSE stream 获取实时节点状态变化，断线后自动降级为轮询</li>
            <li>input 参数：如果 Start 节点未配置 text，则必须传递；已配置则可省略（仅 mode=text）</li>
            <li>mode=json 时，整个 body 将作为 Start 节点输出，下游节点通过 start.fieldName 引用各字段</li>
            <li>Workflow 必须在执行前保存，否则 API 会返回 404</li>
          </ul>
        </div>
      </div>
    </ModalDialog>
  );
}
