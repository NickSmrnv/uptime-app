# Database migrations

Migration files use the `NNNNNN_name.sql` format and are embedded into the backend binary. The API applies pending files during startup and records their versions in PostgreSQL's `schema_migrations` table.

Add a new migration with a larger number. Do not edit a migration after it has been applied to a shared database. Keep each migration compatible with the deployment order and safe to run in the single transaction used by the migration runner.
