# Platform Layer

The platform layer contains configuration, logging, health, HTTP, database,
Redis, error and lifecycle primitives shared by the API, Realtime/Runtime and
Worker processes.

- MySQL is authoritative for committed business facts.
- Redis is cache, presence, lease and coordination infrastructure only.
- Migrations are executed by `cmd/migrate`, never automatically at service
  startup.
- It must not import game domain packages.
