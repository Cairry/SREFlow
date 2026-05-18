"use client";

import { memo } from "react";
import { Handle, Position, type NodeProps } from "@xyflow/react";
import { BaseNode } from "../base-node";
import type { HTTPNodeData } from "@/lib/workflow-types";

function HTTPNodeComponent({ data, selected, ...props }: NodeProps<HTTPNodeData>) {
  const method = data.method || "POST";
  const url = data.url || "";

  return (
    <BaseNode nodeType="HTTP" data={data} selected={selected} runStatus={data.runStatus} {...props}>
      <div className="space-y-1.5">
        <div className="flex items-center gap-2">
          <span className="text-xs font-mono px-1.5 py-0.5 rounded bg-blue-500/20 text-blue-400">
            {method}
          </span>
          <span className="text-xs text-workflow-text-muted font-mono">
            {url ? (url.length > 20 ? url.substring(0, 17) + "..." : url) : "no URL"}
          </span>
        </div>
      </div>
    </BaseNode>
  );
}

export const HTTPNode = memo(HTTPNodeComponent);
