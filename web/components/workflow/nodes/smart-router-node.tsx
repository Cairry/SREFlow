"use client";

import { memo, useMemo } from "react";
import type { NodeProps } from "@xyflow/react";
import { BaseNode } from "../base-node";
import type { SmartRouterNodeData } from "@/lib/workflow-types";

function SmartRouterNodeComponent({ data, selected, ...props }: NodeProps) {
  const nodeData = data as SmartRouterNodeData;

  const { branchHandles, branchCount } = useMemo(() => {
    try {
      const branches = JSON.parse(nodeData.branches || "[]");
      if (!Array.isArray(branches)) return { branchHandles: [], branchCount: 0 };
      return {
        branchHandles: branches.map((b: { id: string; label: string }) => ({
          id: b.id,
          label: b.label,
        })),
        branchCount: branches.length,
      };
    } catch {
      return { branchHandles: [], branchCount: 0 };
    }
  }, [nodeData.branches]);

  return (
    <BaseNode
      nodeType="SmartRouter"
      data={nodeData}
      selected={selected}
      runStatus={nodeData.runStatus}
      hasOutput={false}
      branchHandles={branchHandles}
      {...props}
    >
      <div className="space-y-1.5">
        <div className="text-xs text-amber-600 dark:text-amber-400 font-mono truncate">
          {nodeData.model || "AI Router"}
        </div>
        {branchCount > 0 ? (
          <div className="flex flex-wrap gap-1">
            {branchHandles.map((b) => (
              <span
                key={b.id}
                className="text-[10px] font-mono px-1.5 py-0.5 rounded bg-amber-500/10 text-amber-600 dark:text-amber-400 border border-amber-500/20"
              >
                {b.label}
              </span>
            ))}
          </div>
        ) : (
          <div className="text-xs text-workflow-text-subtle font-mono italic">
            Click to configure branches
          </div>
        )}
      </div>
    </BaseNode>
  );
}

export const SmartRouterNode = memo(SmartRouterNodeComponent);
