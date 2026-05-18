"use client";

import { memo } from "react";
import type { NodeProps } from "@xyflow/react";
import { BaseNode } from "../base-node";
import type { GoScriptNodeData } from "@/lib/workflow-types";

function GoScriptNodeComponent({ data, selected, ...props }: NodeProps) {
  const nodeData = data as GoScriptNodeData;
  return (
    <BaseNode
      nodeType="GoScript"
      data={nodeData}
      selected={selected}
      runStatus={nodeData.runStatus}
      {...props}
    >
      <div className="space-y-1.5">
        <div className="text-xs text-workflow-text-muted font-mono">
          GoScript
          {nodeData.timeout && ` (${nodeData.timeout}ms)`}
        </div>
        <div className="text-xs text-workflow-text-subtle font-mono">
          Custom Code
        </div>
      </div>
    </BaseNode>
  );
}

export const GoScriptNode = memo(GoScriptNodeComponent);
