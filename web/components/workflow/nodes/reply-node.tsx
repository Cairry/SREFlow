"use client";

import { memo } from "react";
import { Handle, Position, type NodeProps } from "@xyflow/react";
import { BaseNode } from "../base-node";
import type { ReplyNodeData } from "@/lib/workflow-types";

function ReplyNodeComponent({ data, selected, ...props }: NodeProps) {
  const nodeData = data as ReplyNodeData;
  const hasContent = !!(nodeData.contentTemplate && nodeData.contentTemplate.trim());

  return (
    <BaseNode nodeType="Reply" data={nodeData} selected={selected} hasOutput={false} runStatus={nodeData.runStatus} {...props}>
      <div className="space-y-1.5">
        <div className="text-xs text-workflow-text-muted font-mono">
          Direct Reply
        </div>
        <div className="text-xs text-workflow-text-subtle font-mono truncate" title={hasContent ? "custom content" : "auto-concat"}>
          {hasContent ? "✓ custom content" : "auto-concat"}
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

export const ReplyNode = memo(ReplyNodeComponent);
