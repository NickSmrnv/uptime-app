import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import DashboardPage from "./page";
import type { User } from "../../lib/api";

const replace = vi.fn();
const logout = vi.fn();
let authState: { status: "authenticated"; user: User; logout: typeof logout } = {
  status: "authenticated" as const,
  user: { id: "user-1", name: "Person", email: "person@example.com" },
  logout,
};

vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace }),
}));

vi.mock("../../auth/AuthProvider", () => ({
  useAuth: () => authState,
}));

describe("DashboardPage", () => {
  afterEach(() => {
    replace.mockReset();
    logout.mockReset();
    authState = {
      status: "authenticated",
      user: { id: "user-1", name: "Person", email: "person@example.com" },
      logout,
    };
  });

  it("shows the authenticated user in the dashboard layout", () => {
    render(<DashboardPage />);

    expect(screen.getByRole("heading", { name: "Добро пожаловать, person" })).toBeInTheDocument();
    expect(screen.getAllByText("person@example.com")).toHaveLength(1);
  });

  it("shows the uploaded avatar in the header", () => {
    authState = { ...authState, user: { ...authState.user, avatarUrl: "/uploads/avatars/avatar.png" } };
    process.env.NEXT_PUBLIC_API_URL = "http://api.test";
    render(<DashboardPage />);

    expect(screen.getByRole("img", { name: "Аватар Person" })).toHaveAttribute("src", "http://api.test/uploads/avatars/avatar.png");
  });

  it("logs out and returns the user to the login page", async () => {
    logout.mockResolvedValueOnce(undefined);
    render(<DashboardPage />);

    fireEvent.click(screen.getByRole("button", { name: "Выйти" }));

    await waitFor(() => expect(logout).toHaveBeenCalledTimes(1));
    expect(replace).toHaveBeenCalledWith("/login");
  });
});
