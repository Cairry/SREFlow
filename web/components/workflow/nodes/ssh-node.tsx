"use client";

import { memo } from "react";
import type { NodeProps } from "@xyflow/react";
import { BaseNode } from "../base-node";
import type { SSHNodeData } from "@/lib/workflow-types";
import { useDatasource } from "../datasource-context";

function SSHNodeComponent({ data, selected, ...props }: NodeProps) {
  const nodeData = data as SSHNodeData;
  const { getDatasourceName } = useDatasource();

  const displayText = getDatasourceName(nodeData.datasourceId);
  const hostDisplay = nodeData.host ? `${nodeData.host}:${nodeData.port || 22}` : "-";

  return (
    <BaseNode
      nodeType="SSH"
      data={nodeData}
      selected={selected}
      runStatus={nodeData.runStatus}
      {...props}
    >
      <div className="space-y-1.5">
        <div className="text-xs text-workflow-text-muted font-mono truncate">
          {displayText || "No datasource"}
        </div>
        <div className="text-xs text-workflow-text-subtle font-mono truncate">
          {hostDisplay}
        </div>
      </div>
    </BaseNode>
  );
}

export const SSHNode = memo(SSHNodeComponent);
