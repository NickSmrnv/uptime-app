"use client";

import { useEffect, useRef, useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "../../auth/AuthProvider";
import { ApiError, avatarSource, type Monitor } from "../../lib/api";

import MonitorHistory, { MonitorStatus } from "./MonitorHistory";

const navigationItems = [
  { label: "Обзор", current: true, href: "/dashboard", unavailable: false },
  { label: "Профиль", current: false, href: "/profile", unavailable: false },
  { label: "Мониторы", current: false, href: "#monitors", unavailable: false },
  { label: "Инциденты", current: false, href: "#", unavailable: true },
];

const intervalMultipliers = {
  seconds: 1,
  minutes: 60,
  hours: 60 * 60,
} as const;

const minMonitorIntervalSeconds = 5;
const maxMonitorIntervalSeconds = 7 * 24 * 60 * 60;

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

function intervalFormValue(seconds: number): { value: string; unit: IntervalUnit } {
  if (seconds % 3600 === 0) {
    return { value: String(seconds / 3600), unit: "hours" };
  }
  if (seconds % 60 === 0) {
    return { value: String(seconds / 60), unit: "minutes" };
  }
  return { value: String(seconds), unit: "seconds" };
}

export default function DashboardPage() {
  const router = useRouter();
  const { apiFetch, status, user, logout } = useAuth();
  const [isLoggingOut, setIsLoggingOut] = useState(false);
  const [monitors, setMonitors] = useState<Monitor[]>([]);
  const [isLoadingMonitors, setIsLoadingMonitors] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [monitorFormMode, setMonitorFormMode] = useState<"create" | "edit" | null>(null);
  const [editingMonitorId, setEditingMonitorId] = useState<string | null>(null);
  const [url, setURL] = useState("");
  const [intervalValue, setIntervalValue] = useState("1");
  const [intervalUnit, setIntervalUnit] = useState<IntervalUnit>("minutes");
  const [createError, setCreateError] = useState<string | null>(null);
  const [isCreating, setIsCreating] = useState(false);
  const [deletingMonitorId, setDeletingMonitorId] = useState<string | null>(null);
  const [pendingDeletionId, setPendingDeletionId] = useState<string | null>(null);
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const monitorRequestVersion = useRef(0);

  useEffect(() => {
    if (status === "anonymous") {
      router.replace("/login");
    }
  }, [router, status]);

  useEffect(() => {
    if (status !== "authenticated") {
      return;
    }
    let isActive = true;
    let inFlight = false;
    const refresh = async () => {
      if (document.hidden || inFlight) return;
      inFlight = true;
      const requestVersion = ++monitorRequestVersion.current;
      try {
        const loadedMonitors = await apiFetch<Monitor[]>("/monitors");
        if (!isActive || requestVersion !== monitorRequestVersion.current) return;
        setMonitors(loadedMonitors);
        setLoadError(null);
      } catch {
        if (isActive && requestVersion === monitorRequestVersion.current) {
          setLoadError("Не удалось загрузить точки мониторинга. Обновите страницу и попробуйте снова.");
        }
      } finally {
        inFlight = false;
        if (isActive && requestVersion === monitorRequestVersion.current) setIsLoadingMonitors(false);
      }
    };
    void refresh();
    const timer = window.setInterval(() => void refresh(), 5_000);
    const onVisibility = () => { if (!document.hidden) void refresh(); };
    document.addEventListener("visibilitychange", onVisibility);
    return () => {
      isActive = false;
      window.clearInterval(timer);
      document.removeEventListener("visibilitychange", onVisibility);
    };
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
    setEditingMonitorId(null);
    setURL("");
    setIntervalValue("1");
    setIntervalUnit("minutes");
    setMonitorFormMode("create");
  };

  const openEditForm = (monitor: Monitor) => {
    const interval = intervalFormValue(monitor.intervalSeconds);
    setCreateError(null);
    setEditingMonitorId(monitor.id);
    setURL(monitor.url);
    setIntervalValue(interval.value);
    setIntervalUnit(interval.unit);
    setMonitorFormMode("edit");
  };

  const closeMonitorForm = () => {
    if (isCreating) return;
    setMonitorFormMode(null);
    setEditingMonitorId(null);
    setCreateError(null);
  };

  const handleCreate = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const trimmedURL = url.trim();
    const interval = Number(intervalValue) * intervalMultipliers[intervalUnit];
    setCreateError(null);

    try {
      const parsedURL = new URL(trimmedURL);
      if ((parsedURL.protocol !== "http:" && parsedURL.protocol !== "https:") || !Number.isInteger(interval) || interval < minMonitorIntervalSeconds || interval > maxMonitorIntervalSeconds) {
        throw new Error("invalid input");
      }
    } catch {
      setCreateError("Укажите корректный адрес с http:// или https:// и интервал от 5 секунд до 7 дней.");
      return;
    }

    const isEditing = monitorFormMode === "edit" && editingMonitorId !== null;
    setIsCreating(true);
    try {
      const path = isEditing ? `/monitors/${editingMonitorId}` : "/monitors";
      const monitor = await apiFetch<Monitor>(path, {
        method: isEditing ? "PATCH" : "POST",
        body: JSON.stringify({ url: trimmedURL, intervalSeconds: interval }),
      });
      monitorRequestVersion.current += 1;
      setIsLoadingMonitors(false);
      setMonitors((current) => isEditing ? current.map((item) => item.id === monitor.id ? monitor : item) : [monitor, ...current]);
      setURL("");
      setIntervalValue("1");
      setIntervalUnit("minutes");
      setMonitorFormMode(null);
      setEditingMonitorId(null);
    } catch (error) {
      setCreateError(error instanceof ApiError && error.status === 400 ? "Проверьте адрес сайта и интервал." : error instanceof ApiError && error.status === 404 ? "Сайт больше не найден. Обновите список и попробуйте снова." : error instanceof ApiError && error.status === 422 ? "Достигнут лимит точек мониторинга для аккаунта." : isEditing ? "Не удалось сохранить изменения. Повторите попытку." : "Не удалось создать точку мониторинга. Повторите попытку.");
    } finally {
      setIsCreating(false);
    }
  };

  const handleDelete = async (monitorID: string) => {
    setDeletingMonitorId(monitorID);
    setDeleteError(null);
    try {
      await apiFetch<void>(`/monitors/${monitorID}`, { method: "DELETE" });
      monitorRequestVersion.current += 1;
      setMonitors((current) => current.filter((monitor) => monitor.id !== monitorID));
      setPendingDeletionId(null);
    } catch (error) {
      setDeleteError(error instanceof ApiError && error.status === 404 ? "Сайт уже удалён или недоступен." : "Не удалось удалить сайт. Повторите попытку.");
    } finally {
      setDeletingMonitorId(null);
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
              <p className="mt-3 max-w-2xl text-slate-600">Проверки доступности, состояние сервисов и история за последние 30 дней.</p>

              <section className="mt-8" id="monitors" aria-labelledby="monitors-heading">
                <div className="flex flex-col justify-between gap-5 sm:flex-row sm:items-center">
                  <div>
                    <h2 className="text-xl font-semibold" id="monitors-heading">Точки мониторинга</h2>
                    <p className="mt-1 text-sm text-slate-600">Добавьте сайт — проверки начнутся автоматически с заданным интервалом.</p>
                  </div>
                  <button className="rounded-lg bg-blue-600 px-4 py-2.5 text-sm font-medium text-white transition hover:bg-blue-700" onClick={openCreateForm} type="button">
                    Добавить сайт
                  </button>
                </div>

                {monitorFormMode ? (
                  <section aria-labelledby="create-monitor-heading" className="mt-6 rounded-2xl border border-blue-100 bg-white p-6 shadow-sm sm:p-8">
                    <div className="flex items-start justify-between gap-4">
                      <div>
                        <h3 className="text-lg font-semibold" id="create-monitor-heading">{monitorFormMode === "edit" ? "Редактирование точки мониторинга" : "Новая точка мониторинга"}</h3>
                        <p className="mt-1 text-sm text-slate-600">После сохранения сразу начнётся новая проверка. Успешным считается ответ HTTP 200.</p>
                      </div>
                      <button aria-label="Закрыть форму" className="text-sm font-medium text-slate-500 hover:text-slate-900" onClick={closeMonitorForm} type="button">Закрыть</button>
                    </div>
                    <form className="mt-6 grid gap-5 sm:grid-cols-[minmax(0,1fr)_auto_auto] sm:items-end" noValidate onSubmit={handleCreate}>
                      <label className="block text-sm font-medium text-slate-800" htmlFor="monitor-url">
                        Адрес сайта
                        <input autoComplete="url" className="mt-2 w-full rounded-lg border border-slate-300 px-3 py-2.5 outline-none ring-blue-600 focus:ring-2" id="monitor-url" name="url" onChange={(event) => setURL(event.target.value)} placeholder="https://example.com" required type="url" value={url} />
                      </label>
                      <label className="block text-sm font-medium text-slate-800" htmlFor="monitor-interval">
                        Интервал
                        <input className="mt-2 w-full rounded-lg border border-slate-300 px-3 py-2.5 outline-none ring-blue-600 focus:ring-2 sm:w-28" id="monitor-interval" min={intervalUnit === "seconds" ? minMonitorIntervalSeconds : 1} name="interval" onChange={(event) => setIntervalValue(event.target.value)} required step="1" type="number" value={intervalValue} />
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
                        {isCreating ? (monitorFormMode === "edit" ? "Сохраняем…" : "Создаём…") : (monitorFormMode === "edit" ? "Сохранить" : "Создать")}
                      </button>
                    </form>
                  </section>
                ) : null}

                {isLoadingMonitors ? <p className="mt-8 text-sm text-slate-500">Загружаем точки мониторинга…</p> : null}
                {loadError ? <p className="mt-8 text-sm text-red-700" role="alert">{loadError}</p> : null}
                {!isLoadingMonitors && !loadError && monitors.length === 0 ? (
                  <div className="mt-8 rounded-2xl border border-dashed border-slate-300 bg-white p-6 sm:p-8">
                    <h3 className="text-lg font-semibold">Мониторов пока нет</h3>
                    <p className="mt-1 text-sm text-slate-600">Добавьте первый адрес, чтобы запустить проверки доступности.</p>
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
                          <div className="flex flex-wrap items-center gap-2">
                            <MonitorStatus monitor={monitor} />
                            <button className="rounded-lg border border-slate-300 px-3 py-1.5 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60" disabled={deletingMonitorId !== null || isCreating} onClick={() => openEditForm(monitor)} type="button">Редактировать</button>
                            <button className="rounded-lg border border-red-200 px-3 py-1.5 text-sm font-medium text-red-700 hover:bg-red-50 disabled:cursor-not-allowed disabled:opacity-60" disabled={deletingMonitorId !== null || isCreating} onClick={() => { setDeleteError(null); setPendingDeletionId(monitor.id); }} type="button">Удалить</button>
                          </div>
                        </div>
                        <MonitorHistory monitor={monitor} />
                        {pendingDeletionId === monitor.id ? (
                          <div className="mt-4 rounded-lg bg-red-50 p-3 text-sm text-red-900" role="alert">
                            <p>Удалить этот сайт?</p>
                            <div className="mt-3 flex gap-2">
                              <button className="rounded-lg bg-red-600 px-3 py-1.5 font-medium text-white hover:bg-red-700 disabled:cursor-not-allowed disabled:opacity-60" disabled={deletingMonitorId === monitor.id} onClick={() => void handleDelete(monitor.id)} type="button">{deletingMonitorId === monitor.id ? "Удаляем…" : "Да, удалить"}</button>
                              <button className="rounded-lg border border-red-200 px-3 py-1.5 font-medium text-red-700 hover:bg-white disabled:cursor-not-allowed disabled:opacity-60" disabled={deletingMonitorId === monitor.id} onClick={() => setPendingDeletionId(null)} type="button">Отмена</button>
                            </div>
                          </div>
                        ) : null}
                        {deleteError && pendingDeletionId === monitor.id ? <p className="mt-3 text-sm text-red-700" role="alert">{deleteError}</p> : null}
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
