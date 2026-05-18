"use client";

import { memo } from "react";
import type { NodeProps } from "@xyflow/react";
import { BaseNode } from "../base-node";
import type { MCPNodeData } from "@/lib/workflow-types";

function MCPNodeComponent({ data, selected, ...props }: NodeProps) {
  const nodeData = data as MCPNodeData;

  return (
    <BaseNode
      nodeType="MCP"
      data={nodeData}
      selected={selected}
      runStatus={nodeData.runStatus}
      {...props}
    >
      <div className="space-y-1.5">
        <div className="text-xs text-workflow-text-muted font-mono truncate">
          {nodeData.serverName || "No server"}
        </div>
        <div className="text-xs text-workflow-text-subtle font-mono truncate">
          {nodeData.toolName || "No tool selected"}
        </div>
      </div>
    </BaseNode>
  );
}

export const MCPNode = memo(MCPNodeComponent);
