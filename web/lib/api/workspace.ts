const API_BASE_URL = process.env.API_BASE_URL || "";

export interface Workspace {
  id: string;
  name: string;
  desc?: string;
  ownerId: string;
}

async function request<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...options.headers,
    },
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

export const workspaceApi = {
  list(): Promise<{ workspaces: Workspace[] }> {
    return request<{ workspaces: Workspace[] }>("/api/workspaces");
  },

  create(data: { name: string; ownerId: string }): Promise<Workspace> {
    return request<Workspace>("/api/workspaces", {
      method: "POST",
      body: JSON.stringify(data),
    });
  },

  update(id: string, data: { name: string }): Promise<Workspace> {
    return request<Workspace>(`/api/workspaces/${id}`, {
      method: "PUT",
      body: JSON.stringify(data),
    });
  },

  delete(id: string): Promise<void> {
    return request<void>(`/api/workspaces?id=${encodeURIComponent(id)}`, {
      method: "DELETE",
    });
  },
};
