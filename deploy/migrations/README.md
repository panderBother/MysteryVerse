# Migrations

Versioned goose migrations live in `backend/migrations` and are embedded into
`backend/cmd/migrate`.

```text
go -C backend run ./cmd/migrate --command=status
go -C backend run ./cmd/migrate --command=up
go -C backend run ./cmd/migrate --command=down --version=0
```

Production execution must use a dedicated migration account, run against a
reviewed shadow database first, and remain outside application startup.
