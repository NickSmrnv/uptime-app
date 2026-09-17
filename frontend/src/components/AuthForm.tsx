"use client";

import { useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { ApiError } from "../lib/api";
import { useAuth } from "../auth/AuthProvider";

type AuthFormProps = {
  mode: "login" | "register";
};

function errorMessage(error: unknown, mode: AuthFormProps["mode"]): string {
  if (error instanceof ApiError) {
    if (error.status === 409) return "Этот email уже зарегистрирован.";
    if (error.status === 401) return "Неверный email или пароль.";
    if (error.status === 400) return "Проверьте email и пароль.";
  }

  return mode === "login" ? "Не удалось выполнить вход. Повторите попытку." : "Не удалось создать аккаунт. Повторите попытку.";
}

export function AuthForm({ mode }: AuthFormProps) {
  const router = useRouter();
  const { login, register } = useAuth();
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const isRegister = mode === "register";

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);

    if (isRegister && name.trim().length === 0) {
      setError("Укажите имя.");
      return;
    }
    if (isRegister && password.length < 8) {
      setError("Пароль должен содержать не менее 8 символов.");
      return;
    }

    setIsSubmitting(true);
    try {
      if (isRegister) {
        await register(name.trim(), email.trim(), password);
      } else {
        await login(email.trim(), password);
      }
      router.replace("/dashboard");
    } catch (requestError) {
      setError(errorMessage(requestError, mode));
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center bg-zinc-50 px-4 text-zinc-900">
      <section className="w-full max-w-md rounded-2xl bg-white p-8 shadow-sm ring-1 ring-zinc-200">
        <h1 className="text-2xl font-semibold">{isRegister ? "Создать аккаунт" : "Войти"}</h1>
        <p className="mt-2 text-sm text-zinc-600">
          {isRegister ? "Зарегистрируйтесь, чтобы продолжить работу с сервисом." : "Введите данные вашего аккаунта."}
        </p>
        <form className="mt-6 space-y-4" onSubmit={handleSubmit} noValidate>
          {isRegister ? <>
            <label className="block text-sm font-medium" htmlFor="name">
              Имя
            </label>
            <input id="name" name="name" type="text" autoComplete="name" required value={name} onChange={(event) => setName(event.target.value)} className="mt-1 w-full rounded-lg border border-zinc-300 px-3 py-2 outline-none ring-blue-600 focus:ring-2" />
          </> : null}
          <label className="block text-sm font-medium" htmlFor="email">
            Email
          </label>
          <input id="email" name="email" type="email" autoComplete="email" required value={email} onChange={(event) => setEmail(event.target.value)} className="mt-1 w-full rounded-lg border border-zinc-300 px-3 py-2 outline-none ring-blue-600 focus:ring-2" />
          <label className="block text-sm font-medium" htmlFor="password">
            Пароль
          </label>
          <input id="password" name="password" type="password" autoComplete={isRegister ? "new-password" : "current-password"} required minLength={isRegister ? 8 : undefined} value={password} onChange={(event) => setPassword(event.target.value)} className="mt-1 w-full rounded-lg border border-zinc-300 px-3 py-2 outline-none ring-blue-600 focus:ring-2" />
          {error ? <p role="alert" className="text-sm text-red-700">{error}</p> : null}
          <button type="submit" disabled={isSubmitting} className="w-full rounded-lg bg-blue-700 px-4 py-2 font-medium text-white disabled:cursor-not-allowed disabled:bg-blue-400">
            {isSubmitting ? "Подождите…" : isRegister ? "Создать аккаунт" : "Войти"}
          </button>
        </form>
      </section>
    </main>
  );
}
