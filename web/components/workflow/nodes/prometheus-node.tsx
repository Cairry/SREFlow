"use client";

import { memo } from "react";
import type { NodeProps } from "@xyflow/react";
import { BaseNode } from "../base-node";
import type { PrometheusNodeData } from "@/lib/workflow-types";

function PrometheusNodeComponent({ data, selected, ...props }: NodeProps) {
  const nodeData = data as PrometheusNodeData;
  
  return (
    <BaseNode 
      nodeType="Prometheus" 
      data={nodeData} 
      selected={selected} 
      runStatus={nodeData.runStatus}
      {...props}
    >
      <div className="space-y-1.5">
        <div className="text-xs text-workflow-text-muted font-mono truncate">
          Prometheus
        </div>
        <div className="text-xs text-workflow-text-subtle font-mono">
          PromQL Query
        </div>
      </div>
    </BaseNode>
  );
}

export const PrometheusNode = memo(PrometheusNodeComponent);
