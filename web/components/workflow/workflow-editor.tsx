"use client";

import React, { useState, useEffect, useCallback, useMemo, useRef } from "react";
import { useRouter } from "next/navigation";

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

import type { WorkflowNodeType, WorkflowNode, NodeRunStatus, ExecutionSSEEvent, WorkflowExecution, ExecutionResult } from "@/lib/workflow-types";
import type { WorkflowExecutionStatus } from "./execution-log-panel";
import { generateNodeId } from "@/lib/node-id-generator";
import { adaptSnapshotToNodeStatuses } from "@/lib/state-adapter";
import { WorkflowCanvas, defaultNodeData } from "./workflow-canvas";
import { WorkflowToolbar } from "./workflow-toolbar";
import { ExecutionLogPanel } from "./execution-log-panel";
import { useToast } from "@/hooks/use-toast";
import { downloadFile } from "@/lib/utils";
import { X } from "lucide-react";
import { LoadWorkflowDialog } from "./load-workflow-dialog";
import { NodeEditPanel } from "./node-edit-panel";
import { WorkflowVersionHistoryDialog } from "./workflow-version-history-dialog";
import { ApiCallDialog } from "./api-call-dialog";
import { AddNodeDialog } from "./add-node-dialog";
import { workflowApi, WorkflowVersion } from "@/lib/api/workflow";
import { connectSSEStream, SSEEvent } from "@/lib/sse";

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
  const router = useRouter();
  
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
  const [isLocked, setIsLocked] = useState(true); // Default to locked
  const [chatMode, setChatMode] = useState(false); // Chat mode toggle

  // Wrap onEdgesChange to respect lock
  const handleEdgesChange = useCallback(
    (changes: any) => {
      if (isLocked) return; // Prevent edge changes if locked
      onEdgesChange(changes);
      setHasChanges(true);
    },
    [onEdgesChange, isLocked]
  );

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
      if (sseControllerRef.current) {
        sseControllerRef.current.abort();
        sseControllerRef.current = null;
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
  const [nodeResults, setNodeResults] = useState<ExecutionResult[]>([]);
  const [showOutput, setShowOutput] = useState(false);
  const [showLoadDialog, setShowLoadDialog] = useState(false);
  const [selectedNode, setSelectedNode] = useState<WorkflowNode | null>(null);
  const [nodeStatuses, setNodeStatuses] = useState<Map<string, NodeRunStatus>>(new Map()); // 节点状态跟踪
  const [isStopping, setIsStopping] = useState(false);
  // SSE AbortController 引用，用于取消流式连接
  const sseControllerRef = useRef<AbortController | null>(null);
  // SSE 收集的节点输出
  const nodeResultsRef = useRef<ExecutionResult[]>([]);
  // Execution status for the log panel (only 4 states: running, completed, failed, cancelled)
  const [executionStatus, setExecutionStatus] = useState<WorkflowExecutionStatus | null>(null);

  // API 调用弹窗状态
  const [showApiCallDialog, setShowApiCallDialog] = useState(false);
  const [showVersionHistoryDialog, setShowVersionHistoryDialog] = useState(false);
  const [versionRefreshTrigger, setVersionRefreshTrigger] = useState(0);

  // Add Node 弹窗状态（右键 + 工具栏共享）
  const [showAddNode, setShowAddNode] = useState(false);
  const addNodePositionRef = useRef<{ x: number; y: number } | null>(null);

  const onConnect = useCallback(
    (params: Connection) => {
      if (isLocked) return; // Prevent connections if locked
      setEdges((eds) => addEdge({ ...params, type: "default", animated: false }, eds));
      setHasChanges(true);
    },
    [setEdges, isLocked]
  );

  // Wrap onNodesChange to prevent start node deletion and respect lock
  const handleNodesChange: OnNodesChange<WorkflowNode> = useCallback(
    (changes) => {
      // Prevent changes if workflow is locked
      if (isLocked) {
        return;
      }
      
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
    [nodes, onNodesChange, toast, isLocked]
  );

  
  const handleAddNode = useCallback(
    (nodeType: WorkflowNodeType, mcpData?: { serverId: string; serverName: string; toolName: string; description: string }) => {
      // Prevent adding nodes if workflow is locked
      if (isLocked) {
        toast({
          title: "Workflow is locked",
          description: "Unlock the workflow to add nodes",
          variant: "destructive",
        });
        return;
      }

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
    [setNodes, nodes, toast, isLocked]
  );

  const handleNodesUpdate = useCallback(
    (updatedNodes: WorkflowNode[]) => {
      if (isLocked) return; // Prevent updates if locked
      setNodes(updatedNodes);
      setHasChanges(true);
    },
    [setNodes, isLocked]
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
  const connectSSE = useCallback((executionId: string) => {
    console.log('[SSE] Connecting for execution:', executionId);
    setCurrentExecutionId(executionId);
    nodeResultsRef.current = [];

    // 辅助：更新节点状态为 running
    const handleNodeStarted = (event: SSEEvent) => {
      const nodeId = event.node?.id;
      if (nodeId) {
        setNodeStatuses(prev => {
          const next = new Map(prev);
          next.set(nodeId, 'running');
          return next;
        });
      }
    };

    // 辅助：更新节点状态为 completed，收集输出
    const handleNodeCompleted = (event: SSEEvent) => {
      const nodeId = event.node?.id;
      const nodeDef = nodes.find(n => n.id === nodeId);
      if (nodeId) {
        setNodeStatuses(prev => {
          const next = new Map(prev);
          next.set(nodeId, 'completed');
          return next;
        });

        // 去重：回放缓冲事件与实时通道可能有重叠
        if (nodeResultsRef.current.some(r => r.nodeId === nodeId)) return;

        const outputs = event.node?.outputs || {};
        const outputValue = extractOutputValue(outputs);
        nodeResultsRef.current.push({
          nodeId,
          nodeType: toWorkflowNodeType(nodeDef?.type || "unknown"),
          output: outputValue as string | null,
          outputs: outputs as Record<string, unknown>,
          timestamp: new Date(),
        });
        setNodeResults([...nodeResultsRef.current]);
      }
    };

    // 辅助：节点执行失败，标记 failed 状态并从 outputs 读取 error
    const handleNodeFailed = (event: SSEEvent) => {
      const nodeId = event.node?.id;
      const nodeDef = nodes.find(n => n.id === nodeId);
      if (nodeId) {
        setNodeStatuses(prev => {
          const next = new Map(prev);
          next.set(nodeId, 'failed');
          return next;
        });

        // 去重
        if (nodeResultsRef.current.some(r => r.nodeId === nodeId)) return;

        const outputs = event.node?.outputs || {};
        const error = (outputs as Record<string, unknown>).error as string
          || event.node?.error
          || event.data?.error;
        nodeResultsRef.current.push({
          nodeId,
          nodeType: toWorkflowNodeType(nodeDef?.type || "unknown"),
          output: error || null,
          outputs: outputs as Record<string, unknown>,
          error,
          timestamp: new Date(),
        });
        setNodeResults([...nodeResultsRef.current]);
        return; // 节点级 failed，不触发工作流结束
      }
    };

    // 辅助：节点被跳过，标记 skipped 状态
    const handleNodeSkipped = (event: SSEEvent) => {
      const nodeId = event.node?.id;
      if (nodeId) {
        setNodeStatuses(prev => {
          const next = new Map(prev);
          next.set(nodeId, 'skipped');
          return next;
        });
      }
    };

    // 初始化所有节点为 pending，同时应用后端快照中的 node_states（如 skipped）
    const handleRunning = (event: SSEEvent) => {
      const statuses = new Map<string, NodeRunStatus>();
      nodes.forEach(n => statuses.set(n.id, 'pending'));

      if (event.data?.node_states) {
        Object.entries(event.data.node_states).forEach(([nodeId, state]) => {
          statuses.set(nodeId, mapBackendToFrontend(state as string));
        });
      }

      setNodeStatuses(statuses);
    };

    // 工作流结束，收尾
    const handleWorkflowDone = (event: SSEEvent) => {
      const evt = event.event;
      const isCancelled = evt === "cancelled";
      const isCompleted = evt === "completed";
      const finalStatus = isCancelled ? 'cancelled' : isCompleted ? 'completed' : 'failed';
      console.log('[SSE] workflow done:', evt, event);
      console.log('[SSE] workflow done - node_states:', event.data?.node_states);

      if (event.data?.node_states) {
        setNodeStatuses(prev => {
          const statusMap = new Map(prev);
          Object.entries(event.data.node_states!).forEach(([nodeId, state]) => {
            const existing = statusMap.get(nodeId);
            if (existing === 'skipped') return;
            statusMap.set(nodeId, mapBackendToFrontend(state as string));
          });
          nodes.forEach(n => {
            if (!statusMap.has(n.id)) {
              statusMap.set(n.id, finalStatus === 'completed' ? 'completed' : 'idle');
            }
          });
          return statusMap;
        });
      } else if (isCancelled) {
        setNodeStatuses(prev => {
          const next = new Map(prev);
          let changed = false;
          next.forEach((status, nodeId) => {
            if (status === 'running' || status === 'pending') {
              next.set(nodeId, 'cancelled');
              changed = true;
            }
          });
          nodes.forEach(n => {
            if (!next.has(n.id)) {
              next.set(n.id, 'cancelled');
              changed = true;
            }
          });
          return changed ? next : prev;
        });
      } else {
        setNodeStatuses(prev => {
          const next = new Map(prev);
          let changed = false;
          next.forEach((status, nodeId) => {
            if (status === 'running') {
              // 工作流失败时，仍在运行的节点标记为 failed
              next.set(nodeId, finalStatus === 'failed' ? 'failed' : 'completed');
              changed = true;
            }
          });
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
      setExecutionStatus(finalStatus);

      if (isCancelled) {
        toast({ title: "Stopped", description: "Workflow execution was cancelled", variant: "destructive" });
      } else if (isCompleted) {
        toast({ title: "Completed", description: "Workflow finished successfully" });
        if (outputNode && outputNode.outputs) {
          const content = (outputNode.outputs.output as string) || outputNode.output || "";
          const filename = (outputNode.outputs.filename as string) || "output.md";
          if (content) {
            downloadFile(content, filename.replace(/\.md$/, ""), "md", false);
          }
        }
      }

      sseControllerRef.current = null;
    };

    sseControllerRef.current = connectSSEStream(
      executionId,
      {
        on: {
          started: handleRunning,
          running: handleRunning,
          // thinking = node starting, thinked/reply = node completed with outputs
          thinking: handleNodeStarted,
          thinked: handleNodeCompleted,
          reply: handleNodeCompleted,
          completed: handleWorkflowDone,
          failed: handleNodeFailed,
          skipped: handleNodeSkipped,
          cancelled: handleWorkflowDone,
        },
        onError: () => {
          sseControllerRef.current = null;
          fallbackPoll(executionId);
        },
      }
    );
  }, [nodes, workflowId, toast]);

  // 轮询兜底（SSE 失败时或备用）
  const fallbackPoll = useCallback(async (executionId: string) => {
    setCurrentExecutionId(executionId);
    let pollCount = 0;
    const maxPolls = 120;

    const poll = async () => {
      if (!sseControllerRef.current) {
        pollCount++;
        try {
          const statusData = await workflowApi.getExecutionStatus(executionId);
          console.log('[Poll] statusData:', statusData);
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
          const isWorkflowFinished = statusData.status === 'completed';

          if (isWorkflowFinished || isAllNodesCompleted) {
            setIsExecuting(false);
            setCurrentExecutionId(null);

            const execData = await workflowApi.getExecution(executionId);
            const nodeResults = Object.entries(execData.nodeResults || {}).map(([nodeId, nodeResult]: [string, any]) => {
              const node = nodes.find(n => n.id === nodeId);
              const outputs = nodeResult.outputs || {};
              const outputValue = extractOutputValue(outputs);
              return { nodeId, nodeType: toWorkflowNodeType(node?.type || "unknown"), output: outputValue, outputs: outputs as Record<string, unknown>, error: nodeResult.error, timestamp: new Date() };
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
    } finally {
      setIsExecuting(false);
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

    // Idempotent: Reset all execution state before starting new run
    setIsExecuting(true);
    setExecutionStatus('running');
    setExecution(null);
    setNodeResults([]);
    nodeResultsRef.current = [];

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
        status: "running",
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
      setExecutionStatus(null);
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
      "Elasticsearch", "Loki", "ClickHouse", "Output", "Reply",
      "Start", "GoScript"
    ];

    if (validTypes.includes(type as WorkflowNodeType)) {
      return type as WorkflowNodeType;
    }

    // 默认返回 Start 类型
    return "Start";
  };

  return (
    <div className="h-screen flex bg-workflow-bg">
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
            onOpenVersionHistory={() => setShowVersionHistoryDialog(true)}
            onSelectTemplate={handleSelectTemplate}
            onAddNode={handleAddNode}
            onAutoLayout={handleAutoLayout}
            onOpenApiCall={() => setShowApiCallDialog(true)}
            hasChanges={hasChanges}
            showOutput={showOutput}
            showAddNode={showAddNode}
            onToggleAddNode={(buttonRect) => {
              if (buttonRect) {
                addNodePositionRef.current = { x: buttonRect.left, y: buttonRect.bottom + 8 };
              } else {
                addNodePositionRef.current = null;
              }
              setShowAddNode(v => !v);
            }}
            isLocked={isLocked}
            onToggleLock={() => setIsLocked(!isLocked)}
            chatMode={chatMode}
            onToggleChatMode={() => {
              if (workflowId) {
                router.push(`/workflows/${workflowId}/chat`);
              } else {
                toast({
                  title: "Save workflow first",
                  description: "Please save the workflow before switching to chat mode",
                  variant: "destructive",
                });
              }
            }}
          />

          <div className="flex-1 flex min-h-0 flex-col">
            {/* Main content area with canvas and sidebar */}
            <div className="flex-1 flex min-h-0">
              <div className="flex-1 flex flex-col min-h-0">
                <WorkflowCanvas
                  nodes={nodes}
                  edges={edges}
                  onNodesChange={handleNodesChange}
                  onEdgesChange={handleEdgesChange}
                  onConnect={onConnect}
                  onNodesUpdate={handleNodesUpdate}
                  onNodeClick={(node) => {
                    setSelectedNode(node);
                    setShowOutput(false);
                  }}
                  nodeStatuses={nodeStatuses}
                  onContextMenu={(pos) => {
                    addNodePositionRef.current = pos;
                    setShowAddNode(true);
                  }}
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
                <div className="border-workflow-border shrink-0 pl-3" style={{ width: "25%" }}>
                  <div className="h-full bg-workflow-node-bg rounded-l-xl border border-workflow-border/50 shadow-lg overflow-hidden">
                    <NodeEditPanel
                      node={selectedNode}
                      nodes={nodes}
                      edges={edges}
                      onClose={() => setSelectedNode(null)}
                      onUpdate={(nodeId, data) => {
                        if (isLocked) return; // Prevent updates if locked
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
                      readOnly={isLocked}
                    />
                  </div>
                </div>
              )}
            </div>

            {/* Execution log panel at the bottom - always visible */}
            <ExecutionLogPanel
              workflowId={workflowId}
              isExecuting={isExecuting}
              executionId={currentExecutionId}
              executionStatus={executionStatus}
              nodeResults={nodeResults}
              nodes={nodes}
              nodeStatuses={nodeStatuses}
              onClear={() => {
                setNodeResults([]);
                nodeResultsRef.current = [];
                setNodeStatuses(new Map());
                setExecutionStatus(null);
              }}
              onStart={handleExecute}
              onStop={handleStop}
            />
          </div>
        </div>

      <LoadWorkflowDialog
        isOpen={showLoadDialog}
        onLoad={handleLoad}
        onClose={() => setShowLoadDialog(false)}
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

      <AddNodeDialog
        isOpen={showAddNode}
        onClose={() => setShowAddNode(false)}
        onAddNode={(nodeType, mcpData) => {
          handleAddNode(nodeType, mcpData);
          setShowAddNode(false);
        }}
        position={addNodePositionRef.current || undefined}
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
