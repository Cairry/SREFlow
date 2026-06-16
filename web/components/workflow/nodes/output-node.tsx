"use client";

import { memo } from "react";
import { Handle, Position, type NodeProps } from "@xyflow/react";
import { BaseNode } from "../base-node";
import type { OutputNodeData } from "@/lib/workflow-types";

function OutputNodeComponent({ data, selected, ...props }: NodeProps) {
  const nodeData = data as OutputNodeData;
  const hasTemplate = !!(nodeData.contentTemplate && nodeData.contentTemplate.trim());
  const filename = nodeData.customFilename || "output";

  return (
    <BaseNode nodeType="Output" data={nodeData} selected={selected} hasOutput={false} runStatus={nodeData.runStatus} {...props}>
      <div className="space-y-1.5">
        <div className="text-xs text-workflow-text-muted font-mono">
          Markdown File
        </div>
        <div className="text-xs text-workflow-text-subtle font-mono truncate" title={filename}>
          {hasTemplate ? `${filename}.md` : "auto-concat"}
        </div>
      </div>
      {/* Transparent source handle to prevent reactflow from centering the node */}
      <Handle
        type="source"
        position={Position.Right}
        className="opacity-0 pointer-events-none"
      />
    </BaseNode>
  );
}

export const OutputNode = memo(OutputNodeComponent);
