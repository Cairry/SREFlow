export const API_BASE_URL = process.env.API_BASE_URL || "http://localhost:9001";

export interface WorkflowNode {
  id: string;
  type: string;
  position: { x: number; y: number };
  data: Record<string, unknown>;
}

export interface WorkflowEdge {
  id: string;
  source: string;
  target: string;
  type?: string;
}

export interface RunHistoryItem {
  id: string;
  workflowId: string;
  workflowName: string;
  status: string;
  finalOutput: string | null;
  startedAt: number;
  finishedAt: number | null;
  createdAt: number;
  updatedAt: number;
}

export interface Workflow {
  id: string;
  workspaceId: string;
  name: string;
  description: string;
  nodes: WorkflowNode[];
  edges: WorkflowEdge[];
  createdAt: string;
  updatedAt: string;
}

function getWorkspaceId(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("flowforge-current-workspace-id");
}

async function request<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const workspaceId = getWorkspaceId();
  const headers: HeadersInit = {
    "Content-Type": "application/json",
    ...(workspaceId ? { "X-Workspace-Id": workspaceId } : {}),
    ...options.headers,
  };

  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...options,
    headers,
  });

  const result = await response.json();

  if (result.code === 200 && result.data !== undefined) {
    return result.data as T;
  }

  if (!response.ok) {
    throw new Error(result.msg || result.Message || `Request failed with status ${response.status}`);
  }

  throw new Error(result.msg || result.Message || "API request failed");
}

export interface WorkflowVersion {
  workflowId: string;
  version: number;
  name: string;
  description: string;
  nodes: WorkflowNode[];
  edges: WorkflowEdge[];
  isRelease: boolean;
  createdAt: string;
}

export const workflowApi = {
  list(q?: string): Promise<{ workflows: Workflow[] }> {
    const qs = q ? `?q=${encodeURIComponent(q)}` : "";
    return request<{ workflows: Workflow[] }>(`/api/workflows${qs}`);
  },

  getById(id: string): Promise<Workflow> {
    return request<Workflow>(`/api/workflow/${id}`);
  },

  create(data: {
    name: string;
    description: string;
    nodes?: WorkflowNode[];
    edges?: WorkflowEdge[];
  }): Promise<Workflow> {
    const body = {
      ...data,
      workspaceId: getWorkspaceId(),
      nodes: data.nodes ?? [
        {
          id: "start",
          type: "Start",
          position: { x: 100, y: 200 },
          data: { label: "Start", text: "" },
        },
      ],
      edges: data.edges ?? [],
    };
    return request<Workflow>("/api/workflows", {
      method: "POST",
      body: JSON.stringify(body),
    });
  },

  update(
    id: string,
    data: {
      name: string;
      description: string;
      nodes: WorkflowNode[];
      edges: WorkflowEdge[];
    }
  ): Promise<Workflow> {
    return request<Workflow>(`/api/workflow/${id}`, {
      method: "PUT",
      body: JSON.stringify({
        id,
        workspaceId: getWorkspaceId(),
        ...data,
      }),
    });
  },

  delete(id: string): Promise<void> {
    return request<void>(`/api/workflow/${id}`, {
      method: "DELETE",
    });
  },

  execute(workflowId: string, input?: string): Promise<{ executionId: string; workflowId: string; status: string }> {
    const body: Record<string, unknown> = {};
    if (input !== undefined) {
      body.input = input;
    }
    return request<{ executionId: string; workflowId: string; status: string }>(
      `/api/workflow/${workflowId}/execute`,
      { method: "POST", body: JSON.stringify(body) }
    );
  },

  getExecutionStatus(executionId: string): Promise<{
    status: string;
    completedNodes: string[];
    failedNodes: string[];
    totalNodes: number;
  }> {
    return request(`/api/workflow/executions/${executionId}/status`);
  },

  getExecution(executionId: string): Promise<{
    id: string;
    workflowId: string;
    status: string;
    nodeResults: Record<string, any>;
    variables: Record<string, any>;
    startedAt: number;
    finishedAt: number | null;
  }> {
    return request(`/api/workflow/executions/${executionId}`);
  },

  cancelExecution(executionId: string): Promise<void> {
    return request<void>(`/api/workflow/executions/${executionId}/cancel`, {
      method: "POST",
    });
  },

  getRunHistory(workflowId: string, limit = 50): Promise<{ history: RunHistoryItem[] }> {
    return request<{ history: RunHistoryItem[] }>(
      `/api/workflow/${workflowId}/run-history?limit=${limit}`
    );
  },

  listRunHistory(limit = 100): Promise<{ history: RunHistoryItem[] }> {
    return request<{ history: RunHistoryItem[] }>(
      `/api/workflow/run-history?limit=${limit}`
    );
  },

  deleteRunHistory(executionId: string): Promise<void> {
    return request<void>(`/api/workflow/run-history/${executionId}`, {
      method: "DELETE",
    });
  },

  getVersionHistory(workflowId: string, limit = 50): Promise<{ workflowId: string; versions: WorkflowVersion[]; total: number; currentVersionNum: number }> {
    return request<{ workflowId: string; versions: WorkflowVersion[]; total: number; currentVersionNum: number }>(
      `/api/workflow/${workflowId}/versions?limit=${limit}`
    );
  },

  getVersion(workflowId: string, version: number): Promise<WorkflowVersion> {
    return request<WorkflowVersion>(`/api/workflow/${workflowId}/versions/${version}`);
  },

  gotoVersion(workflowId: string, version: number): Promise<{
    id: string;
    versionNum: number;
    contentHash: string;
    nodes: WorkflowNode[];
    edges: WorkflowEdge[];
    name: string;
    description: string;
    currentVersionNum: number;
  }> {
    return request<{
      id: string;
      versionNum: number;
      contentHash: string;
      nodes: WorkflowNode[];
      edges: WorkflowEdge[];
      name: string;
      description: string;
      currentVersionNum: number;
    }>(`/api/workflow/${workflowId}/goto/${version}`, {
      method: "POST",
    });
  },

  deleteVersion(workflowId: string, version: number): Promise<{ message: string; workflowId: string; versionNum: number }> {
    return request<{ message: string; workflowId: string; versionNum: number }>(
      `/api/workflow/${workflowId}/versions/${version}`,
      { method: "DELETE" }
    );
  },

  setRelease(workflowId: string, version: number): Promise<{
    message: string;
    workflowId: string;
    versionNum: number;
    name: string;
    description: string;
  }> {
    return request<{
      message: string;
      workflowId: string;
      versionNum: number;
      name: string;
      description: string;
    }>(`/api/workflow/${workflowId}/release/${version}`, {
      method: "POST",
    });
  },

  cancelRelease(workflowId: string, version: number): Promise<{ message: string; workflowId: string; versionNum: number }> {
    return request<{ message: string; workflowId: string; versionNum: number }>(
      `/api/workflow/${workflowId}/release/${version}`,
      { method: "DELETE" }
    );
  },
};
