"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { api, ApiError, type AuthResponse, type UploadedFile, type User } from "../lib/api";

type AuthStatus = "loading" | "authenticated" | "anonymous";

type AuthContextValue = {
  status: AuthStatus;
  user: User | null;
  accessToken: string | null;
  register: (name: string, email: string, password: string) => Promise<void>;
  login: (email: string, password: string) => Promise<void>;
  refresh: () => Promise<AuthResponse | null>;
  logout: () => Promise<void>;
  updateProfile: (name: string) => Promise<User>;
  uploadAvatar: (file: File) => Promise<User>;
  uploadFile: (file: File) => Promise<UploadedFile>;
  apiFetch: <T>(path: string, init?: RequestInit) => Promise<T>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<AuthStatus>("loading");
  const [user, setUser] = useState<User | null>(null);
  const [accessToken, setAccessToken] = useState<string | null>(null);
  const refreshInFlight = useRef<Promise<AuthResponse | null> | null>(null);

  const applySession = useCallback((session: AuthResponse) => {
    setAccessToken(session.accessToken);
    setUser(session.user);
    setStatus("authenticated");
  }, []);

  const clearSession = useCallback(() => {
    setAccessToken(null);
    setUser(null);
    setStatus("anonymous");
  }, []);

  const refresh = useCallback((): Promise<AuthResponse | null> => {
    if (refreshInFlight.current) {
      return refreshInFlight.current;
    }

    const pending = api
      .refresh()
      .then((session) => {
        applySession(session);
        return session;
      })
      .catch((error: unknown) => {
        if (error instanceof ApiError && error.status === 401) {
          clearSession();
          return null;
        }
        clearSession();
        return null;
      })
      .finally(() => {
        refreshInFlight.current = null;
      });

    refreshInFlight.current = pending;
    return pending;
  }, [applySession, clearSession]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const register = useCallback(
    async (name: string, email: string, password: string) => {
      const session = await api.register(name, email, password);
      applySession(session);
    },
    [applySession],
  );

  const login = useCallback(
    async (email: string, password: string) => {
      const session = await api.login(email, password);
      applySession(session);
    },
    [applySession],
  );

  const logout = useCallback(async () => {
    try {
      await api.logout();
    } finally {
      clearSession();
    }
  }, [clearSession]);

  const apiFetch = useCallback(
    async <T,>(path: string, init: RequestInit = {}) => {
      if (!accessToken) {
        throw new Error("The user is not authenticated");
      }

      try {
        return await api.authorized<T>(path, accessToken, init);
      } catch (error) {
        if (!(error instanceof ApiError) || error.status !== 401) {
          throw error;
        }

        const session = await refresh();
        if (!session) {
          throw error;
        }

        return api.authorized<T>(path, session.accessToken, init);
      }
    },
    [accessToken, refresh],
  );

  const updateProfile = useCallback(async (name: string) => {
    const updatedUser = await apiFetch<User>("/users/me", {
      method: "PATCH",
      body: JSON.stringify({ name }),
    });
    setUser(updatedUser);
    return updatedUser;
  }, [apiFetch]);

  const uploadAvatar = useCallback(async (file: File) => {
    if (!accessToken) {
      throw new Error("The user is not authenticated");
    }
    try {
      const updatedUser = await api.uploadAvatar(accessToken, file);
      setUser(updatedUser);
      return updatedUser;
    } catch (error) {
      if (!(error instanceof ApiError) || error.status !== 401) {
        throw error;
      }
      const session = await refresh();
      if (!session) {
        throw error;
      }
      const updatedUser = await api.uploadAvatar(session.accessToken, file);
      setUser(updatedUser);
      return updatedUser;
    }
  }, [accessToken, refresh]);

  const uploadFile = useCallback(async (file: File) => {
    if (!accessToken) {
      throw new Error("The user is not authenticated");
    }
    try {
      return await api.uploadFile(accessToken, file);
    } catch (error) {
      if (!(error instanceof ApiError) || error.status !== 401) {
        throw error;
      }
      const session = await refresh();
      if (!session) {
        throw error;
      }
      return api.uploadFile(session.accessToken, file);
    }
  }, [accessToken, refresh]);

  const value = useMemo(
    () => ({ status, user, accessToken, register, login, refresh, logout, updateProfile, uploadAvatar, uploadFile, apiFetch }),
    [accessToken, apiFetch, login, logout, refresh, register, status, updateProfile, uploadAvatar, uploadFile, user],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);

  if (!context) {
    throw new Error("useAuth must be used inside AuthProvider");
  }

  return context;
}
