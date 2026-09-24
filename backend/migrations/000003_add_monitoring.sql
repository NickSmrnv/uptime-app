ALTER TABLE monitors
    ADD COLUMN config_version bigint NOT NULL DEFAULT 1,
    ADD COLUMN history_version bigint NOT NULL DEFAULT 1,
    ADD COLUMN next_check_at timestamptz NOT NULL DEFAULT now(),
    ADD COLUMN lease_until timestamptz,
    ADD COLUMN attempt_id uuid,
    ADD COLUMN last_checked_at timestamptz,
    ADD COLUMN last_status_code integer,
    ADD COLUMN last_error varchar(32) NOT NULL DEFAULT '',
    ADD COLUMN last_duration_ms bigint;

CREATE INDEX idx_monitors_due ON monitors (next_check_at);

CREATE TABLE monitor_minutes (
    monitor_id uuid NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    history_version bigint NOT NULL,
    minute timestamptz NOT NULL,
    successes bigint NOT NULL DEFAULT 0 CHECK (successes >= 0),
    failures bigint NOT NULL DEFAULT 0 CHECK (failures >= 0),
    PRIMARY KEY (monitor_id, history_version, minute)
);

CREATE INDEX idx_monitor_minutes_retention ON monitor_minutes (minute);
