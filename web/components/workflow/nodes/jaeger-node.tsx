"use client";

import { memo } from "react";
import type { NodeProps } from "@xyflow/react";
import { BaseNode } from "../base-node";
import type { JaegerNodeData } from "@/lib/workflow-types";

function JaegerNodeComponent({ data, selected, ...props }: NodeProps) {
  const nodeData = data as JaegerNodeData;
  
  return (
    <BaseNode 
      nodeType="Jaeger" 
      data={nodeData} 
      selected={selected} 
      runStatus={nodeData.runStatus}
      {...props}
    >
      <div className="space-y-1.5">
        <div className="text-xs text-workflow-text-muted font-mono truncate">
          Jaeger
        </div>
        <div className="text-xs text-workflow-text-subtle font-mono">
          {nodeData.service || 'service'} / {nodeData.operation || 'operation'}
        </div>
      </div>
    </BaseNode>
  );
}

export const JaegerNode = memo(JaegerNodeComponent);
