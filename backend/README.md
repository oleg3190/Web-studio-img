# Backend database foundation

## Local PostgreSQL

Start PostgreSQL with Docker Compose from the repository root:

```bash
docker compose up -d postgres
```

Connection string:

```text
postgres://webstudio:webstudio@localhost:5432/webstudio?sslmode=disable
```

## Migrations

Install golang-migrate and run:

```bash
make migrate-up
make migrate-down
```

The migration is reversible and creates the foundation tables required by the technical specification.

## sqlc

From `backend/`:

```bash
sqlc generate
```

SQL queries live in `queries/`; generated Go code is emitted to `internal/db/`.

Production object storage remains Yandex Object Storage through its S3-compatible API; PostgreSQL is the source of truth for business data.
