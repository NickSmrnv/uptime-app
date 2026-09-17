import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { AuthProvider, useAuth } from "./AuthProvider";

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
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
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("restores a session from the HttpOnly refresh cookie", async () => {
    vi.mocked(fetch).mockResolvedValueOnce(jsonResponse({ accessToken: "access", user: { id: "1", name: "Person", email: "person@example.com" } }));
    render(<AuthProvider><Probe /></AuthProvider>);

    await screen.findByText("authenticated:person@example.com");
    expect(fetch).toHaveBeenCalledWith("http://api.test/auth/refresh", expect.objectContaining({ credentials: "include", method: "POST" }));
    expect(sessionStorage.getItem("accessToken")).toBeNull();
  });

  it("stores a login session only in React state", async () => {
    vi.mocked(fetch)
      .mockResolvedValueOnce(jsonResponse({ error: "invalid refresh token" }, 401))
      .mockResolvedValueOnce(jsonResponse({ accessToken: "access", user: { id: "1", name: "Person", email: "person@example.com" } }));
    render(<AuthProvider><Probe /></AuthProvider>);

    await screen.findByText("anonymous:");
    fireEvent.click(screen.getByRole("button", { name: "login" }));
    await screen.findByText("authenticated:person@example.com");
    expect(sessionStorage.getItem("accessToken")).toBeNull();
  });

  it("creates an account and clears the session on logout", async () => {
    vi.mocked(fetch)
      .mockResolvedValueOnce(jsonResponse({ error: "invalid refresh token" }, 401))
      .mockResolvedValueOnce(jsonResponse({ accessToken: "access", user: { id: "2", name: "New person", email: "new@example.com" } }))
      .mockResolvedValueOnce(new Response(null, { status: 204 }));
    render(<AuthProvider><Probe /></AuthProvider>);

    await screen.findByText("anonymous:");
    fireEvent.click(screen.getByRole("button", { name: "register" }));
    await screen.findByText("authenticated:new@example.com");
    fireEvent.click(screen.getByRole("button", { name: "logout" }));
    await waitFor(() => expect(screen.getByText("anonymous:")).toBeInTheDocument());
  });
});
