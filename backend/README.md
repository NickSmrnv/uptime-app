# Backend

Minimal Go project structure for the uptime service backend.

## Layout

- `cmd/api` — application entry point.
- `internal/app` — application composition.
- `internal/config` — configuration loading.
- `internal/handler` — transport handlers.
- `internal/repository` — data-access implementations.
- `internal/service` — application services.
- `internal/server` — server setup.
- `pkg` — reusable exported packages.

The project currently provides email/password authentication backed by PostgreSQL and GORM.

## Authentication configuration

The API requires the following environment variables before it can start:

- `DATABASE_URL` — PostgreSQL connection string.
- `JWT_SECRET` — secret for HS256 signing, at least 32 bytes long.

Optional settings:

- `HTTP_ADDR` — listen address; defaults to `:8080`.
- `JWT_ISSUER` — JWT issuer; defaults to `uptime-api`.
- `ACCESS_TOKEN_TTL` — access-token duration; defaults to `24h`.
- `REFRESH_TOKEN_TTL` — refresh-session duration; defaults to `720h` (30 days).
- `COOKIE_SECURE` — refresh-cookie `Secure` flag; defaults to `true` and must remain enabled when `APP_ENV=production`.
- `CORS_ALLOWED_ORIGIN` — exact frontend origin allowed to make credentialed `/auth/*` requests; defaults to `http://localhost:3000` in development and is required in production.
- `UPLOAD_STORAGE_DIR` — local directory for all uploaded files; defaults to `uploads` relative to the backend working directory. This directory is excluded from Git and served at `/uploads/{category}/{filename}`. Avatars are stored in the `avatars` category.
- `DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`, `DB_CONN_MAX_LIFETIME` — database pool settings.

The refresh token is stored only in an `HttpOnly`, `Secure`, `SameSite=Strict` cookie. For local HTTP development set `COOKIE_SECURE=false` and leave `CORS_ALLOWED_ORIGIN=http://localhost:3000`. To run the optional PostgreSQL integration test, set `TEST_DATABASE_URL` to a disposable database; the test drops and recreates its authentication tables.

## File uploads

`POST /uploads` accepts an authenticated `multipart/form-data` request with one `file` field up to 10 MB and returns its key and `/uploads/files/{filename}` URL. Files are stored locally under `uploads/files` and served as downloads. Profile avatars are a specialized JPEG/PNG upload flow stored in `uploads/avatars` and served inline.
