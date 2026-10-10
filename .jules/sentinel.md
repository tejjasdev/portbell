## 2026-10-10 - Update SaveRequest Directory Permissions
**Vulnerability:** Saved request directories had overly permissive permissions (`0o755`), potentially allowing local users to view saved files including sensitive payloads/headers.
**Learning:** The application captures arbitrary incoming HTTP requests and allows users to export/save them locally. While `SaveRequest` attempts to redact sensitive headers, other non-redacted data might be sensitive. The directory `SaveDir` where requests are saved was created with `0o755`.
**Prevention:** Create directories containing sensitive user-data/traffic with `0o700` permissions so only the owner has access.
