/**
 * 共享 SSE 流式连接工具
 * 基于 fetch + ReadableStream，支持 AbortSignal、自定义 header，
 * 为 workflow editor 和 chat 页面提供统一的 SSE 消费接口。
 */

import { API_BASE_URL } from "@/lib/api/workflow";

// ─── Types ────────────────────────────────────────────────────────────────────

export interface SSENode {
  id: string;
  type: string;
  title?: string;
  outputs?: Record<string, string>;
  error?: string;
  elapsed_time?: string;
}

export interface SSEEventData {
  id: string;
  workflow_id: string;
  workflow_run_id: string;
  status: string;
  error?: string;
  node_states?: Record<string, string>;
  total_nodes?: number;
  completed_nodes?: number;
  duration?: number;
}

export interface SSEEvent {
  event: string;
  id: string;
  task_id: string;
  workflow?: { id: string; name: string };
  node?: SSENode;
  data: SSEEventData;
}

export interface SSEHandler {
  /** 收到任意命名事件时触发（在 onEvent 之前调用） */
  onEvent?: (event: SSEEvent) => void;
  /** 按事件名分发的处理器，key = event name */
  on?: Record<string, (event: SSEEvent) => void>;
  /** 连接异常或流断开时触发，可在此启动轮询降级 */
  onError?: (error?: Error) => void;
  /** 流正常关闭（done = true）时触发 */
  onClose?: () => void;
}

// ─── SSE Parser ───────────────────────────────────────────────────────────────

/**
 * 从 SSE 文本块中解析事件。
 * 返回已完整解析的事件列表；未完成的尾部数据保留在 buffer 中（由调用方管理）。
 */
export function parseSSEEvents(text: string): { events: SSEEvent[]; remainder: string } {
  const events: SSEEvent[] = [];
  // SSE 事件以 \n\n 分隔
  const parts = text.split("\n\n");
  // 最后一段可能不完整，保留为 remainder
  const remainder = parts.pop() ?? "";

  for (const part of parts) {
    if (!part.trim() || part.startsWith(":")) continue; // 注释行（如 ": connected"）
    const event = parseOneEvent(part);
    if (event) events.push(event);
  }

  return { events, remainder };
}

function parseOneEvent(block: string): SSEEvent | null {
  let eventName = "message";
  let dataStr = "";

  for (const line of block.split("\n")) {
    if (line.startsWith("event:")) {
      eventName = line.slice(6).trim();
    } else if (line.startsWith("data:")) {
      dataStr += line.slice(5).trim();
    }
  }

  if (!dataStr) return null;

  try {
    const parsed = JSON.parse(dataStr);
    // 后端 StreamStyleEvent 结构：顶层有 event, node, data 等字段
    return {
      event: parsed.event ?? eventName,
      id: parsed.id ?? "",
      task_id: parsed.task_id ?? "",
      workflow: parsed.workflow,
      node: parsed.node,
      data: parsed.data ?? {},
    } as SSEEvent;
  } catch {
    return null;
  }
}

// ─── Connect ──────────────────────────────────────────────────────────────────

/**
 * 建立 SSE 流式连接，自动解析事件并分发给 handler。
 * 返回 AbortController，调用 .abort() 即可断开。
 *
 * @param executionId  执行 ID
 * @param handler      事件处理器
 * @param endpoint     SSE 端点路径，默认 /api/workflow/{id}/stream
 */
export function connectSSEStream(
  executionId: string,
  handler: SSEHandler,
  options?: {
    endpoint?: string;   // 自定义端点，如 `/api/chat/${id}/stream`
    signal?: AbortSignal;
  }
): AbortController {
  const controller = new AbortController();
  const signal = options?.signal ?? controller.signal;
  const endpoint = options?.endpoint ?? `/api/workflow/${executionId}/stream`;

  const workspaceId =
    typeof window !== "undefined"
      ? localStorage.getItem("flowforge-current-workspace-id")
      : null;

  const url = `${API_BASE_URL}${endpoint}${workspaceId ? `?workspaceId=${workspaceId}` : ""}`;

  (async () => {
    try {
      const response = await fetch(url, {
        headers: workspaceId ? { "X-Workspace-Id": workspaceId } : {},
        signal,
      });

      if (!response.ok) {
        handler.onError?.(new Error(`SSE HTTP ${response.status}`));
        return;
      }

      const reader = response.body?.getReader();
      if (!reader) {
        handler.onError?.(new Error("No response body"));
        return;
      }

      const decoder = new TextDecoder();
      let buffer = "";

      while (true) {
        const { done, value } = await reader.read();
        if (done) {
          handler.onClose?.();
          break;
        }

        buffer += decoder.decode(value, { stream: true });
        const { events, remainder } = parseSSEEvents(buffer);
        buffer = remainder;

        for (const event of events) {
          if (event.event === "heartbeat") continue;
          handler.onEvent?.(event);
          handler.on?.[event.event]?.(event);
        }
      }
    } catch (err) {
      if (err instanceof DOMException && err.name === "AbortError") {
        handler.onClose?.();
      } else {
        handler.onError?.(err instanceof Error ? err : new Error(String(err)));
      }
    }
  })();

  return controller;
}
