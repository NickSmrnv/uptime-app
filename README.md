# Uptime

## Local development

1. Ensure PostgreSQL is running and create a database for the application.
2. Copy `.env.example` to `.env`, then set `DATABASE_URL` and a random `JWT_SECRET` of at least 32 bytes.
3. Install frontend dependencies once with `cd frontend && npm install`.
4. Run both services from the project root with `./dev.sh`.

The frontend is available at [http://localhost:3000](http://localhost:3000) and the API defaults to [http://localhost:8080](http://localhost:8080). Press `Ctrl+C` to stop both processes.
