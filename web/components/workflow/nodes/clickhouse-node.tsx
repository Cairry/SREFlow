"use client";

import { memo } from "react";
import type { NodeProps } from "@xyflow/react";
import { BaseNode } from "../base-node";
import type { ClickHouseNodeData } from "@/lib/workflow-types";

function ClickHouseNodeComponent({ data, selected, ...props }: NodeProps) {
  const nodeData = data as ClickHouseNodeData;
  
  return (
    <BaseNode 
      nodeType="ClickHouse" 
      data={nodeData} 
      selected={selected} 
      runStatus={nodeData.runStatus}
      {...props}
    >
      <div className="space-y-1.5">
        <div className="text-xs text-workflow-text-muted font-mono truncate">
          ClickHouse
        </div>
        <div className="text-xs text-workflow-text-subtle font-mono">
          {nodeData.database || 'default'}
        </div>
      </div>
    </BaseNode>
  );
}

export const ClickHouseNode = memo(ClickHouseNodeComponent);
