const API_BASE_URL = process.env.API_BASE_URL || "http://localhost:9001";

export interface HttpConfig {
  endpoint?: string;
  headers?: Record<string, string>;
  auth?: {
    type: string;
    username?: string;
    password?: string;
    token?: string;
    apiKey?: string;
    privateKey?: string;  // SSH private key (only for SSHAuth type)
  };
}

export interface AIConfig {
  api: string;
  apiKey: string;
  model: string;
  baseURL?: string;
}

export interface KubeConfig {
  kubeconfig: string;
}

export interface Datasource {
  id: string;
  workspaceId: string;
  name: string;
  description: string;
  type: string;
  httpConfig?: HttpConfig;
  aiConfig?: AIConfig;
  kubeConfig?: KubeConfig;
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

export const datasourceApi = {
  list(type?: string): Promise<{ datasources: Datasource[] }> {
    const qs = type ? `?type=${type}` : "";
    return request<{ datasources: Datasource[] }>(`/api/datasources${qs}`);
  },

  getById(id: string): Promise<Datasource> {
    return request<Datasource>(`/api/datasources/${id}`);
  },

  create(data: {
    name: string;
    description: string;
    type: string;
    httpConfig?: HttpConfig;
    aiConfig?: AIConfig;
    kubeConfig?: KubeConfig;
  }): Promise<Datasource> {
    return request<Datasource>("/api/datasources", {
      method: "POST",
      body: JSON.stringify(data),
    });
  },

  update(
    id: string,
    data: {
      name: string;
      description: string;
      httpConfig?: HttpConfig;
      aiConfig?: AIConfig;
    }
  ): Promise<Datasource> {
    return request<Datasource>(`/api/datasources/${id}`, {
      method: "PUT",
      body: JSON.stringify({ ...data, id }),
    });
  },

  delete(id: string): Promise<void> {
    return request<void>(`/api/datasources/${id}`, {
      method: "DELETE",
    });
  },

  getNamespaces(id: string): Promise<{ namespaces: string[] }> {
    return request<{ namespaces: string[] }>(`/api/datasources/${id}/namespaces`);
  },

  getPods(id: string, namespace: string): Promise<{ pods: string[] }> {
    return request<{ pods: string[] }>(`/api/datasources/${id}/pods?namespace=${encodeURIComponent(namespace)}`);
  },

  testK8sConnection(kubeconfig: string): Promise<{ connected: boolean; serverVersion: string; namespaceCount: number }> {
    return request("/api/datasources/test-k8s", {
      method: "POST",
      body: JSON.stringify({ kubeconfig }),
    });
  },
};
