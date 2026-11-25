# Database Platform

This package owns MySQL connection pooling, transaction boundaries and stable
storage error mapping. Domain modules must use generated `sqlc` queries rather
than issuing ad hoc SQL from HTTP handlers.

Rules:

- MySQL is authoritative for business state.
- Application code owns transaction scope; repositories do not open nested
  transactions.
- Use `READ COMMITTED` by default and explicit row locks for game command paths.
- Never log DSNs, passwords or decrypted secret payloads.
