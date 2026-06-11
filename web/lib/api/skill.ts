const API_BASE_URL_SKILL = process.env.API_BASE_URL || "";

export interface Skill {
  id: string;
  workspaceId: string;
  name: string;
  description: string;
  content: string;
  enabled: boolean;
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

  const response = await fetch(`${API_BASE_URL_SKILL}${path}`, {
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

export const skillApi = {
  list(): Promise<{ skills: Skill[] }> {
    return request<{ skills: Skill[] }>("/api/skills");
  },

  getById(id: string): Promise<{ skill: Skill }> {
    return request<{ skill: Skill }>(`/api/skills/${id}`);
  },

  create(data: {
    name: string;
    description?: string;
    content: string;
    enabled?: boolean;
  }): Promise<{ skill: Skill }> {
    return request<{ skill: Skill }>("/api/skills", {
      method: "POST",
      body: JSON.stringify(data),
    });
  },

  update(
    id: string,
    data: {
      name?: string;
      description?: string;
      content?: string;
      enabled?: boolean;
    }
  ): Promise<{ skill: Skill }> {
    return request<{ skill: Skill }>(`/api/skills/${id}`, {
      method: "PUT",
      body: JSON.stringify({ ...data, id }),
    });
  },

  delete(id: string): Promise<{ message: string }> {
    return request<{ message: string }>(`/api/skills/${id}`, {
      method: "DELETE",
    });
  },
};
