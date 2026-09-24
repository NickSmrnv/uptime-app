"use client";

import { useEffect, useId, useRef, useState, type PointerEvent } from "react";
import { useAuth } from "../../auth/AuthProvider";
import type { Monitor, MonitorPeriod, MonitorStats } from "../../lib/api";

const periods: { value: MonitorPeriod; label: string }[] = [
  { value: "1h", label: "1 час" }, { value: "24h", label: "24 часа" },
  { value: "7d", label: "7 дней" }, { value: "30d", label: "30 дней" },
];

const states = {
  pending: { label: "Ожидает проверки", color: "bg-slate-100 text-slate-600" },
  up: { label: "Работает", color: "bg-emerald-50 text-emerald-700" },
  down: { label: "Ошибка", color: "bg-red-50 text-red-700" },
  stale: { label: "Данные устарели", color: "bg-amber-50 text-amber-800" },
};

const errors: Record<string, string> = {
  timeout: "Превышено время ожидания", dns: "Ошибка DNS", tls: "Ошибка TLS",
  connection: "Ошибка соединения", blocked_address: "Адрес недоступен для публичного мониторинга",
};

function dateLabel(value: string): string {
  return new Date(value).toLocaleString("ru-RU", { day: "2-digit", month: "2-digit", hour: "2-digit", minute: "2-digit" });
}

function percent(value: number | null): string {
  return value === null ? "Нет данных" : `${value.toLocaleString("ru-RU", { maximumFractionDigits: 2 })}%`;
}

export function MonitorStatus({ monitor }: { monitor: Monitor }) {
  const state = states[monitor.status] ?? states.pending;
  return <span className={`w-fit rounded-full px-3 py-1 text-xs font-medium ${state.color}`}>{state.label}</span>;
}

export default function MonitorHistory({ monitor }: { monitor: Monitor }) {
  const { apiFetch } = useAuth();
  const root = useRef<HTMLDivElement>(null);
  const tooltipID = useId();
  const [period, setPeriod] = useState<MonitorPeriod>("24h");
  const [response, setResponse] = useState<{ key: string; data: MonitorStats } | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [active, setActive] = useState<number | null>(null);
  const requestKey = `${monitor.id}:${monitor.configVersion}:${monitor.historyVersion}:${period}`;
  const stats = response?.key === requestKey ? response.data : null;

  useEffect(() => {
    const element = root.current;
    if (!element) return;
    let alive = true;
    let visible = false;
    let controller: AbortController | null = null;
    let sequence = 0;
    const refresh = async () => {
      if (!visible || document.hidden || controller) return;
      const current = ++sequence;
      controller = new AbortController();
      try {
        const data = await apiFetch<MonitorStats>(`/monitors/${monitor.id}/stats?period=${period}`, { signal: controller.signal });
        if (!alive || current !== sequence) return;
        if (data.historyVersion !== monitor.historyVersion) return;
        setResponse({ key: requestKey, data });
        setError(null);
      } catch {
        if (alive && current === sequence) setError("Не удалось обновить график. Повторим автоматически.");
      } finally {
        if (current === sequence) controller = null;
      }
    };
    const suspend = () => {
      sequence += 1;
      controller?.abort();
      controller = null;
    };
    const observer = new IntersectionObserver(([entry]) => {
      visible = entry.isIntersecting;
      if (visible) void refresh();
      else suspend();
    });
    observer.observe(element);
    const onVisibility = () => {
      if (document.hidden) suspend();
      else void refresh();
    };
    document.addEventListener("visibilitychange", onVisibility);
    const timer = window.setInterval(() => void refresh(), 30_000);
    return () => {
      alive = false;
      suspend();
      observer.disconnect();
      window.clearInterval(timer);
      document.removeEventListener("visibilitychange", onVisibility);
    };
  }, [apiFetch, monitor.id, monitor.historyVersion, period, requestKey]);

  const bucket = active === null ? null : stats?.buckets[active];
  const selectBucket = (event: PointerEvent<HTMLDivElement>) => {
    if (!stats) return;
    const bounds = event.currentTarget.getBoundingClientRect();
    setActive(Math.max(0, Math.min(stats.buckets.length - 1, Math.floor((event.clientX - bounds.left) / bounds.width * stats.buckets.length))));
  };
  return (
    <div className="mt-4 border-t border-slate-100 pt-4" ref={root}>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-sm text-slate-600">Успешные проверки: <strong className="text-slate-900">{stats ? percent(stats.successPercent) : "—"}</strong></p>
        <div aria-label={`Период графика ${monitor.url}`} className="flex flex-wrap gap-1">
          {periods.map((item) => (
            <button aria-pressed={period === item.value} className={`rounded-md px-2 py-1 text-xs focus-visible:outline-2 focus-visible:outline-blue-600 ${period === item.value ? "bg-blue-50 text-blue-700" : "text-slate-500 hover:bg-slate-50"}`} key={item.value} onClick={() => { setPeriod(item.value); setActive(null); setError(null); }} type="button">{item.label}</button>
          ))}
        </div>
      </div>
      {stats ? (
        <>
          <div
            aria-label={`График успешных проверок ${monitor.url}. Стрелки влево и вправо выбирают интервал.`}
            aria-describedby={bucket ? tooltipID : undefined}
            className="relative mt-3 rounded outline-offset-4 focus-visible:outline-2 focus-visible:outline-blue-600"
            role="group"
            tabIndex={0}
            onKeyDown={(event) => {
              if (event.key !== "ArrowLeft" && event.key !== "ArrowRight") return;
              event.preventDefault();
              const direction = event.key === "ArrowLeft" ? -1 : 1;
              setActive((value) => Math.max(0, Math.min(stats.buckets.length - 1, (value ?? -1) + direction)));
            }}
            onPointerMove={selectBucket}
            onPointerDown={selectBucket}
            onPointerLeave={() => setActive(null)}
            onBlur={() => setActive(null)}
          >
            {bucket && <div className="pointer-events-none absolute bottom-full left-0 z-10 mb-2 max-w-full rounded-lg bg-slate-900 px-3 py-2 text-xs leading-5 text-white shadow-lg" id={tooltipID} role="tooltip">
              {`${dateLabel(bucket.start)} — ${bucket.successPercent === null ? "Нет данных" : `Успешно: ${bucket.successes} (${percent(bucket.successPercent)}), ошибки: ${bucket.failures} (${percent(bucket.failurePercent)})`}`}
            </div>}
            <svg aria-hidden="true" className="h-16 w-full" preserveAspectRatio="none" viewBox={`0 0 ${stats.buckets.length * 4} 100`}>
              {stats.buckets.map((point, index) => (
                <g key={point.start} opacity={active === null || active === index ? 1 : 0.6}>
                  <rect x={index * 4} y={0} width={3} height={100} fill={point.successPercent === null ? "#e2e8f0" : "#ef4444"} rx={0.5} />
                  {point.successPercent !== null && <rect x={index * 4} y={100 - point.successPercent} width={3} height={point.successPercent} fill="#10b981" rx={0.5} />}
                </g>
              ))}
            </svg>
          </div>
          <div className="mt-1 flex justify-between text-[11px] text-slate-500"><span>{dateLabel(stats.from)}</span><span>{dateLabel(stats.to)}</span></div>
          <p className="mt-2 min-h-10 text-xs leading-5 text-slate-500">Зелёный — успешно · красный — ошибка · серый — нет данных. Выберите столбец для подробностей.</p>
        </>
      ) : <p className="py-6 text-sm text-slate-500">{error ? "График недоступен" : "Загружаем историю проверок…"}</p>}
      {error && <p className="mb-2 text-xs text-amber-700" role="status">{error}</p>}
      <p className="text-xs leading-5 text-slate-500">
        {monitor.lastCheckedAt ? `Последняя проверка: ${dateLabel(monitor.lastCheckedAt)} · ${monitor.lastError ? errors[monitor.lastError] ?? "Ошибка проверки" : `HTTP ${monitor.lastStatusCode}`} · ${monitor.lastDurationMs ?? 0} мс` : "Первая проверка запустится автоматически."}
      </p>
    </div>
  );
}
