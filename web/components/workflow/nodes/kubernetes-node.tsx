"use client";

import { memo } from "react";
import type { NodeProps } from "@xyflow/react";
import { BaseNode } from "../base-node";
import type { KubernetesNodeData } from "@/lib/workflow-types";
import { useDatasource } from "../datasource-context";

function KubernetesNodeComponent({ data, selected, ...props }: NodeProps) {
  const nodeData = data as KubernetesNodeData;
  const { getDatasourceName } = useDatasource();

  const displayText = getDatasourceName(nodeData.datasourceId);
  const nsDisplay = nodeData.namespace || "-";
  const podDisplay = nodeData.pod || "-";
  const typeDisplay = nodeData.resourceType === "event" ? "Events" : "Pod Status";

  return (
    <BaseNode
      nodeType="Kubernetes"
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
          {nsDisplay}/{podDisplay} - {typeDisplay}
        </div>
      </div>
    </BaseNode>
  );
}

export const KubernetesNode = memo(KubernetesNodeComponent);
