import axios from "axios";

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

export type Monitor = {
  id: string;
  url: string;
  intervalSeconds: number;
  createdAt: string;
  status: "pending" | "up" | "down" | "stale";
  configVersion: number;
  historyVersion: number;
  nextCheckAt: string;
  lastCheckedAt: string | null;
  lastStatusCode: number | null;
  lastError: string;
  lastDurationMs: number | null;
};

export type MonitorPeriod = "1h" | "24h" | "7d" | "30d";

export type MonitorBucket = {
  start: string;
  successes: number;
  failures: number;
  successPercent: number | null;
  failurePercent: number | null;
};

export type MonitorStats = {
  period: MonitorPeriod;
  from: string;
  to: string;
  bucketSeconds: number;
  historyVersion: number;
  successes: number;
  failures: number;
  successPercent: number | null;
  failurePercent: number | null;
  buckets: MonitorBucket[];
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

const apiClient = axios.create({ withCredentials: true });

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  const isFormData = typeof FormData !== "undefined" && init.body instanceof FormData;

  if (!headers.has("Content-Type") && !isFormData) {
    headers.set("Content-Type", "application/json");
  }

  const axiosHeaders: Record<string, string> = {};
  headers.forEach((value, key) => {
    axiosHeaders[key] = value;
  });

  try {
    const response = await apiClient.request<T>({
      url: apiURL(path),
      method: init.method,
      data: init.body,
      headers: axiosHeaders,
      signal: init.signal ?? undefined,
    });
    return response.status === 204 ? undefined as T : response.data;
  } catch (error) {
    if (axios.isAxiosError(error)) {
      const message = typeof error.response?.data?.error === "string" ? error.response.data.error : "Request failed";
      throw new ApiError(error.response?.status ?? 0, message);
    }
    throw error;
  }
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
  setAccessToken(accessToken: string | null) {
    if (accessToken) {
      apiClient.defaults.headers.common.Authorization = `Bearer ${accessToken}`;
      return;
    }
    delete apiClient.defaults.headers.common.Authorization;
  },
  request<T>(path: string, init: RequestInit = {}) {
    return request<T>(path, init);
  },
};
