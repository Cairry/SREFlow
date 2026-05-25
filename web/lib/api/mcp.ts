const API_BASE_URL_MCP = process.env.API_BASE_URL || "";

export interface MCPServerTool {
  name: string;
  description: string;
  input_schema: Record<string, unknown>;
}

export interface MCPServer {
  id: string;
  workspaceId: string;
  name: string;
  type: string;
  url: string;
  headers?: Record<string, string>;
  apiKey?: string;
  timeout: number;
  enabled: boolean;
  description: string;
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

  const response = await fetch(`${API_BASE_URL_MCP}${path}`, {
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

export const mcpApi = {
  list(): Promise<{ servers: MCPServer[] }> {
    return request<{ servers: MCPServer[] }>("/api/mcp-servers");
  },

  getById(id: string): Promise<MCPServer> {
    return request<MCPServer>(`/api/mcp-servers/${id}`);
  },

  create(data: {
    name: string;
    type: string;
    url: string;
    headers?: Record<string, string>;
    apiKey?: string;
    timeout?: number;
    enabled?: boolean;
    description?: string;
  }): Promise<MCPServer> {
    return request<MCPServer>("/api/mcp-servers", {
      method: "POST",
      body: JSON.stringify(data),
    });
  },

  update(
    id: string,
    data: {
      name?: string;
      type?: string;
      url?: string;
      headers?: Record<string, string>;
      apiKey?: string;
      timeout?: number;
      enabled?: boolean;
      description?: string;
    }
  ): Promise<MCPServer> {
    return request<MCPServer>(`/api/mcp-servers/${id}`, {
      method: "PUT",
      body: JSON.stringify({ ...data, id }),
    });
  },

  delete(id: string): Promise<void> {
    return request<void>(`/api/mcp-servers/${id}`, {
      method: "DELETE",
    });
  },

  listTools(id: string): Promise<{ tools: MCPServerTool[] }> {
    return request<{ tools: MCPServerTool[] }>(`/api/mcp-servers/${id}/tools`);
  },

  getStatus(id: string): Promise<{ connected: boolean; message?: string }> {
    return request<{ connected: boolean; message?: string }>(`/api/mcp-servers/${id}/status`);
  },
};
