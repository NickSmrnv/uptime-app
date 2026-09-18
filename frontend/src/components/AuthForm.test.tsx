import { fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { AuthProvider } from "../auth/AuthProvider";
import { AuthForm } from "./AuthForm";

const replace = vi.fn();

const { axiosClient, axiosRequest } = vi.hoisted(() => {
  const axiosRequest = vi.fn();
  return {
    axiosRequest,
    axiosClient: { request: axiosRequest, defaults: { headers: { common: {} as Record<string, string> } } },
  };
});

vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace }),
}));

vi.mock("axios", () => ({
  default: {
    create: vi.fn(() => axiosClient),
    isAxiosError: (error: unknown) => typeof error === "object" && error !== null && "isAxiosError" in error && error.isAxiosError === true,
  },
}));

function axiosError(status: number, message: string) {
  return { isAxiosError: true, response: { status, data: { error: message } } };
}

describe("AuthForm", () => {
  beforeEach(() => {
    process.env.NEXT_PUBLIC_API_URL = "http://api.test";
    replace.mockReset();
    axiosRequest.mockReset();
    axiosRequest.mockRejectedValueOnce(axiosError(401, "invalid refresh token"));
    axiosClient.defaults.headers.common = {};
  });

  afterEach(() => vi.clearAllMocks());

  it("validates a registration password before sending a request", async () => {
    render(<AuthProvider><AuthForm mode="register" /></AuthProvider>);
    fireEvent.change(screen.getByLabelText("Имя"), { target: { value: "Person" } });
    fireEvent.change(screen.getByLabelText("Email"), { target: { value: "person@example.com" } });
    fireEvent.change(screen.getByLabelText("Пароль"), { target: { value: "short" } });
    fireEvent.submit(screen.getByRole("button", { name: "Создать аккаунт" }).closest("form")!);

    expect(await screen.findByRole("alert")).toHaveTextContent("не менее 8 символов");
    expect(axiosRequest).toHaveBeenCalledTimes(1);
  });

  it("shows a duplicate-email error returned by the API", async () => {
    axiosRequest.mockRejectedValueOnce(axiosError(409, "email already used"));
    render(<AuthProvider><AuthForm mode="register" /></AuthProvider>);
    fireEvent.change(screen.getByLabelText("Имя"), { target: { value: "Person" } });
    fireEvent.change(screen.getByLabelText("Email"), { target: { value: "person@example.com" } });
    fireEvent.change(screen.getByLabelText("Пароль"), { target: { value: "password" } });
    fireEvent.submit(screen.getByRole("button", { name: "Создать аккаунт" }).closest("form")!);

    expect(await screen.findByRole("alert")).toHaveTextContent("уже зарегистрирован");
  });
});
