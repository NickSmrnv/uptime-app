import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import ProfilePage from "./page";

const replace = vi.fn();
const apiFetch = vi.fn();
const updateProfile = vi.fn();
const uploadAvatar = vi.fn();

vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace }),
}));

vi.mock("../../auth/AuthProvider", () => ({
  useAuth: () => ({
    status: "authenticated",
    user: { id: "user-1", name: "Alex", email: "person@example.com" },
    apiFetch,
    updateProfile,
    uploadAvatar,
  }),
}));

describe("ProfilePage", () => {
  afterEach(() => {
    replace.mockReset();
    apiFetch.mockReset();
    updateProfile.mockReset();
    uploadAvatar.mockReset();
  });

  it("loads the current profile through the protected API route", async () => {
    apiFetch.mockResolvedValue({ id: "user-1", name: "Alex", email: "person@example.com" });
    render(<ProfilePage />);

    await waitFor(() => expect(apiFetch).toHaveBeenCalledWith("/users/me"));
    expect(screen.getByDisplayValue("person@example.com")).toBeInTheDocument();
    expect(screen.getByDisplayValue("Alex")).toBeInTheDocument();
  });

  it("sends the edited name to the backend", async () => {
    apiFetch.mockResolvedValue({ id: "user-1", name: "Alex", email: "person@example.com" });
    updateProfile.mockResolvedValueOnce({ id: "user-1", name: "Alexandra", email: "person@example.com" });
    render(<ProfilePage />);

    await screen.findByDisplayValue("Alex");
    fireEvent.change(screen.getByLabelText("Имя"), { target: { value: "Alexandra" } });
    fireEvent.click(screen.getByRole("button", { name: "Сохранить" }));

    await waitFor(() => expect(updateProfile).toHaveBeenCalledWith("Alexandra"));
  });

  it("uploads a selected avatar when the profile is saved", async () => {
    apiFetch.mockResolvedValue({ id: "user-1", name: "Alex", email: "person@example.com" });
    uploadAvatar.mockResolvedValueOnce({ id: "user-1", name: "Alex", email: "person@example.com", avatarUrl: "/uploads/avatars/avatar.png" });
    render(<ProfilePage />);

    await screen.findByDisplayValue("Alex");
    const file = new File(["image"], "avatar.png", { type: "image/png" });
    fireEvent.change(screen.getByLabelText("Изменить аватар"), { target: { files: [file] } });
    fireEvent.click(screen.getByRole("button", { name: "Сохранить" }));

    await waitFor(() => expect(uploadAvatar).toHaveBeenCalledWith(file));
  });
});
