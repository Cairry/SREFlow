"use client";

import React from "react"

import { useCallback, useRef, useState, useEffect, useMemo } from "react";
import { useTheme } from "next-themes";
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  addEdge,
  type Connection,
  type Edge,
  type NodeTypes,
  BackgroundVariant,
  type OnEdgesChange,
  type OnNodesChange,
  type NodeMouseHandler,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";

import type { WorkflowNodeType, WorkflowNode, NodeRunStatus } from "@/lib/workflow-types";
import { generateNodeId } from "@/lib/node-id-generator";
import {
  LLMNode,
  ConditionNode,
  SmartRouterNode,
  PrometheusNode,
  JaegerNode,
  ElasticsearchNode,
  LokiNode,
  ClickHouseNode,
  SSHNode,
  KubernetesNode,
  OutputNode,
  StartNode,
  GoScriptNode,
  HTTPNode,
  MCPNode,
  AgentNode,
} from "./nodes";

const nodeTypes: NodeTypes = {
  LLM: LLMNode,
  Condition: ConditionNode,
  SmartRouter: SmartRouterNode,
  Prometheus: PrometheusNode,
  Jaeger: JaegerNode,
  Elasticsearch: ElasticsearchNode,
  Loki: LokiNode,
  ClickHouse: ClickHouseNode,
  SSH: SSHNode,
  Kubernetes: KubernetesNode,
  Output: OutputNode,
  Start: StartNode,
  GoScript: GoScriptNode,
  HTTP: HTTPNode,
  MCP: MCPNode,
  Agent: AgentNode,
};

export const defaultNodeData: Record<WorkflowNodeType, object> = {
  LLM: {
    label: "LLM",
    model: "",
    prompt: "",
  },
  Condition: {
    label: "Condition",
    condition: "",
    operator: "contains",
    value: "",
  },
  SmartRouter: {
    label: "SmartRouter",
    model: "",
    branches: "[]",
    input: "",
  },
  Prometheus: {
    label: "Prometheus",
    prometheusUrl: "http://localhost:9090",
    query: "up",
    timeout: 30,
  },
  Jaeger: {
    label: "Jaeger",
    jaegerUrl: "http://localhost:16686",
    service: "",
  },
  Elasticsearch: {
    label: "Elasticsearch",
    elasticsearchUrl: "http://localhost:9200",
    index: "log-*",
    query: "*",
  },
  Loki: {
    label: "Loki",
    lokiUrl: "http://localhost:3100",
    query: "{job=\"fluentd\"}",
  },
  ClickHouse: {
    label: "ClickHouse",
    clickhouseUrl: "http://localhost:8123",
    database: "default",
    query: "SELECT 1",
  },
  SSH: {
    label: "SSH",
    host: "",
    port: "22",
    command: "",
    timeout: 30,
  },
  Output: {
    label: "Output",
    outputType: "webhook",
    webhook: {
      url: "",
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: "{\n  \"data\": {{input}}\n}",
      timeout: 30,
    },
  },
  Start: {
    label: "Start",
    text: "",
  },
  GoScript: {
    label: "GoScript",
    code: "",
    timeout: 5000,
  },
  HTTP: {
    label: "HTTP",
    url: "",
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: "{\n  \"data\": {{input}}\n}",
    timeout: 30,
  },
  Kubernetes: {
    label: "Kubernetes",
    namespace: "",
    pod: "all",
    resourceType: "status",
  },
  MCP: {
    label: "MCP",
    serverId: "",
    serverName: "",
    toolName: "",
    description: "",
    arguments: {},
  },
  Agent: {
    label: "Agent",
    model: "",
    prompt: "",
    maxIterations: 10,
    skillIds: [],
  },
};

interface WorkflowCanvasProps {
  nodes: WorkflowNode[];
  edges: Edge[];
  onNodesChange: OnNodesChange<WorkflowNode>;
  onEdgesChange: OnEdgesChange<Edge>;
  onConnect: (connection: Connection) => void;
  onNodesUpdate: (nodes: WorkflowNode[]) => void;
  onEdgesUpdate: (edges: Edge[]) => void;
  onNodeClick?: (node: WorkflowNode) => void;
  onContextMenu?: (position: { x: number; y: number }) => void;
  nodeStatuses?: Map<string, NodeRunStatus>;
  onStartNodeBlocked?: () => void;
}

export function WorkflowCanvas({
  nodes,
  edges,
  onNodesChange,
  onEdgesChange,
  onConnect,
  onNodesUpdate,
  onNodeClick,
  onContextMenu,
  nodeStatuses,
  onStartNodeBlocked,
}: WorkflowCanvasProps) {
  const { resolvedTheme } = useTheme();
  const [mounted, setMounted] = useState(false);
  const reactFlowWrapper = useRef<HTMLDivElement>(null);
  const connectingNodeId = useRef<string | null>(null);

  useEffect(() => {
    setMounted(true);
  }, []);

  const isDark = mounted ? resolvedTheme === "dark" : true;

  // Update edge styles when theme changes without remounting
  const themedEdges = useMemo(() => {
    return edges.map((edge) => ({
      ...edge,
      style: {
        ...edge.style,
        strokeWidth: 2,
        stroke: isDark ? "#52525b" : "#a1a1aa",
      },
    }));
  }, [edges, isDark]);

  const onDragOver = useCallback((event: React.DragEvent) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = "move";
  }, []);

  const handlePaneContextMenu = useCallback((event: MouseEvent | React.MouseEvent) => {
    event.preventDefault();
    onContextMenu?.({ x: (event as MouseEvent).clientX, y: (event as MouseEvent).clientY });
  }, [onContextMenu]);

  const onDrop = useCallback(
    (event: React.DragEvent) => {
      event.preventDefault();

      const type = event.dataTransfer.getData("application/reactflow") as WorkflowNodeType;
      if (!type) return;

      // Prevent adding multiple start nodes
      if (type === "Start" && nodes.some(node => node.type === "Start")) {
        onStartNodeBlocked?.();
        return;
      }

      const bounds = reactFlowWrapper.current?.getBoundingClientRect();
      if (!bounds) return;

      const position = {
        x: event.clientX - bounds.left - 140,
        y: event.clientY - bounds.top - 20,
      };

      const nodeData = defaultNodeData[type];
      if (!nodeData) {
        console.error(`No default data found for node type: ${type}`);
        return;
      }

      const newNode: WorkflowNode = {
        id: generateNodeId(type),
        type,
        position,
        data: nodeData as WorkflowNode["data"],
      };

      onNodesUpdate([...nodes, newNode]);
    },
    [nodes, onNodesUpdate, onStartNodeBlocked]
  );

  const handleConnect = useCallback(
    (params: Connection) => {
      onConnect(params);
    },
    [onConnect]
  );

  // 为节点添加状态信息
  const nodesWithStatus = nodes.map(node => ({
    ...node,
    data: {
      ...node.data,
      runStatus: nodeStatuses?.get(node.id) || 'idle'
    }
  }));

  if (!mounted) {
    return (
      <div className="h-full w-full flex items-center justify-center bg-workflow-bg">
        <div className="text-workflow-text">Loading workflow...</div>
      </div>
    );
  }

  const handleNodeClick: NodeMouseHandler<WorkflowNode> = (event, node) => {
    if (onNodeClick) {
      onNodeClick(node as WorkflowNode);
    }
  };

  return (
    <div className="h-full w-full" ref={reactFlowWrapper}>
      <ReactFlow
        nodes={nodesWithStatus}
        edges={themedEdges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={handleConnect}
        onNodeClick={handleNodeClick}
        onPaneClick={() => {
          if (onNodeClick) {
            onNodeClick(null as any);
          }
        }}
        onPaneContextMenu={handlePaneContextMenu}
        nodeTypes={nodeTypes}
        fitView
        fitViewOptions={{
          padding: 0.2,
          maxZoom: 1.0,
          minZoom: 0.5,
        }}
        snapToGrid
        snapGrid={[16, 16]}
        defaultEdgeOptions={{
          type: "smoothstep",
          style: { strokeWidth: 2, stroke: isDark ? "#52525b" : "#a1a1aa" },
          animated: false,
        }}
        selectNodesOnDrag={false}
        panOnDrag={true}
        panOnScroll={false}
        className="bg-workflow-bg transition-colors duration-200"
      >
        <Background
          variant={BackgroundVariant.Dots}
          gap={20}
          size={1}
          color={isDark ? "#374151" : "#e5e7eb"}
        />
        <Controls showInteractive={false} />
        <MiniMap
          nodeColor={isDark ? "#3f3f46" : "#a1a1aa"}
          maskColor={isDark ? "rgba(0, 0, 0, 0.7)" : "rgba(255, 255, 255, 0.7)"}
        />
      </ReactFlow>
    </div>
  );
}
