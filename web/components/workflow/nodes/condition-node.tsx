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
        <div className="flex items-center gap-1.5">
          <span className="text-xs text-workflow-text-subtle font-mono px-1.5 py-0.5 rounded bg-workflow-surface/60">
            {nodeData.operator || 'contains'}
          </span>
          {nodeData.operator !== 'isEmpty' && nodeData.operator !== 'isNotEmpty' && (
            <span className="text-xs text-workflow-text-muted font-mono truncate">
              {nodeData.value || 'value'}
            </span>
          )}
        </div>
      </div>
    </BaseNode>
  );
}

export const ConditionNode = memo(ConditionNodeComponent);
