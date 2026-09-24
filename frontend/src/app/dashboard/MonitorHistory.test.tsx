import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import MonitorHistory, { MonitorStatus } from "./MonitorHistory";
import type { Monitor, MonitorStats } from "../../lib/api";

const apiFetch = vi.fn();
vi.mock("../../auth/AuthProvider", () => ({ useAuth: () => ({ apiFetch }) }));

const monitor: Monitor = {
  id: "point", url: "https://example.com", intervalSeconds: 5, createdAt: "2026-09-23T00:00:00Z",
  status: "up", configVersion: 1, historyVersion: 1, nextCheckAt: "2026-09-23T12:00:05Z",
  lastCheckedAt: "2026-09-23T12:00:00Z", lastStatusCode: 200, lastError: "", lastDurationMs: 40,
};
const stats: MonitorStats = {
  period: "24h", from: "2026-09-22T12:00:00Z", to: "2026-09-23T12:00:00Z", bucketSeconds: 300,
  historyVersion: 1, successes: 9, failures: 3, successPercent: 75, failurePercent: 25,
  buckets: [
    { start: "2026-09-23T11:55:00Z", successes: 0, failures: 0, successPercent: null, failurePercent: null },
    { start: "2026-09-23T12:00:00Z", successes: 9, failures: 3, successPercent: 75, failurePercent: 25 },
  ],
};
let intersect: (entries: { isIntersecting: boolean }[]) => void;

describe("MonitorHistory", () => {
  beforeEach(() => {
    apiFetch.mockResolvedValue(stats);
    vi.stubGlobal("IntersectionObserver", class {
      constructor(callback: typeof intersect) { intersect = callback; }
      observe() { intersect([{ isIntersecting: true }]); }
      disconnect() {}
    });
  });
  afterEach(() => { apiFetch.mockReset(); vi.unstubAllGlobals(); vi.useRealTimers(); });

  it("shows percentages, empty intervals, keyboard detail and the last result", async () => {
    render(<MonitorHistory monitor={monitor} />);
    expect(await screen.findByText("75%")).toBeInTheDocument();
    const chart = screen.getByRole("group", { name: /График успешных/ });
    fireEvent.keyDown(chart, { key: "ArrowRight" });
    expect(screen.getByText(/— Нет данных/)).toBeInTheDocument();
    fireEvent.keyDown(chart, { key: "ArrowRight" });
    expect(screen.getByText(/Успешно: 9 \(75%\), ошибки: 3 \(25%\)/)).toBeInTheDocument();
    expect(screen.getByText(/HTTP 200 · 40 мс/)).toBeInTheDocument();
  });

  it("changes periods and does not apply an old URL response", async () => {
    let resolveOld: (value: MonitorStats) => void = () => {};
    apiFetch.mockImplementationOnce(() => new Promise<MonitorStats>((resolve) => { resolveOld = resolve; }));
    const { rerender } = render(<MonitorHistory monitor={monitor} />);
    apiFetch.mockResolvedValue({ ...stats, historyVersion: 2, successPercent: 100 });
    rerender(<MonitorHistory monitor={{ ...monitor, configVersion: 2, historyVersion: 2 }} />);
    expect(await screen.findByText("100%")).toBeInTheDocument();
    await act(async () => resolveOld(stats));
    expect(screen.queryByText("75%")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "7 дней" }));
    await waitFor(() => expect(apiFetch).toHaveBeenLastCalledWith("/monitors/point/stats?period=7d", expect.anything()));
  });

  it("pauses offscreen and hidden-tab polling, then refreshes on return", async () => {
    vi.useFakeTimers();
    render(<MonitorHistory monitor={monitor} />);
    await act(async () => {});
    expect(apiFetch).toHaveBeenCalledTimes(1);
    await act(async () => intersect([{ isIntersecting: false }]));
    await act(async () => vi.advanceTimersByTime(30_000));
    expect(apiFetch).toHaveBeenCalledTimes(1);
    await act(async () => intersect([{ isIntersecting: true }]));
    expect(apiFetch).toHaveBeenCalledTimes(2);
    const hidden = vi.spyOn(document, "hidden", "get").mockReturnValue(true);
    fireEvent(document, new Event("visibilitychange"));
    await act(async () => vi.advanceTimersByTime(30_000));
    expect(apiFetch).toHaveBeenCalledTimes(2);
    hidden.mockReturnValue(false);
    await act(async () => { fireEvent(document, new Event("visibilitychange")); });
    expect(apiFetch).toHaveBeenCalledTimes(3);
    hidden.mockRestore();
  });

  it("shows fetch errors and distinct monitor states", async () => {
    apiFetch.mockRejectedValue(new Error("offline"));
    const { rerender } = render(<><MonitorStatus monitor={monitor} /><MonitorHistory monitor={monitor} /></>);
    expect(await screen.findByText(/Не удалось обновить график/)).toBeInTheDocument();
    expect(screen.getByText("Работает")).toBeInTheDocument();
    rerender(<MonitorStatus monitor={{ ...monitor, status: "stale" }} />);
    expect(screen.getByText("Данные устарели")).toBeInTheDocument();
  });
});
