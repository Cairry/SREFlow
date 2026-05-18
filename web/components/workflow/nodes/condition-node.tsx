"use client";

import { memo } from "react";
import type { NodeProps } from "@xyflow/react";
import { BaseNode } from "../base-node";
import type { ConditionNodeData } from "@/lib/workflow-types";

function ConditionNodeComponent({ data, selected, ...props }: NodeProps) {
  const nodeData = data as ConditionNodeData;
  return (
    <BaseNode
      nodeType="Condition"
      data={nodeData}
      selected={selected}
      runStatus={nodeData.runStatus}
      hasOutput={false}
      hasTrueOutput={true}
      {...props}
    >
      <div className="space-y-1.5">
        <div className="text-xs text-workflow-text-muted font-mono truncate">
          {nodeData.condition || 'variable'}
        </div>
        <div className="text-xs text-workflow-text-subtle font-mono">
          {nodeData.operator || 'contains'}
        </div>
      </div>
    </BaseNode>
  );
}

export const ConditionNode = memo(ConditionNodeComponent);
