"use client";

import { memo } from "react";
import type { NodeProps } from "@xyflow/react";
import { BaseNode } from "../base-node";
import type { AgentNodeData } from "@/lib/workflow-types";

function AgentNodeComponent({ data, selected, ...props }: NodeProps) {
  const nodeData = data as AgentNodeData;
  const skillCount = nodeData.skillIds?.length || 0;
  const toolCount = nodeData.selectedTools?.length || 0;

  return (
    <BaseNode
      nodeType="Agent"
      data={nodeData}
      selected={selected}
      runStatus={nodeData.runStatus}
      {...props}
    >
      <div className="space-y-1.5">
        <div className="text-xs text-workflow-text-muted font-mono truncate">
          {nodeData.model || "Agent"}
        </div>
        <div className="text-xs text-workflow-text-subtle font-mono truncate">
          ReAct Agent{skillCount > 0 ? ` · ${skillCount} skill${skillCount > 1 ? "s" : ""}` : ""}{toolCount > 0 ? ` · ${toolCount} tool${toolCount > 1 ? "s" : ""}` : ""}
        </div>
      </div>
    </BaseNode>
  );
}

export const AgentNode = memo(AgentNodeComponent);
