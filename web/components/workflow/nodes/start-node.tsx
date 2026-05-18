"use client";

import { memo } from "react";
import type { NodeProps } from "@xyflow/react";
import { BaseNode } from "../base-node";
import type { StartNodeData } from "@/lib/workflow-types";

function StartNodeComponent({ data, selected, ...props }: NodeProps) {
  const nodeData = data as StartNodeData;
  return (
    <BaseNode 
      nodeType="Start" 
      data={nodeData} 
      selected={selected} 
      runStatus={nodeData.runStatus}
      hasInput={false}
      hasOutput={true}
      {...props}
    >
      <div className="space-y-1.5">
        <div className="text-xs text-workflow-text-muted font-mono">
          Start Node
        </div>
        <div className="text-xs text-workflow-text-subtle font-mono truncate">
          {nodeData.text || "Click to edit"}
        </div>
      </div>
    </BaseNode>
  );
}

export const StartNode = memo(StartNodeComponent);
