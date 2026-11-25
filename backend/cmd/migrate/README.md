# Migration Command

The migrate command reads `MYSQL_*` configuration and executes embedded goose
migrations. It never runs automatically from API, Realtime/Runtime or Worker
startup.

```text
go run ./cmd/migrate --command=status
go run ./cmd/migrate --command=up
go run ./cmd/migrate --command=down
go run ./cmd/migrate --command=down --version=3
```

Production migration execution requires a dedicated `mv_migrator` account and a
reviewed shadow-database run.
