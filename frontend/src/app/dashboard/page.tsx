"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "../../auth/AuthProvider";
import { avatarSource } from "../../lib/api";

const navigationItems = [
  { label: "Обзор", current: true, href: "/dashboard", unavailable: false },
  { label: "Профиль", current: false, href: "/profile", unavailable: false },
  { label: "Мониторы", current: false, href: "#", unavailable: true },
  { label: "Инциденты", current: false, href: "#", unavailable: true },
];

export default function DashboardPage() {
  const router = useRouter();
  const { status, user, logout } = useAuth();
  const [isLoggingOut, setIsLoggingOut] = useState(false);

  useEffect(() => {
    if (status === "anonymous") {
      router.replace("/login");
    }
  }, [router, status]);

  const handleLogout = async () => {
    setIsLoggingOut(true);

    try {
      await logout();
      router.replace("/login");
    } finally {
      setIsLoggingOut(false);
    }
  };

  if (status === "loading") {
    return (
      <main className="grid min-h-screen place-items-center bg-slate-50 px-4 text-sm text-slate-500">
        Восстанавливаем сессию…
      </main>
    );
  }
  if (status === "anonymous" || !user) {
    return null;
  }

  return (
    <main className="min-h-screen bg-slate-50 text-slate-950">
      <div className="mx-auto flex min-h-screen max-w-[1440px]">
        <aside className="hidden w-64 shrink-0 flex-col border-r border-slate-200 bg-white p-5 lg:flex">
          <a className="flex items-center gap-3 px-2 py-2" href="/dashboard">
            <span className="grid size-9 place-items-center rounded-xl bg-blue-600 text-lg font-bold text-white">U</span>
            <span className="text-lg font-semibold tracking-tight">Uptime</span>
          </a>

          <nav aria-label="Основная навигация" className="mt-10 space-y-1">
            {navigationItems.map((item) => (
              <a
                aria-current={item.current ? "page" : undefined}
                className={`flex w-full items-center rounded-lg px-3 py-2.5 text-left text-sm font-medium ${
                  item.current
                    ? "bg-blue-50 text-blue-700"
                    : item.unavailable ? "cursor-not-allowed text-slate-400" : "text-slate-600 hover:bg-slate-50 hover:text-slate-950"
                }`}
                href={item.href}
                key={item.label}
                onClick={item.unavailable ? (event) => event.preventDefault() : undefined}
              >
                {item.label}
                {item.unavailable && <span className="ml-auto text-xs font-normal">Скоро</span>}
              </a>
            ))}
          </nav>

          <div className="mt-auto rounded-xl bg-slate-50 p-4">
            <p className="text-sm font-medium text-slate-900">Нужна помощь?</p>
            <p className="mt-1 text-xs leading-5 text-slate-500">Настройте первый монитор, когда будете готовы.</p>
          </div>
        </aside>

        <div className="flex min-w-0 flex-1 flex-col">
          <header className="flex h-16 items-center justify-between border-b border-slate-200 bg-white px-4 sm:px-8">
            <a className="flex items-center gap-2 font-semibold lg:hidden" href="/dashboard">
              <span className="grid size-8 place-items-center rounded-lg bg-blue-600 text-sm text-white">U</span>
              Uptime
            </a>
            <p className="hidden text-sm text-slate-500 lg:block">Обзор</p>
            <div className="flex items-center gap-3">
              <div className="hidden text-right sm:block">
                <p className="max-w-56 truncate text-sm font-medium text-slate-800">{user.email}</p>
                <p className="text-xs text-slate-500">Мой аккаунт</p>
              </div>
              {user.avatarUrl ? (
                // Avatar files are served by the API and must bypass Next.js image optimization.
                // eslint-disable-next-line @next/next/no-img-element
                <img alt={`Аватар ${user.name}`} className="size-9 shrink-0 rounded-full border border-slate-200 object-cover" src={avatarSource(user.avatarUrl)} />
              ) : (
                <span aria-hidden="true" className="grid size-9 shrink-0 place-items-center rounded-full bg-blue-100 text-sm font-semibold text-blue-700">
                  {user.email.slice(0, 1).toUpperCase()}
                </span>
              )}
              <button
                className="rounded-lg border border-slate-300 px-3 py-2 text-sm font-medium text-slate-700 transition hover:border-slate-400 hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60"
                disabled={isLoggingOut}
                onClick={() => void handleLogout()}
                type="button"
              >
                {isLoggingOut ? "Выходим…" : "Выйти"}
              </button>
            </div>
          </header>

          <section className="flex-1 px-4 py-8 sm:px-8 sm:py-10">
            <div className="mx-auto max-w-5xl">
              <p className="text-sm font-medium text-blue-700">Рабочее пространство</p>
              <h1 className="mt-2 text-3xl font-semibold tracking-tight sm:text-4xl">Добро пожаловать, {user.email.split("@")[0]}</h1>
              <p className="mt-3 max-w-2xl text-slate-600">Здесь будут отображаться ваши проверки доступности и состояние сервисов.</p>

              <section className="mt-8 rounded-2xl border border-dashed border-slate-300 bg-white p-6 sm:p-8">
                <div className="flex flex-col justify-between gap-5 sm:flex-row sm:items-center">
                  <div>
                    <h2 className="text-lg font-semibold">Мониторов пока нет</h2>
                    <p className="mt-1 text-sm text-slate-600">Добавьте первый адрес, чтобы начать отслеживать его доступность.</p>
                  </div>
                  <button className="cursor-not-allowed rounded-lg bg-blue-600 px-4 py-2.5 text-sm font-medium text-white opacity-60" disabled type="button">
                    Добавить монитор
                  </button>
                </div>
              </section>
            </div>
          </section>
        </div>
      </div>
    </main>
  );
}
