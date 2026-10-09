## 2024-05-24 - [Protect Local Exports from Unauthorized Access]
**Vulnerability:** Saved requests were created in a directory with 0755 permissions, making them readable by any user on the system.
**Learning:** For a local developer tool that inspects and exports potentially sensitive HTTP requests (containing auth tokens, cookies, payloads), exporting them with world-readable permissions poses a local privilege escalation/information disclosure risk.
**Prevention:** Use stricter directory creation permissions (0o700) for paths handling sensitive user exports, ensuring only the owner has access. Redact `Set-Cookie` in exports in addition to `Cookie` and `Authorization` headers.
