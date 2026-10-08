## Sentinel's Journal

## 2026-10-08 - Added CORS headers protection
**Vulnerability:** CRLF Injection in headers via Environment Variables (HTTP Response Splitting).
**Learning:** `responder.Render()` expands `${VAR}` in header values using `r.env.Expand(h.Value)`. However, the environment variable values themselves are not sanitized! `Spec.Validate()` only checks the literal header values in the config/CLI before expansion (`validHeaderValue`). If an environment variable like `PORTBELL_API_KEY` contains `\r\n` (CRLF), the expanded value will be injected into the raw HTTP response, leading to HTTP Response Splitting.
**Prevention:** Always validate expanded values, or strip `\r` and `\n` from environment variables during expansion in headers.
