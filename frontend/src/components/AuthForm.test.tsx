import { fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { AuthProvider } from "../auth/AuthProvider";
import { AuthForm } from "./AuthForm";

const replace = vi.fn();

vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace }),
}));

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

describe("AuthForm", () => {
  beforeEach(() => {
    process.env.NEXT_PUBLIC_API_URL = "http://api.test";
    replace.mockReset();
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(jsonResponse({ error: "invalid refresh token" }, 401)));
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("validates a registration password before sending a request", async () => {
    render(<AuthProvider><AuthForm mode="register" /></AuthProvider>);
    fireEvent.change(screen.getByLabelText("Имя"), { target: { value: "Person" } });
    fireEvent.change(screen.getByLabelText("Email"), { target: { value: "person@example.com" } });
    fireEvent.change(screen.getByLabelText("Пароль"), { target: { value: "short" } });
    fireEvent.submit(screen.getByRole("button", { name: "Создать аккаунт" }).closest("form")!);

    expect(await screen.findByRole("alert")).toHaveTextContent("не менее 8 символов");
    expect(fetch).toHaveBeenCalledTimes(1);
  });

  it("shows a duplicate-email error returned by the API", async () => {
    vi.mocked(fetch).mockResolvedValueOnce(jsonResponse({ error: "email already used" }, 409));
    render(<AuthProvider><AuthForm mode="register" /></AuthProvider>);
    fireEvent.change(screen.getByLabelText("Имя"), { target: { value: "Person" } });
    fireEvent.change(screen.getByLabelText("Email"), { target: { value: "person@example.com" } });
    fireEvent.change(screen.getByLabelText("Пароль"), { target: { value: "password" } });
    fireEvent.submit(screen.getByRole("button", { name: "Создать аккаунт" }).closest("form")!);

    expect(await screen.findByRole("alert")).toHaveTextContent("уже зарегистрирован");
  });
});
