import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { AuthProvider, useAuth } from "./AuthProvider";

const { axiosClient, axiosRequest } = vi.hoisted(() => {
  const axiosRequest = vi.fn();
  return {
    axiosRequest,
    axiosClient: { request: axiosRequest, defaults: { headers: { common: {} as Record<string, string> } } },
  };
});

vi.mock("axios", () => ({
  default: {
    create: vi.fn(() => axiosClient),
    isAxiosError: (error: unknown) => typeof error === "object" && error !== null && "isAxiosError" in error && error.isAxiosError === true,
  },
}));

function axiosError(status: number, message: string) {
  return { isAxiosError: true, response: { status, data: { error: message } } };
}

function Probe() {
  const auth = useAuth();
  return (
    <>
      <p>{`${auth.status}:${auth.user?.email ?? ""}`}</p>
      <button onClick={() => void auth.login("person@example.com", "password")}>login</button>
      <button onClick={() => void auth.register("New person", "new@example.com", "password")}>register</button>
      <button onClick={() => void auth.logout()}>logout</button>
    </>
  );
}

describe("AuthProvider", () => {
  beforeEach(() => {
    process.env.NEXT_PUBLIC_API_URL = "http://api.test";
    axiosRequest.mockReset();
    axiosClient.defaults.headers.common = {};
  });

  afterEach(() => vi.clearAllMocks());

  it("restores a session from the HttpOnly refresh cookie", async () => {
    axiosRequest.mockResolvedValueOnce({ status: 200, data: { accessToken: "access", user: { id: "1", name: "Person", email: "person@example.com" } } });
    render(<AuthProvider><Probe /></AuthProvider>);

    await screen.findByText("authenticated:person@example.com");
    expect(axiosRequest).toHaveBeenCalledWith(expect.objectContaining({ url: "http://api.test/auth/refresh", method: "POST" }));
    expect(axiosClient.defaults.headers.common.Authorization).toBe("Bearer access");
    expect(sessionStorage.getItem("accessToken")).toBeNull();
  });

  it("stores a login session only in React state", async () => {
    axiosRequest
      .mockRejectedValueOnce(axiosError(401, "invalid refresh token"))
      .mockResolvedValueOnce({ status: 200, data: { accessToken: "access", user: { id: "1", name: "Person", email: "person@example.com" } } });
    render(<AuthProvider><Probe /></AuthProvider>);

    await screen.findByText("anonymous:");
    fireEvent.click(screen.getByRole("button", { name: "login" }));
    await screen.findByText("authenticated:person@example.com");
    expect(axiosClient.defaults.headers.common.Authorization).toBe("Bearer access");
    expect(sessionStorage.getItem("accessToken")).toBeNull();
  });

  it("creates an account and clears the session on logout", async () => {
    axiosRequest
      .mockRejectedValueOnce(axiosError(401, "invalid refresh token"))
      .mockResolvedValueOnce({ status: 200, data: { accessToken: "access", user: { id: "2", name: "New person", email: "new@example.com" } } })
      .mockResolvedValueOnce({ status: 204, data: "" });
    render(<AuthProvider><Probe /></AuthProvider>);

    await screen.findByText("anonymous:");
    fireEvent.click(screen.getByRole("button", { name: "register" }));
    await screen.findByText("authenticated:new@example.com");
    fireEvent.click(screen.getByRole("button", { name: "logout" }));
    await waitFor(() => expect(screen.getByText("anonymous:")).toBeInTheDocument());
    expect(axiosClient.defaults.headers.common.Authorization).toBeUndefined();
  });
});
