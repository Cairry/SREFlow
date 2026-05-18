// 状态管理适配器 - 桥接后端和前端状态

// 后端NodeState到前端NodeRunStatus的映射
export const mapBackendToFrontendStatus = (backendStatus: string): 'idle' | 'pending' | 'running' | 'completed' | 'failed' | 'skipped' => {
  switch (backendStatus) {
    case 'idle':
      return 'idle';
    case 'pending':
    case 'ready':
      return 'pending';
    case 'running':
      return 'running';
    case 'completed':
      return 'completed';
    case 'failed':
    case 'cancelled':
      return 'failed';
    case 'skipped':
      return 'skipped';
    default:
      return 'idle';
  }
};

// 前端NodeRunStatus到后端NodeState的映射
export const mapFrontendToBackendStatus = (frontendStatus: 'idle' | 'pending' | 'running' | 'completed' | 'failed' | 'skipped'): string => {
  switch (frontendStatus) {
    case 'idle':
      return 'idle';
    case 'pending':
      return 'pending';
    case 'running':
      return 'running';
    case 'completed':
      return 'completed';
    case 'failed':
      return 'failed';
    case 'skipped':
      return 'skipped';
    default:
      return 'idle';
  }
};

// 状态快照适配器 - 使用 snake_case 匹配后端 API 响应
export interface WorkflowStateSnapshot {
  executionId: string;
  workflowId: string;
  workflowName: string;
  status: string;
  startedAt: number;
  finishedAt?: number;
  runningNodes: string[];
  completedNodes: string[];
  failedNodes: string[];
  pendingNodes: string[];
  skippedNodes?: string[];
  totalNodes: number;
}

// 将后端状态快照转换为前端节点状态Map
export const adaptSnapshotToNodeStatuses = (
  snapshot: WorkflowStateSnapshot,
  allNodeIds: string[]
): Map<string, 'idle' | 'pending' | 'running' | 'completed' | 'failed' | 'skipped'> => {
  const nodeStatuses = new Map<string, 'idle' | 'pending' | 'running' | 'completed' | 'failed' | 'skipped'>();

  // 为所有节点初始化为idle状态
  allNodeIds.forEach(nodeId => {
    nodeStatuses.set(nodeId, 'idle');
  });

  // 设置明确的状态（优先级最高）
  snapshot.failedNodes.forEach(nodeId => {
    nodeStatuses.set(nodeId, 'failed');
  });

  snapshot.runningNodes.forEach(nodeId => {
    nodeStatuses.set(nodeId, 'running');
  });

  snapshot.completedNodes.forEach(nodeId => {
    nodeStatuses.set(nodeId, 'completed');
  });

  snapshot.pendingNodes.forEach(nodeId => {
    nodeStatuses.set(nodeId, 'pending');
  });

  // 处理跳过状态（孤立节点）
  if (snapshot.skippedNodes && snapshot.skippedNodes.length > 0) {
    snapshot.skippedNodes.forEach(nodeId => {
      nodeStatuses.set(nodeId, 'skipped');
    });
  }

  // 工作流已完成或失败时，对于未明确设置状态的节点进行推断
  if (snapshot.status === 'completed' || snapshot.status === 'failed') {
    const finalStatus: 'completed' | 'failed' = snapshot.status as 'completed' | 'failed';
    allNodeIds.forEach(nodeId => {
      const currentStatus = nodeStatuses.get(nodeId);
      // 只有在节点状态为idle时才进行推断（即后端没有明确追踪该节点状态）
      if (currentStatus === 'idle') {
        nodeStatuses.set(nodeId, finalStatus);
      }
    });
  }

  console.log('[StateAdapter] Final node statuses:', nodeStatuses);
  return nodeStatuses;
};

// 状态验证器
export const validateStateTransition = (
  from: string,
  to: string
): boolean => {
  const validTransitions: Record<string, string[]> = {
    'idle': ['pending'],
    'pending': ['ready', 'cancelled'],
    'ready': ['running', 'cancelled'],
    'running': ['completed', 'failed', 'cancelled'],
    'completed': [],
    'failed': ['pending'],
    'skipped': [],
    'cancelled': []
  };
  
  const allowed = validTransitions[from] || [];
  return allowed.includes(to);
};

// 状态聚合器
export const aggregateWorkflowState = (
  nodeStatuses: Map<string, 'idle' | 'pending' | 'running' | 'completed' | 'failed' | 'skipped'>
): {
  total: number;
  idle: number;
  pending: number;
  running: number;
  completed: number;
  failed: number;
  skipped: number;
  overallStatus: 'idle' | 'pending' | 'running' | 'completed' | 'failed';
} => {
  const counts = {
    total: nodeStatuses.size,
    idle: 0,
    pending: 0,
    running: 0,
    completed: 0,
    failed: 0,
    skipped: 0,
    overallStatus: 'idle' as 'idle' | 'pending' | 'running' | 'completed' | 'failed'
  };

  nodeStatuses.forEach(status => {
    switch (status) {
      case 'idle':
        counts.idle++;
        break;
      case 'pending':
        counts.pending++;
        break;
      case 'running':
        counts.running++;
        break;
      case 'completed':
        counts.completed++;
        break;
      case 'failed':
        counts.failed++;
        break;
      case 'skipped':
        counts.skipped++;
        break;
    }
  });

  // 确定整体状态
  if (counts.running > 0) {
    counts.overallStatus = 'running';
  } else if (counts.failed > 0) {
    counts.overallStatus = 'failed';
  } else if (counts.completed + counts.skipped === counts.total && counts.total > 0) {
    counts.overallStatus = 'completed';
  } else if (counts.pending > 0 || counts.idle > 0) {
    counts.overallStatus = 'pending';
  } else {
    counts.overallStatus = 'idle';
  }

  return counts;
};
