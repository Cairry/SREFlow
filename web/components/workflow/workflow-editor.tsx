"use client";

import React, { useState, useEffect, useCallback, useMemo } from "react";

import {
  ReactFlowProvider,
  useNodesState,
  useEdgesState,
  addEdge,
  type Connection,
  type Edge,
  type OnNodesChange,
  type OnEdgesChange,
} from "@xyflow/react";

import type { WorkflowNodeType, WorkflowNode, NodeRunStatus, ExecutionSSEEvent } from "@/lib/workflow-types";
import { generateNodeId } from "@/lib/node-id-generator";
import { adaptSnapshotToNodeStatuses } from "@/lib/state-adapter";
import { WorkflowCanvas, defaultNodeData } from "./workflow-canvas";
import { WorkflowToolbar } from "./workflow-toolbar";
import { OutputPanel } from "./output-panel";
import { useToast } from "@/hooks/use-toast";
import { X } from "lucide-react";
import { LoadWorkflowDialog } from "./load-workflow-dialog";
import { NodeEditPanel } from "./node-edit-panel";
import { RunHistoryDialog, RunHistoryContent, RunDetails } from "./run-history-dialog";
import { WorkflowVersionHistoryDialog } from "./workflow-version-history-dialog";
import { ApiCallDialog } from "./api-call-dialog";
import { workflowApi, API_BASE_URL, WorkflowVersion } from "@/lib/api/workflow";

// Empty workflow template with default Start node
const getInitialNodes = (): WorkflowNode[] => [
  {
    id: "Start",
    type: "Start",
    position: { x: 100, y: 200 },
    data: { label: "Start", text: "" },
  },
];
const initialEdges: Edge[] = [];

// 从节点 outputs 中提取展示用的值
// 如果 outputs 包含多个 key（即 map 类型），序列化整个对象；否则取单一值
function extractOutputValue(outputs: Record<string, unknown>): string | null {
  const keys = Object.keys(outputs);
  if (keys.length === 0) return null;
  if (keys.length === 1) return (outputs[keys[0]] as string | null) ?? null;
  // 多 key 的 map 类型，序列化整个对象
  return JSON.stringify(outputs, null, 2);
}

function WorkflowEditorInner({ workflowId: urlWorkflowId }: { workflowId?: Promise<{ id: string }> }) {
  // Check for template data in sessionStorage on mount
  const getInitialData = () => {
    if (typeof window !== 'undefined') {
      const templateData = sessionStorage.getItem('selectedTemplate');
      if (templateData) {
        const template = JSON.parse(templateData);
        sessionStorage.removeItem('selectedTemplate'); // Clear after use
        return {
          name: template.name,
          description: template.description || "",
          nodes: template.nodes,
          edges: template.edges,
        };
      }
    }
    return {
      name: "Untitled Workflow",
      description: "",
      nodes: getInitialNodes(),
      edges: initialEdges,
    };
  };

  const initialData = getInitialData();
  const [nodes, setNodes, onNodesChange] = useNodesState(initialData.nodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialData.edges);
  const [workflowId, setWorkflowId] = useState<string | null>(null);
  const [workflowName, setWorkflowName] = useState(initialData.name);
  const [workflowDescription, setWorkflowDescription] = useState(initialData.description || "");
  const [hasChanges, setHasChanges] = useState(false);
  const [initialDescription, setInitialDescription] = useState(initialData.description || "");

  // Track description changes
  useEffect(() => {
    if (workflowDescription !== initialDescription) {
      setHasChanges(true);
    }
  }, [workflowDescription, initialDescription]);
  const [isExecuting, setIsExecuting] = useState(false);
  const [currentExecutionId, setCurrentExecutionId] = useState<string | null>(null);
  const { toast } = useToast();

  // 组件卸载时清理 SSE 连接
  useEffect(() => {
    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
    };
  }, []);

  // 从URL参数获取workflowId
  useEffect(() => {
    const loadWorkflowFromUrl = async () => {
      try {
        if (urlWorkflowId) {
          const params = await urlWorkflowId;
          const id = params.id;
          if (id) {
            console.log('[WorkflowEditor] Loading workflow from URL:', id);
            // 加载工作流数据
            await handleLoad(id);
          }
        }
      } catch (error) {
        console.error("Failed to load workflow from URL:", error);
        toast({
          title: "Error",
          description: "Failed to load workflow from URL",
          variant: "destructive",
        });
      }
    };
    
    loadWorkflowFromUrl();
  }, [urlWorkflowId, toast]);

    const [isSaving, setIsSaving] = useState(false);
  const [execution, setExecution] = useState<WorkflowExecution | null>(null);
  const [showOutput, setShowOutput] = useState(false);
  const [showLoadDialog, setShowLoadDialog] = useState(false);
  const [showHistoryDialog, setShowHistoryDialog] = useState(false);
  const [selectedNode, setSelectedNode] = useState<WorkflowNode | null>(null);
  const [showTemplatesDialog, setShowTemplatesDialog] = useState(false); // Declare showTemplatesDialog
  const [nodeStatuses, setNodeStatuses] = useState<Map<string, NodeRunStatus>>(new Map()); // 节点状态跟踪
  const [isStopping, setIsStopping] = useState(false);
  // SSE EventSource 引用，用于清理
  const eventSourceRef = React.useRef<EventSource | null>(null);
  // SSE 收集的节点输出
  const nodeResultsRef = React.useRef<ExecutionResult[]>([]);

  // 侧边栏状态：'progress' | 'history' | null
  const [sidebarMode, setSidebarMode] = useState<'progress' | 'history' | null>(null);

  // API 调用弹窗状态
  const [showApiCallDialog, setShowApiCallDialog] = useState(false);
  const [showVersionHistoryDialog, setShowVersionHistoryDialog] = useState(false);
  const [versionRefreshTrigger, setVersionRefreshTrigger] = useState(0);

  // Run History 详情状态
  const [selectedRun, setSelectedRun] = useState<any>(null);

  const onConnect = useCallback(
    (params: Connection) => {
      setEdges((eds) => addEdge({ ...params, type: "default", animated: false }, eds));
      setHasChanges(true);
    },
    [setEdges]
  );

  // Wrap onNodesChange to prevent start node deletion
  const handleNodesChange: OnNodesChange<WorkflowNode> = useCallback(
    (changes) => {
      // Filter out delete changes for start nodes
      const filteredChanges = changes.filter(change => {
        if (change.type === "remove") {
          const nodeToDelete = nodes.find(n => n.id === change.id);
          if (nodeToDelete?.type === "Start") {
            toast({
              title: "Cannot delete Start node",
              description: "A workflow must have exactly one Start node",
              variant: "destructive",
            });
            return false;
          }
        }
        return true;
      });

      if (filteredChanges.length > 0) {
        onNodesChange(filteredChanges);
        // Check if any actual changes were made
        if (filteredChanges.some(c => c.type !== "remove" || !nodes.find(n => n.id === c.id && n.type === "Start"))) {
          setHasChanges(true);
        }
      }
    },
    [nodes, onNodesChange, toast]
  );

  
  const handleAddNode = useCallback(
    (nodeType: WorkflowNodeType, mcpData?: { serverId: string; serverName: string; toolName: string; description: string }) => {
      // Prevent adding multiple start nodes
      if (nodeType === "Start" && nodes.some(node => node.type === "Start")) {
        toast({
          title: "Cannot add Start node",
          description: "A workflow can only have one Start node",
          variant: "destructive",
        });
        return;
      }

      // Smart positioning for new nodes - find optimal spot
      const horizontalSpacing = 340;
      const verticalSpacing = 100;

      // Find the rightmost position
      let maxX = 0;
      let maxY = 0;

      nodes.forEach(node => {
        if (node.position.x > maxX) maxX = node.position.x;
        if (node.position.y > maxY) maxY = node.position.y;
      });

      // Calculate new position with some spacing
      const newPosition = {
        x: maxX + horizontalSpacing,
        y: Math.max(0, maxY - verticalSpacing)
      };

      const newNode: WorkflowNode = {
        id: generateNodeId(nodeType),
        type: nodeType,
        position: newPosition,
        data: {
          ...defaultNodeData[nodeType],
          ...(mcpData || {}),
        },
      };

      setNodes((nds) => [...nds, newNode]);
      setHasChanges(true);
    },
    [setNodes, nodes, toast]
  );

  const handleNodesUpdate = useCallback(
    (updatedNodes: WorkflowNode[]) => {
      setNodes(updatedNodes);
      setHasChanges(true);
    },
    [setNodes]
  );

// ─── Auto Layout helpers ────────────────────────────────────────────────────

const NODE_W = 280;
const NODE_H = 120;
const COL_W = NODE_W + 160;   // 列间距
const ROW_H = NODE_H + 100;   // 同列节点行间距
const PAD_X = 80;
const PAD_Y = 80;

/** BFS 分层：同层节点竖向排列，不同层横向排列（左 → 右） */
function layoutNodes(nodes: WorkflowNode[], edges: Edge[]): Map<string, { x: number; y: number }> {
  const incoming = new Map<string, string[]>();
  const outgoing = new Map<string, string[]>();
  nodes.forEach(n => { incoming.set(n.id, []); outgoing.set(n.id, []); });
  edges.forEach(e => {
    incoming.get(e.target)?.push(e.source);
    outgoing.get(e.source)?.push(e.target);
  });

  // 层级：同批入队的为同一层（竖向并行）
  const levels: string[][] = [];
  const visited = new Set<string>();
  let queue = nodes.filter(n => (incoming.get(n.id)?.length ?? 0) === 0).map(n => n.id);

  while (queue.length) {
    const next: string[] = [];
    const level: string[] = [];
    queue.forEach(id => {
      if (visited.has(id)) return;
      visited.add(id);
      level.push(id);
      outgoing.get(id)?.forEach(childId => {
        if ((incoming.get(childId) ?? []).every(p => visited.has(p))) next.push(childId);
      });
    });
    levels.push(level);
    queue = next;
  }

  // 孤立节点
  nodes.forEach(n => { if (!visited.has(n.id)) levels.push([n.id]); });

  // 按层分配坐标
  const positions = new Map<string, { x: number; y: number }>();
  levels.forEach((level, col) => {
    const colX = PAD_X + col * COL_W;
    level.forEach((id, row) => {
      positions.set(id, { x: colX, y: PAD_Y + row * ROW_H });
    });
  });

  return positions;
}

// ─── ────────────────────────────────────────────────────────────────────────

  const handleAutoLayout = useCallback(() => {
    if (nodes.length === 0) return;
    const positions = layoutNodes(nodes, edges);
    setNodes(ns => ns.map(n => ({ ...n, position: positions.get(n.id) ?? n.position })));
    setHasChanges(true);
  }, [nodes, edges, setNodes]);

  // 通过 SSE 实时接收节点状态更新（主路径，失败时降级到轮询）
  // 注意：SSE named events 必须用 addEventListener，onmessage 只接收无名事件
  const connectSSE = useCallback((executionId: string) => {
    console.log('[SSE] Connecting for execution:', executionId);
    setCurrentExecutionId(executionId);
    nodeResultsRef.current = [];

    const workspaceId = localStorage.getItem("flowforge-current-workspace-id");
    const sseUrl = `${API_BASE_URL}/api/workflow/executions/${executionId}/stream${workspaceId ? `?workspaceId=${workspaceId}` : ""}`;
    const eventSource = new EventSource(sseUrl);
    eventSourceRef.current = eventSource;

    eventSource.onopen = () => {};

    // 辅助：更新节点状态为 running
    const handleNodeStarted = (e: MessageEvent) => {
      try {
        const eventData = JSON.parse(e.data);
        if (eventData.node?.id) {
          setNodeStatuses(prev => {
            const next = new Map(prev);
            next.set(eventData.node.id, 'running');
            return next;
          });
        }
      } catch (err) {
        console.error('[SSE] node_started parse error:', err);
      }
    };

    // 辅助：更新节点状态为 completed，收集输出
    const handleNodeCompleted = (e: MessageEvent) => {
      try {
        const eventData = JSON.parse(e.data);
        const node = nodes.find(n => n.id === eventData.node?.id);
        if (eventData.node?.id) {
          const outputs = eventData.node.outputs || {};
          const outputValue = extractOutputValue(outputs);
          nodeResultsRef.current.push({
            nodeId: eventData.node.id,
            nodeType: toWorkflowNodeType(node?.type || "unknown"),
            output: outputValue as string | null,
            timestamp: new Date(),
          });
          setNodeStatuses(prev => {
            const next = new Map(prev);
            next.set(eventData.node.id, 'completed');
            return next;
          });
        }
      } catch (err) {
        console.error('[SSE] node_completed parse error:', err);
      }
    };

    // 辅助：更新节点状态为 failed
    const handleNodeFailed = (e: MessageEvent) => {
      try {
        const eventData = JSON.parse(e.data);
        if (eventData.node?.id) {
          const node = nodes.find(n => n.id === eventData.node?.id);
          nodeResultsRef.current.push({
            nodeId: eventData.node?.id,
            nodeType: toWorkflowNodeType(node?.type || "unknown"),
            output: null,
            error: eventData.node?.error,
            timestamp: new Date(),
          });
          setNodeStatuses(prev => {
            const next = new Map(prev);
            next.set(eventData.node?.id, 'failed');
            return next;
          });
        }
      } catch (err) {
        console.error('[SSE] node_failed parse error:', err);
      }
    };

    // 辅助：更新节点状态为 skipped（孤立节点）
    const handleNodeSkipped = (e: MessageEvent) => {
      try {
        const eventData = JSON.parse(e.data);
        if (eventData.node?.id) {
          setNodeStatuses(prev => {
            const next = new Map(prev);
            next.set(eventData.node.id, 'skipped');
            return next;
          });
        }
      } catch (err) {
        console.error('[SSE] node_skipped parse error:', err);
      }
    };

    // 初始化所有节点为 pending，同时应用后端快照中的 node_states（如 skipped）
    const handleRunning = (e: MessageEvent) => {
      try {
        const eventData = JSON.parse(e.data);
        const statuses = new Map<string, NodeRunStatus>();
        nodes.forEach(n => statuses.set(n.id, 'pending'));

        // 如果后端发送了 node_states 快照（包含 skipped 等状态），优先使用
        if (eventData.data?.node_states) {
          Object.entries(eventData.data.node_states).forEach(([nodeId, state]) => {
            statuses.set(nodeId, mapBackendToFrontend(state as string));
          });
        }

        setNodeStatuses(statuses);
      } catch (err) {
        console.error('[SSE] running parse error:', err);
      }
    };

    // 工作流结束，收尾
    const handleWorkflowDone = (e: MessageEvent) => {
      try {
        const eventData = JSON.parse(e.data);
        const evt = eventData.event;
        const isCancelled = evt === "workflow_cancelled";
        const isCompleted = evt === "workflow_completed";
        const finalStatus = isCancelled ? 'cancelled' : isCompleted ? 'completed' : 'failed';
        console.log('[SSE] workflow done:', evt, eventData);
        console.log('[SSE] workflow done - node_states:', eventData.data?.node_states);

        if (eventData.data?.node_states) {
          // 正常完成/失败：从 node_states 快照刷新状态，始终保护 skipped 节点
          setNodeStatuses(prev => {
            const statusMap = new Map(prev); // 保留之前的状态（如 skipped）
            Object.entries(eventData.data.node_states).forEach(([nodeId, state]) => {
              // 始终保护 skipped 节点，不被后端快照覆盖
              const existing = statusMap.get(nodeId);
              if (existing === 'skipped') {
                return; // 跳过，不覆盖
              }
              statusMap.set(nodeId, mapBackendToFrontend(state as string));
            });
            // 补充未在快照中的节点
            nodes.forEach(n => {
              if (!statusMap.has(n.id)) {
                statusMap.set(n.id, finalStatus === 'completed' ? 'completed' : 'idle');
              }
            });
            return statusMap;
          });
        } else if (isCancelled) {
          // 取消：把当前所有 running/pending 节点标记为 cancelled
          setNodeStatuses(prev => {
            const next = new Map(prev);
            let changed = false;
            next.forEach((status, nodeId) => {
              if (status === 'running' || status === 'pending') {
                next.set(nodeId, 'cancelled');
                changed = true;
              }
            });
            // 未出现在 nodeStatuses 的节点也设为 cancelled
            nodes.forEach(n => {
              if (!next.has(n.id)) {
                next.set(n.id, 'cancelled');
                changed = true;
              }
            });
            return changed ? next : prev;
          });
        } else {
          // 没有快照且非取消：保护已有的 skipped 节点，把 running 刷新为 completed
          setNodeStatuses(prev => {
            const next = new Map(prev);
            let changed = false;
            next.forEach((status, nodeId) => {
              if (status === 'running') {
                next.set(nodeId, 'completed');
                changed = true;
              }
            });
            // 补充未在 map 中的节点
            nodes.forEach(n => {
              if (!next.has(n.id)) {
                next.set(n.id, finalStatus === 'completed' ? 'completed' : 'idle');
              }
            });
            return changed ? next : prev;
          });
        }

        let finalOutput = "";
        const outputNode = nodeResultsRef.current.find(r =>
          nodes.find(n => n.id === r.nodeId && n.type === "Output")
        );
        if (outputNode) finalOutput = outputNode.output || "";

        setExecution({
          id: executionId,
          workflowId: workflowId || "",
          status: finalStatus,
          results: nodeResultsRef.current,
          finalOutput,
          startedAt: new Date(),
          completedAt: new Date(),
        });
        setIsExecuting(false);
        setIsStopping(false);
        setCurrentExecutionId(null);

        if (isCancelled) {
          toast({ title: "Stopped", description: "Workflow execution was cancelled", variant: "destructive" });
        } else if (isCompleted) {
          toast({ title: "Completed", description: "Workflow finished successfully" });
        } else {
          toast({ title: "Failed", description: "Workflow execution failed", variant: "destructive" });
        }

        eventSource.close();
        eventSourceRef.current = null;
      } catch (err) {
        console.error('[SSE] workflow done parse error:', err);
      }
    };

    // 为每个命名事件注册专用的 listener
    eventSource.addEventListener('running', handleRunning);
    eventSource.addEventListener('node_started', handleNodeStarted);
    eventSource.addEventListener('node_completed', handleNodeCompleted);
    eventSource.addEventListener('node_failed', handleNodeFailed);
    eventSource.addEventListener('node_skipped', handleNodeSkipped);
    eventSource.addEventListener('workflow_completed', handleWorkflowDone);
    eventSource.addEventListener('workflow_failed', handleWorkflowDone);
    eventSource.addEventListener('workflow_cancelled', handleWorkflowDone);

    eventSource.onerror = () => {
      eventSource.close();
      eventSourceRef.current = null;
      fallbackPoll(executionId);
    };
  }, [nodes, workflowId, toast]);

  // 轮询兜底（SSE 失败时或备用）
  const fallbackPoll = useCallback(async (executionId: string) => {
    setCurrentExecutionId(executionId);
    let pollCount = 0;
    const maxPolls = 120;

    const poll = async () => {
      if (!eventSourceRef.current) {
        pollCount++;
        try {
          const statusData = await workflowApi.getExecutionStatus(executionId);
          console.log('[Poll] statusData:', { status: statusData.status, skippedNodes: statusData.skippedNodes, completedNodes: statusData.completedNodes });

          const newStatuses = adaptSnapshotToNodeStatuses(statusData, nodes.map(n => n.id));
          // 保护已标记为 skipped 的孤立节点，并参考后端 skippedNodes
          setNodeStatuses(prev => {
            const merged = new Map(newStatuses);
            prev.forEach((status, nodeId) => {
              if (status === 'skipped') {
                merged.set(nodeId, 'skipped');
              }
            });
            // 后端返回的 skippedNodes 也保护
            (statusData.skippedNodes || []).forEach((nodeId: string) => {
              merged.set(nodeId, 'skipped');
            });
            return merged;
          });

          const completedCount = (statusData.completedNodes?.length || 0) + (statusData.failedNodes?.length || 0);
          const totalNodes = statusData.totalNodes || nodes.length;
          const isAllNodesCompleted = completedCount >= totalNodes;
          const isWorkflowFinished = statusData.status === 'completed' || statusData.status === 'failed';

          if (isWorkflowFinished || isAllNodesCompleted) {
            setIsExecuting(false);
            setCurrentExecutionId(null);

            const execData = await workflowApi.getExecution(executionId);
            const nodeResults = Object.entries(execData.nodeResults || {}).map(([nodeId, nodeResult]: [string, any]) => {
              const node = nodes.find(n => n.id === nodeId);
              const outputs = nodeResult.outputs || {};
              const outputValue = extractOutputValue(outputs);
              return { nodeId, nodeType: toWorkflowNodeType(node?.type || "unknown"), output: outputValue, error: nodeResult.error, timestamp: new Date() };
            });
            setExecution({
              id: execData.id, workflowId: execData.workflowId, status: statusData.status,
              results: nodeResults, finalOutput: execData.variables?.finalOutput || "",
              startedAt: new Date(execData.startedAt * 1000),
              completedAt: execData.finishedAt ? new Date(execData.finishedAt * 1000) : new Date(),
            });
            const finalStatuses = new Map<string, NodeRunStatus>();
            nodes.forEach(node => {
              const nr = nodeResults.find(r => r.nodeId === node.id);
              // 如果后端返回了 skippedNodes，保留 skipped 状态
              const isSkipped = statusData.skippedNodes?.includes(node.id);
              finalStatuses.set(node.id, isSkipped ? 'skipped' : (nr?.error ? 'failed' : 'completed'));
            });
            console.log('[Poll] Final statuses:', Object.fromEntries(finalStatuses));
            setNodeStatuses(finalStatuses);
            toast({ title: statusData.status === 'completed' ? "Completed" : "Failed", variant: statusData.status === 'failed' ? "destructive" : "default" });
            return;
          }

          if (pollCount >= maxPolls) {
            setIsExecuting(false);
            setCurrentExecutionId(null);
            toast({ title: "Timeout", description: "Workflow execution timeout", variant: "destructive" });
            return;
          }

          setTimeout(poll, 500);
        } catch (err) {
          console.error('[Poll] Error:', err);
          setIsExecuting(false);
          setCurrentExecutionId(null);
        }
      }
    };

    poll();
  }, [nodes, toast]);

  // 工具函数：将后端状态字符串映射为前端 NodeRunStatus
  const mapBackendToFrontend = (backendStatus: string): NodeRunStatus => {
    switch (backendStatus) {
      case 'idle': return 'idle';
      case 'pending': case 'ready': return 'pending';
      case 'running': return 'running';
      case 'completed': return 'completed';
      case 'failed': return 'failed';
      case 'cancelled': return 'cancelled';
      case 'skipped': return 'skipped';
      default: return 'idle';
    }
  };

  const handleSave = async () => {
    setIsSaving(true);

    try {
      const hasId = !!workflowId;
      const requestBody = {
        name: workflowName,
        description: workflowDescription,
        nodes,
        edges,
      };

      let result: Awaited<ReturnType<typeof workflowApi.create>>;
      if (hasId && workflowId) {
        result = await workflowApi.update(workflowId, requestBody);
      } else {
        result = await workflowApi.create(requestBody);
        if (result.id) {
          setWorkflowId(result.id);
        }
      }

      setHasChanges(false);
      setInitialDescription(workflowDescription);
      toast({
        title: "Success",
        description: hasId ? "Workflow updated successfully" : "Workflow created successfully",
      });
    } catch (error) {
      console.error("Save error:", error);
      toast({
        title: "Error",
        description: error instanceof Error ? error.message : "Failed to save workflow",
        variant: "destructive",
      });
    } finally {
      setIsSaving(false);
    }
  };

  const handleStop = async () => {
    if (!currentExecutionId) return;
    setIsStopping(true);
    try {
      await workflowApi.cancelExecution(currentExecutionId);
    } catch (err) {
      console.error("[Stop] Error:", err);
      toast({
        title: "Stop Failed",
        description: err instanceof Error ? err.message : "Failed to stop workflow",
        variant: "destructive",
      });
      setIsStopping(false);
    }
  };

  const handleExecute = async () => {
    if (!workflowId) {
      toast({
        title: "Error",
        description: "Please save the workflow first",
        variant: "destructive",
      });
      return;
    }

    setIsExecuting(true);
    setSidebarMode('progress');
    setExecution(null);

    // 前端直接计算孤立节点（无入边且无出边），不依赖 SSE 时序
    const inDegree = new Map<string, number>();
    const outDegree = new Map<string, number>();
    nodes.forEach(n => { inDegree.set(n.id, 0); outDegree.set(n.id, 0); });
    edges.forEach(e => {
      inDegree.set(e.target!, (inDegree.get(e.target!) || 0) + 1);
      outDegree.set(e.source!, (outDegree.get(e.source!) || 0) + 1);
    });
    const isolatedNodeIds = nodes
      .filter(n => (inDegree.get(n.id) || 0) === 0 && (outDegree.get(n.id) || 0) === 0)
      .map(n => n.id);

    console.log('[Execute] All edges:', edges.map(e => ({ source: e.source, target: e.target })));
    console.log('[Execute] In/Out degrees:', { inDegree: Object.fromEntries(inDegree), outDegree: Object.fromEntries(outDegree) });
    console.log('[Execute] Isolated nodes:', isolatedNodeIds);

    const newStatuses = new Map<string, NodeRunStatus>();
    nodes.forEach(node => {
      const status = isolatedNodeIds.includes(node.id) ? 'skipped' : 'pending';
      newStatuses.set(node.id, status);
    });
    setNodeStatuses(newStatuses);

    try {
      if (hasChanges && workflowId) {
        await handleSave(); // Save before executing if there are changes
      }
      const executionData = await workflowApi.execute(workflowId);
      setExecution({
        id: executionData.executionId,
        workflowId: executionData.workflowId,
        status: executionData.status,
        results: [],
        startedAt: new Date(),
      });

      connectSSE(executionData.executionId);

      toast({
        title: "Execution Started",
        description: "Workflow execution has started",
      });
    } catch (error) {
      console.error("Execute error:", error);
      toast({
        title: "Error",
        description: error instanceof Error ? error.message : "Failed to execute workflow",
        variant: "destructive",
      });
      setIsExecuting(false);
      setSidebarMode(null);
    }
  };

  const handleLoad = async (id: string) => {
    try {
      const workflowData = await workflowApi.getById(id);
      console.log('[WorkflowEditor] Workflow data:', workflowData);

      // 验证数据完整性
      if (!workflowData || !workflowData.id) {
        console.error('[WorkflowEditor] Invalid workflow data:', workflowData);
        throw new Error("Invalid workflow data received");
      }

      // 转换节点数据格式以匹配前端期望
      const convertedNodes = (workflowData.nodes || []).map((node: any) => {
        let nodeType = node.type;
        let nodeData = { ...node.data };

        const convertedNode: WorkflowNode = {
          id: node.id,
          type: nodeType as WorkflowNodeType,
          position: node.position,
          measured: node.measured,
          data: {
            ...nodeData,
            runStatus: 'idle' as const
          }
        };

        return convertedNode;
      });

      const finalNodes = convertedNodes.length > 0 ? convertedNodes : getInitialNodes();

      setWorkflowId(workflowData.id);
      setWorkflowName(workflowData.name || "Untitled Workflow");
      setWorkflowDescription(workflowData.description || "");
      setInitialDescription(workflowData.description || "");
      setNodes(finalNodes);
      setEdges(workflowData.edges || []);
      setHasChanges(false);
      setSelectedNode(null);
      setNodeStatuses(new Map());

      console.log('[WorkflowEditor] Workflow loaded successfully');
    } catch (error) {
      console.error("[WorkflowEditor] Load error:", error);
      toast({
        title: "Error",
        description: error instanceof Error ? error.message : "Failed to load workflow",
        variant: "destructive",
      });
    }
  };

  const handleClear = () => {
    setNodes(getInitialNodes());
    setEdges([]);
    setHasChanges(true);
    setSelectedNode(null);
    setNodeStatuses(new Map()); // 清理节点状态
  };

  const handleSelectTemplate = (templateNodes: WorkflowNode[], templateEdges: Edge[], name: string, description?: string) => {
    setNodes(templateNodes);
    setEdges(templateEdges);
    setWorkflowName(name);
    setWorkflowDescription(description || "");
    setInitialDescription(description || "");
    setHasChanges(true);
    setSelectedNode(null);
    setNodeStatuses(new Map()); // 清理节点状态
  };

  const handleRestored = (restored: { versionNum: number; contentHash: string; nodes: any[]; edges: any[]; name: string; description: string; currentVersionNum: number }) => {
    const restoredNodes = (restored.nodes || []).map((node: any) => ({
      id: node.id,
      type: node.type,
      position: node.position,
      data: {
        ...node.data,
        runStatus: 'idle' as const
      }
    }));

    setNodes(restoredNodes);
    setEdges(restored.edges || []);
    setWorkflowName(restored.name || workflowName);
    setWorkflowDescription(restored.description || "");
    setInitialDescription(restored.description || "");
    setHasChanges(false); // 前往版本后恢复的数据本身就是已保存状态，无需再次保存
    setSelectedNode(null);
    setNodeStatuses(new Map());

    toast({
      title: "版本切换成功",
      description: `已切换到 v${restored.currentVersionNum}，当前版本`,
    });
    setVersionRefreshTrigger(n => n + 1);
  };

  // 类型转换函数：将字符串转换为有效的 WorkflowNodeType
  const toWorkflowNodeType = (type: string): WorkflowNodeType => {
    const validTypes: WorkflowNodeType[] = [
      "LLM", "Condition", "SmartRouter", "Prometheus", "Jaeger",
      "Elasticsearch", "Loki", "ClickHouse", "Output",
      "Start", "GoScript"
    ];

    if (validTypes.includes(type as WorkflowNodeType)) {
      return type as WorkflowNodeType;
    }

    // 默认返回 Start 类型
    return "Start";
  };

  return (
    <div className="h-[calc(100vh-64px)] flex bg-workflow-bg">
        {/* Main Content Area */}
        <div className="flex-1 flex flex-col min-h-0">
          <WorkflowToolbar
            workflowName={workflowName}
            onSave={handleSave}
            isSaving={isSaving}
            onExecute={handleExecute}
            onStop={handleStop}
            isExecuting={isExecuting}
            isStopping={isStopping}
            onClear={handleClear}
            onNew={handleClear}
            onOpenHistory={() => setSidebarMode('history')}
            onOpenVersionHistory={() => setShowVersionHistoryDialog(true)}
            onSelectTemplate={handleSelectTemplate}
            onAddNode={handleAddNode}
            onAutoLayout={handleAutoLayout}
            onOpenApiCall={() => setShowApiCallDialog(true)}
            hasChanges={hasChanges}
            showOutput={showOutput}
          />

          <div className="flex-1 flex min-h-0">
          <div className="flex-1 flex flex-col min-h-0">
            <WorkflowCanvas
              nodes={nodes}
              edges={edges}
              onNodesChange={handleNodesChange}
              onEdgesChange={onEdgesChange}
              onConnect={onConnect}
              onNodesUpdate={handleNodesUpdate}
              onNodeClick={(node) => {
                setSelectedNode(node);
                setShowOutput(false);
              }}
              nodeStatuses={nodeStatuses}
              onStartNodeBlocked={() => {
                toast({
                  title: "Cannot add Start node",
                  description: "A workflow can only have one Start node",
                  variant: "destructive",
                });
              }}
            />
          </div>

          {selectedNode && !showOutput && (
            <div className="w-80 border-workflow-border shrink-0 pl-3">
              <div className="h-full bg-workflow-node-bg rounded-l-xl border border-workflow-border/50 shadow-lg overflow-hidden">
                <NodeEditPanel
                  node={selectedNode}
                  nodes={nodes}
                  edges={edges}
                  onClose={() => setSelectedNode(null)}
                  onUpdate={(nodeId, data) => {
                    setNodes((nds) =>
                      nds.map((node) =>
                        node.id === nodeId ? { ...node, data: { ...node.data, ...data } } : node
                      )
                    );
                    setSelectedNode((prev) =>
                      prev && prev.id === nodeId
                        ? { ...prev, data: { ...prev.data, ...data } }
                        : prev
                    );
                    setHasChanges(true);
                  }}
                />
              </div>
            </div>
          )}

          {sidebarMode === 'progress' && (execution || isExecuting) && (
            <div className="w-80 border-workflow-border shrink-0 pl-3">
              <div className="h-full bg-workflow-node-bg rounded-l-xl border border-workflow-border/50 shadow-lg overflow-hidden">
                <OutputPanel
                  execution={execution}
                  isExecuting={isExecuting}
                  onClose={() => {
                    setSidebarMode(null);
                    setExecution(null);
                  }}
                  nodes={nodes}
                  nodeStatuses={nodeStatuses}
                />
              </div>
            </div>
          )}

          {sidebarMode === 'history' && (
            <div className="w-80 border-workflow-border shrink-0 pl-3">
              <div className="h-full bg-workflow-node-bg rounded-l-xl border border-workflow-border/50 shadow-lg overflow-hidden flex flex-col">
                <div className="flex items-center justify-between p-3 border-b border-workflow-border">
                  <h2 className="font-mono text-sm font-semibold text-workflow-text">Run History</h2>
                  <button
                    type="button"
                    onClick={() => setSidebarMode(null)}
                    className="p-1 hover:bg-workflow-surface-hover rounded-lg transition-colors duration-200 text-workflow-text-muted hover:text-workflow-text"
                  >
                    <X className="w-4 h-4" />
                  </button>
                </div>
                {selectedRun ? (
                  <div className="flex-1 overflow-auto">
                    <div className="p-3 border-b border-workflow-border">
                      <button
                        type="button"
                        onClick={() => setSelectedRun(null)}
                        className="flex items-center gap-2 text-xs font-mono text-workflow-text-muted hover:text-workflow-text transition-colors duration-200"
                      >
                        ← Back to History
                      </button>
                    </div>
                    <div className="p-3">
                      <RunDetails run={selectedRun} onClose={() => setSelectedRun(null)} />
                    </div>
                  </div>
                ) : (
                  <div className="flex-1 overflow-auto">
                    <RunHistoryContent
                      workflowId={workflowId || ''}
                      onSelectRun={setSelectedRun}
                    />
                  </div>
                )}
              </div>
            </div>
          )}
        </div>
        </div>

      <LoadWorkflowDialog
        isOpen={showLoadDialog}
        onLoad={handleLoad}
        onClose={() => setShowLoadDialog(false)}
      />

      <RunHistoryDialog
        isOpen={showHistoryDialog}
        workflowId={workflowId || ''}
        onClose={() => setShowHistoryDialog(false)}
        onSelectRun={() => {}}
      />

      <ApiCallDialog
        isOpen={showApiCallDialog}
        onClose={() => setShowApiCallDialog(false)}
        workflowId={workflowId}
        workflowName={workflowName}
      />

      <WorkflowVersionHistoryDialog
        isOpen={showVersionHistoryDialog}
        onClose={() => setShowVersionHistoryDialog(false)}
        workflowId={workflowId}
        refreshTrigger={versionRefreshTrigger}
        onRestored={handleRestored}
      />
    </div>
  );
}

export function WorkflowEditor({ workflowId }: { workflowId?: Promise<{ id: string }> }) {
  return (
    <ReactFlowProvider>
      <WorkflowEditorInner workflowId={workflowId} />
    </ReactFlowProvider>
  );
}
