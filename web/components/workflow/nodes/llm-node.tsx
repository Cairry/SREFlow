"use client";

import { memo } from "react";
import type { NodeProps } from "@xyflow/react";
import { BaseNode } from "../base-node";
import type { LLMNodeData } from "@/lib/workflow-types";

function LLMNodeComponent({ data, selected, ...props }: NodeProps) {
  const nodeData = data as LLMNodeData;

  return (
    <BaseNode
      nodeType="LLM"
      data={nodeData}
      selected={selected}
      runStatus={nodeData.runStatus}
      {...props}
    >
      <div className="space-y-1.5">
        <div className="text-xs text-workflow-text-muted font-mono truncate">
          {nodeData.model || "LLM"}
        </div>
        <div className="text-xs text-workflow-text-subtle font-mono truncate">
          Ai analysis node
        </div>
      </div>
    </BaseNode>
  );
}

export const LLMNode = memo(LLMNodeComponent);
