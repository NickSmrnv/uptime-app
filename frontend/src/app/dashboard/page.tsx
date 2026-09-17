"use client";

import { useEffect, useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "../../auth/AuthProvider";
import { ApiError, avatarSource, type Monitor } from "../../lib/api";

const navigationItems = [
  { label: "Обзор", current: true, href: "/dashboard", unavailable: false },
  { label: "Профиль", current: false, href: "/profile", unavailable: false },
  { label: "Мониторы", current: false, href: "#", unavailable: true },
  { label: "Инциденты", current: false, href: "#", unavailable: true },
];

const intervalMultipliers = {
  seconds: 1,
  minutes: 60,
  hours: 60 * 60,
} as const;

type IntervalUnit = keyof typeof intervalMultipliers;

function formatInterval(seconds: number): string {
  if (seconds % 3600 === 0) {
    const hours = seconds / 3600;
    return `Каждый ${hours === 1 ? "час" : `${hours} ч.`}`;
  }
  if (seconds % 60 === 0) {
    const minutes = seconds / 60;
    return `Каждые ${minutes} мин.`;
  }
  return `Каждые ${seconds} сек.`;
}

export default function DashboardPage() {
  const router = useRouter();
  const { apiFetch, status, user, logout } = useAuth();
  const [isLoggingOut, setIsLoggingOut] = useState(false);
  const [monitors, setMonitors] = useState<Monitor[]>([]);
  const [isLoadingMonitors, setIsLoadingMonitors] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [isCreateFormOpen, setIsCreateFormOpen] = useState(false);
  const [url, setURL] = useState("");
  const [intervalValue, setIntervalValue] = useState("1");
  const [intervalUnit, setIntervalUnit] = useState<IntervalUnit>("minutes");
  const [createError, setCreateError] = useState<string | null>(null);
  const [isCreating, setIsCreating] = useState(false);

  useEffect(() => {
    if (status === "anonymous") {
      router.replace("/login");
    }
  }, [router, status]);

  useEffect(() => {
    if (status !== "authenticated") {
      return;
    }
    void apiFetch<Monitor[]>("/monitors")
      .then((loadedMonitors) => {
        setMonitors(loadedMonitors);
        setLoadError(null);
      })
      .catch(() => setLoadError("Не удалось загрузить точки мониторинга. Обновите страницу и попробуйте снова."))
      .finally(() => setIsLoadingMonitors(false));
  }, [apiFetch, status]);

  const handleLogout = async () => {
    setIsLoggingOut(true);

    try {
      await logout();
      router.replace("/login");
    } finally {
      setIsLoggingOut(false);
    }
  };

  const openCreateForm = () => {
    setCreateError(null);
    setIsCreateFormOpen(true);
  };

  const handleCreate = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const trimmedURL = url.trim();
    const interval = Number(intervalValue) * intervalMultipliers[intervalUnit];
    setCreateError(null);

    try {
      const parsedURL = new URL(trimmedURL);
      if ((parsedURL.protocol !== "http:" && parsedURL.protocol !== "https:") || !Number.isInteger(interval) || interval < 5 || interval > 604800) {
        throw new Error("invalid input");
      }
    } catch {
      setCreateError("Укажите корректный адрес с http:// или https:// и интервал от 5 секунд до 7 дней.");
      return;
    }

    setIsCreating(true);
    try {
      const monitor = await apiFetch<Monitor>("/monitors", {
        method: "POST",
        body: JSON.stringify({ url: trimmedURL, intervalSeconds: interval }),
      });
      setMonitors((current) => [monitor, ...current]);
      setURL("");
      setIntervalValue("1");
      setIntervalUnit("minutes");
      setIsCreateFormOpen(false);
    } catch (error) {
      setCreateError(error instanceof ApiError && error.status === 400 ? "Проверьте адрес сайта и интервал." : "Не удалось создать точку мониторинга. Повторите попытку.");
    } finally {
      setIsCreating(false);
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

              <section className="mt-8" aria-labelledby="monitors-heading">
                <div className="flex flex-col justify-between gap-5 sm:flex-row sm:items-center">
                  <div>
                    <h2 className="text-xl font-semibold" id="monitors-heading">Точки мониторинга</h2>
                    <p className="mt-1 text-sm text-slate-600">Создайте адрес и задайте будущую периодичность проверки.</p>
                  </div>
                  <button className="rounded-lg bg-blue-600 px-4 py-2.5 text-sm font-medium text-white transition hover:bg-blue-700" onClick={openCreateForm} type="button">
                    Добавить сайт
                  </button>
                </div>

                {isCreateFormOpen ? (
                  <section aria-labelledby="create-monitor-heading" className="mt-6 rounded-2xl border border-blue-100 bg-white p-6 shadow-sm sm:p-8">
                    <div className="flex items-start justify-between gap-4">
                      <div>
                        <h3 className="text-lg font-semibold" id="create-monitor-heading">Новая точка мониторинга</h3>
                        <p className="mt-1 text-sm text-slate-600">Проверки ещё не запускаются — сейчас мы только сохраняем настройки.</p>
                      </div>
                      <button aria-label="Закрыть форму" className="text-sm font-medium text-slate-500 hover:text-slate-900" onClick={() => setIsCreateFormOpen(false)} type="button">Закрыть</button>
                    </div>
                    <form className="mt-6 grid gap-5 sm:grid-cols-[minmax(0,1fr)_auto_auto] sm:items-end" noValidate onSubmit={handleCreate}>
                      <label className="block text-sm font-medium text-slate-800" htmlFor="monitor-url">
                        Адрес сайта
                        <input autoComplete="url" className="mt-2 w-full rounded-lg border border-slate-300 px-3 py-2.5 outline-none ring-blue-600 focus:ring-2" id="monitor-url" name="url" onChange={(event) => setURL(event.target.value)} placeholder="https://example.com" required type="url" value={url} />
                      </label>
                      <label className="block text-sm font-medium text-slate-800" htmlFor="monitor-interval">
                        Интервал
                        <input className="mt-2 w-full rounded-lg border border-slate-300 px-3 py-2.5 outline-none ring-blue-600 focus:ring-2 sm:w-28" id="monitor-interval" min={intervalUnit === "seconds" ? 5 : 1} name="interval" onChange={(event) => setIntervalValue(event.target.value)} required step="1" type="number" value={intervalValue} />
                      </label>
                      <label className="block text-sm font-medium text-slate-800" htmlFor="monitor-interval-unit">
                        Единица
                        <select className="mt-2 w-full rounded-lg border border-slate-300 bg-white px-3 py-2.5 outline-none ring-blue-600 focus:ring-2" id="monitor-interval-unit" onChange={(event) => setIntervalUnit(event.target.value as IntervalUnit)} value={intervalUnit}>
                          <option value="seconds">секунды</option>
                          <option value="minutes">минуты</option>
                          <option value="hours">часы</option>
                        </select>
                      </label>
                      {createError ? <p className="sm:col-span-3 text-sm text-red-700" role="alert">{createError}</p> : null}
                      <button className="justify-self-start rounded-lg bg-blue-600 px-4 py-2.5 text-sm font-medium text-white disabled:cursor-not-allowed disabled:bg-blue-400 sm:col-span-3" disabled={isCreating} type="submit">
                        {isCreating ? "Создаём…" : "Создать"}
                      </button>
                    </form>
                  </section>
                ) : null}

                {isLoadingMonitors ? <p className="mt-8 text-sm text-slate-500">Загружаем точки мониторинга…</p> : null}
                {loadError ? <p className="mt-8 text-sm text-red-700" role="alert">{loadError}</p> : null}
                {!isLoadingMonitors && !loadError && monitors.length === 0 ? (
                  <div className="mt-8 rounded-2xl border border-dashed border-slate-300 bg-white p-6 sm:p-8">
                    <h3 className="text-lg font-semibold">Мониторов пока нет</h3>
                    <p className="mt-1 text-sm text-slate-600">Добавьте первый адрес, чтобы сохранить будущую проверку доступности.</p>
                  </div>
                ) : null}
                {monitors.length > 0 ? (
                  <ul className="mt-6 grid gap-4" aria-label="Созданные точки мониторинга">
                    {monitors.map((monitor) => (
                      <li className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm" key={monitor.id}>
                        <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-center">
                          <div className="min-w-0">
                            <p className="truncate font-medium text-slate-900">{monitor.url}</p>
                            <p className="mt-1 text-sm text-slate-600">{formatInterval(monitor.intervalSeconds)}</p>
                          </div>
                          <span className="w-fit rounded-full bg-slate-100 px-3 py-1 text-xs font-medium text-slate-600">Ожидает запуска</span>
                        </div>
                      </li>
                    ))}
                  </ul>
                ) : null}
              </section>
            </div>
          </section>
        </div>
      </div>
    </main>
  );
}
