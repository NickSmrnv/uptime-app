export type User = {
  id: string;
  name: string;
  email: string;
  avatarUrl?: string;
};

export type AuthResponse = {
  accessToken: string;
  user: User;
};

export type UploadedFile = {
  key: string;
  url: string;
  filename: string;
  size: number;
};

export class ApiError extends Error {
  constructor(
    public readonly status: number,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

function apiURL(path: string): string {
  const baseURL = process.env.NEXT_PUBLIC_API_URL?.replace(/\/$/, "");

  if (!baseURL) {
    throw new Error("NEXT_PUBLIC_API_URL is not configured");
  }

  return `${baseURL}${path}`;
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  const isFormData = typeof FormData !== "undefined" && init.body instanceof FormData;

  if (!headers.has("Content-Type") && !isFormData) {
    headers.set("Content-Type", "application/json");
  }

  const response = await fetch(apiURL(path), {
    ...init,
    credentials: "include",
    headers,
  });
  const payload = response.status === 204 ? null : await response.json().catch(() => null);

  if (!response.ok) {
    const message = typeof payload?.error === "string" ? payload.error : "Request failed";
    throw new ApiError(response.status, message);
  }

  return payload as T;
}

export function avatarSource(avatarURL: string): string {
  if (/^https?:\/\//.test(avatarURL)) {
    return avatarURL;
  }

  return apiURL(avatarURL);
}

export const api = {
  register(name: string, email: string, password: string) {
    return request<AuthResponse>("/auth/register", {
      method: "POST",
      body: JSON.stringify({ name, email, password }),
    });
  },
  login(email: string, password: string) {
    return request<AuthResponse>("/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password }),
    });
  },
  refresh() {
    return request<AuthResponse>("/auth/refresh", { method: "POST" });
  },
  logout() {
    return request<void>("/auth/logout", { method: "POST" });
  },
  profile(accessToken: string) {
    return this.authorized<User>("/users/me", accessToken, { method: "GET" });
  },
  updateProfile(accessToken: string, name: string) {
    return this.authorized<User>("/users/me", accessToken, {
      method: "PATCH",
      body: JSON.stringify({ name }),
    });
  },
  uploadAvatar(accessToken: string, file: File) {
    const formData = new FormData();
    formData.append("avatar", file);
    return this.authorized<User>("/users/me/avatar", accessToken, {
      method: "PUT",
      body: formData,
    });
  },
  uploadFile(accessToken: string, file: File) {
    const formData = new FormData();
    formData.append("file", file);
    return this.authorized<UploadedFile>("/uploads", accessToken, {
      method: "POST",
      body: formData,
    });
  },
  authorized<T>(path: string, accessToken: string, init: RequestInit = {}) {
    return request<T>(path, {
      ...init,
      headers: {
        Authorization: `Bearer ${accessToken}`,
        ...init.headers,
      },
    });
  },
};
