# Go Commands

```text
api/               public REST API process
realtime-runtime/  WebSocket and game runtime process
worker/            timers, Outbox and maintenance process
```

All three processes use the shared platform layer and stop on SIGINT/SIGTERM.
