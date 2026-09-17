import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import DashboardPage from "./page";
import { ApiError, type Monitor, type User } from "../../lib/api";

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

  it("keeps a created monitor when the initial list resolves late", async () => {
    let resolveInitialList: (monitors: Monitor[]) => void;
    const initialList = new Promise<Monitor[]>((resolve) => {
      resolveInitialList = resolve;
    });
    apiFetch.mockImplementationOnce(() => initialList).mockResolvedValueOnce({ id: "monitor-1", url: "https://example.com", intervalSeconds: 60, createdAt: "2026-09-17T10:00:00Z" });
    render(<DashboardPage />);

    await waitFor(() => expect(apiFetch).toHaveBeenCalledWith("/monitors"));
    fireEvent.click(screen.getByRole("button", { name: "Добавить сайт" }));
    fireEvent.change(screen.getByLabelText("Адрес сайта"), { target: { value: "https://example.com" } });
    fireEvent.click(screen.getByRole("button", { name: "Создать" }));

    expect(await screen.findByText("https://example.com")).toBeInTheDocument();
    resolveInitialList!([]);
    await waitFor(() => expect(screen.getByText("https://example.com")).toBeInTheDocument());
  });

  it("shows a monitor limit error returned by the API", async () => {
    apiFetch.mockResolvedValueOnce([]).mockRejectedValueOnce(new ApiError(422, "monitor limit reached"));
    render(<DashboardPage />);

    await screen.findByText("Мониторов пока нет");
    fireEvent.click(screen.getByRole("button", { name: "Добавить сайт" }));
    fireEvent.change(screen.getByLabelText("Адрес сайта"), { target: { value: "https://example.com" } });
    fireEvent.click(screen.getByRole("button", { name: "Создать" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Достигнут лимит точек мониторинга для аккаунта.");
  });

  it("shows a loading error from the protected API", async () => {
    apiFetch.mockRejectedValueOnce(new Error("network error"));
    render(<DashboardPage />);

    expect(await screen.findByRole("alert")).toHaveTextContent("Не удалось загрузить точки мониторинга. Обновите страницу и попробуйте снова.");
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
