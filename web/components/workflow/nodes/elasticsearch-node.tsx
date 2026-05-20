"use client";

import { memo } from "react";
import type { NodeProps } from "@xyflow/react";
import { BaseNode } from "../base-node";
import type { ElasticsearchNodeData } from "@/lib/workflow-types";

function ElasticsearchNodeComponent({ data, selected, ...props }: NodeProps) {
  const nodeData = data as ElasticsearchNodeData;
  
  return (
    <BaseNode 
      nodeType="Elasticsearch" 
      data={nodeData} 
      selected={selected} 
      runStatus={nodeData.runStatus}
      {...props}
    >
      <div className="space-y-1.5">
        <div className="text-xs text-workflow-text-muted font-mono truncate">
          Elasticsearch
        </div>
        <div className="text-xs text-workflow-text-subtle font-mono">
          Index: {nodeData.index || 'log-*'}
        </div>
      </div>
    </BaseNode>
  );
}

export const ElasticsearchNode = memo(ElasticsearchNodeComponent);
