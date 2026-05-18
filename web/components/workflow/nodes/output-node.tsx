"use client";

import { memo } from "react";
import { Handle, Position, type NodeProps } from "@xyflow/react";
import { BaseNode } from "../base-node";
import type { OutputNodeData } from "@/lib/workflow-types";

function OutputNodeComponent({ data, selected, ...props }: NodeProps<OutputNodeData>) {
  const getOutputTypeLabel = () => {
    switch (data.outputType) {
      case "file": return "File";
      case "webhook": return "Webhook";
      default: return "File";
    }
  };

  const getWebhookSummary = () => {
    if (data.outputType !== "webhook" || !data.webhook?.url) return null;
    const url = data.webhook.url;
    const shortUrl = url.length > 25 ? url.substring(0, 22) + "..." : url;
    return `${shortUrl}`;
  };

  return (
    <BaseNode nodeType="Output" data={data} selected={selected} hasOutput={false} runStatus={data.runStatus} {...props}>
      <div className="space-y-1.5">
        <div className="text-xs text-workflow-text-muted font-mono">
          {getOutputTypeLabel()}
        </div>
        {data.outputType === "webhook" && data.webhook?.url ? (
          <div className="text-xs text-workflow-text-subtle font-mono truncate" title={data.webhook.url}>
            {getWebhookSummary()}
          </div>
        ) : (
          <div className="text-xs text-workflow-text-subtle font-mono">
            Export Output
          </div>
        )}
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
