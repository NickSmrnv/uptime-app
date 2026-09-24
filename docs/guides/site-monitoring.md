# Site monitoring

The API starts its background runner after database migrations. Creating or editing a monitor makes it immediately eligible; the runner checks for work every second. Restart the backend after upgrading to apply migration `000003_add_monitoring.sql` and start checks for existing monitors.

## Configuration and capacity

`MONITOR_CONCURRENCY` defaults to `20` (accepted range: 1–1000). Set it in the backend environment to size the worker pool. The pool is per API process; PostgreSQL leases coordinate multiple processes. No additional service or queue is required. PostgreSQL 13 or later is required for `gen_random_uuid()`.

Only HTTP GET responses with status 200 succeed. Redirects are not followed. The deadline is `min(10 seconds, interval × 0.8)` and covers DNS, connection, TLS and response headers. Bodies are closed without being downloaded or stored. The probe measures response-header latency, not full-page loading. No HTTP retry is made; connection establishment can try another validated IP within the same deadline. Internal/special-use IPv4 and IPv6 addresses, including addresses returned by DNS, are blocked. Proxy environment variables are not used.

The process emits a `monitoring_metrics` log every minute: active checks, completed observations, claim/write errors and maximum scheduling delay in milliseconds. The counters and maximum cover the preceding reporting interval. Cleanup failures are logged separately. A database outage or process shutdown creates missing observations, not a failed-site sample.

Capacity depends on latency, not only the number of monitors. A local controlled load scenario with 100 monitors at five-second intervals and 20 workers measured about 4 seconds of initial scheduling delay at 20 ms probe latency, and 16 seconds at 3 seconds probe latency. All 100 were processed without storage errors. For 100 slow monitors at five-second intervals, start with `MONITOR_CONCURRENCY=100`, then size using observed latency and scheduling delay. Missed slots are skipped; increasing concurrency never backfills observations.

## Storage and history

`monitors` holds the persistent schedule, 30-second lease, attempt UUID, configuration/history versions and the latest observation. Completion fences by monitor ID, configuration version, attempt UUID and unexpired lease, then updates the next slot and minute counters in the same transaction. Duplicate completion cannot count twice. Expired leases can be reclaimed after a restart. Editing invalidates an in-flight result; deleting cascades its history.

`monitor_minutes` stores successful/failed counts for each UTC minute and URL-history version. Changing URL starts a new graph; changing only the interval keeps history. Old versions expire with the same retention window. There are at most approximately 43,200 minute buckets per continuously monitored URL version over 30 days; URL changes can temporarily retain multiple versions. Aggregation reduces rows, not the number of writes or WAL volume. PostgreSQL autovacuum should remain enabled.

Retention runs on startup and hourly in batches of 5,000. API queries exclude samples older than 30 days even before deletion. Whole-minute storage cannot split the oldest partial minute, so API ranges round their lower bound up to the next minute. Empty intervals have null percentages. Success percentages count observations; they are not time-weighted uptime and do not imply availability between infrequent checks.

## API and dashboard

Authenticated monitor responses include `status` (`pending`, `up`, `down`, `stale`), `configVersion`, `historyVersion`, `nextCheckAt`, `lastCheckedAt`, `lastStatusCode`, `lastError` and `lastDurationMs`. Missing result fields are null. A monitor is stale when its next scheduled check is overdue by more than its timeout plus five seconds.

`GET /monitors/{id}/stats?period=1h|24h|7d|30d` returns the active URL history, weighted totals and time buckets, including gaps. Bucket widths are respectively one minute, five minutes, one hour and six hours; SQL aggregates counts before returning them. Missing or foreign monitors return 404. Error categories are `timeout`, `dns`, `tls`, `connection` and `blocked_address`.

Dashboard statuses poll every five seconds; visible charts poll every 30 seconds. Hidden tabs stop refreshing. Chart hover, touch and arrow keys expose bucket details. The initial chart period is 24 hours.

## Verification

Run from `backend/` against a **disposable** PostgreSQL database; the existing migration test drops application tables:

```sh
TEST_DATABASE_URL='postgres://…/disposable_test' go test -race ./...
go vet ./...
MONITOR_LOAD_TEST=1 TEST_DATABASE_URL='postgres://…/disposable_test' go test ./internal/monitoring -run '^TestMonitorLoad$' -count=1 -v
```

The load test uses the real PostgreSQL scheduler and synthetic 20 ms/3 s probes, without external requests. Run it separately from the suite that drops tables. Frontend checks are `npm test`, `npm run lint` and `npm run build`. If Turbopack cannot start its local worker in a restricted environment, `npx next build --webpack` checks a production build with Webpack.
