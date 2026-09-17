import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import DashboardPage from "./page";
import type { User } from "../../lib/api";

const replace = vi.fn();
const logout = vi.fn();
const apiFetch = vi.fn();
let authState: { status: "authenticated"; user: User; logout: typeof logout; apiFetch: typeof apiFetch } = {
  status: "authenticated" as const,
  user: { id: "user-1", name: "Person", email: "person@example.com" },
  logout,
  apiFetch,
};

vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace }),
}));

vi.mock("../../auth/AuthProvider", () => ({
  useAuth: () => authState,
}));

describe("DashboardPage", () => {
  beforeEach(() => {
    apiFetch.mockResolvedValue([]);
  });

  afterEach(() => {
    replace.mockReset();
    logout.mockReset();
    apiFetch.mockReset();
    apiFetch.mockResolvedValue([]);
    authState = {
      status: "authenticated",
      user: { id: "user-1", name: "Person", email: "person@example.com" },
      logout,
      apiFetch,
    };
  });

  it("creates a monitor and shows it in the dashboard", async () => {
    apiFetch.mockResolvedValueOnce([]).mockResolvedValueOnce({ id: "monitor-1", url: "https://example.com", intervalSeconds: 300, createdAt: "2026-09-17T10:00:00Z" });
    render(<DashboardPage />);

    await waitFor(() => expect(apiFetch).toHaveBeenCalledWith("/monitors"));
    fireEvent.click(screen.getByRole("button", { name: "Добавить сайт" }));
    fireEvent.change(screen.getByLabelText("Адрес сайта"), { target: { value: "https://example.com" } });
    fireEvent.change(screen.getByLabelText("Интервал"), { target: { value: "5" } });
    fireEvent.change(screen.getByLabelText("Единица"), { target: { value: "minutes" } });
    fireEvent.click(screen.getByRole("button", { name: "Создать" }));

    await waitFor(() => expect(apiFetch).toHaveBeenCalledWith("/monitors", { method: "POST", body: JSON.stringify({ url: "https://example.com", intervalSeconds: 300 }) }));
    expect(await screen.findByText("https://example.com")).toBeInTheDocument();
    expect(screen.getByText("Каждые 5 мин.")).toBeInTheDocument();
  });

  it("shows the authenticated user in the dashboard layout", async () => {
    render(<DashboardPage />);

    await screen.findByText("Мониторов пока нет");
    expect(screen.getByRole("heading", { name: "Добро пожаловать, person" })).toBeInTheDocument();
    expect(screen.getAllByText("person@example.com")).toHaveLength(1);
  });

  it("shows the uploaded avatar in the header", async () => {
    authState = { ...authState, user: { ...authState.user, avatarUrl: "/uploads/avatars/avatar.png" } };
    process.env.NEXT_PUBLIC_API_URL = "http://api.test";
    render(<DashboardPage />);

    await screen.findByText("Мониторов пока нет");
    expect(screen.getByRole("img", { name: "Аватар Person" })).toHaveAttribute("src", "http://api.test/uploads/avatars/avatar.png");
  });

  it("logs out and returns the user to the login page", async () => {
    logout.mockResolvedValueOnce(undefined);
    render(<DashboardPage />);

    await screen.findByText("Мониторов пока нет");
    fireEvent.click(screen.getByRole("button", { name: "Выйти" }));

    await waitFor(() => expect(logout).toHaveBeenCalledTimes(1));
    expect(replace).toHaveBeenCalledWith("/login");
  });
});
