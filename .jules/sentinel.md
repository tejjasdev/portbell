## 2026-10-12 - [Missing WriteTimeout in Server]
**Vulnerability:** The HTTP server in internal/server/server.go does not set a WriteTimeout.
**Learning:** This exposes the application to resource exhaustion / Denial of Service (DoS) attacks, such as slowloris attacks on the response side. If a client reads the response very slowly, the connection and associated resources will stay open indefinitely.
**Prevention:** Always configure WriteTimeout on http.Server to ensure connections are closed within a reasonable timeframe if the client is too slow.
