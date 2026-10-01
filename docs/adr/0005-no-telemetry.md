# 0005: No telemetry

**Status:** Accepted

**Context.** portbell shows requests that may contain secrets and personal data.

**Decision.** The program makes no network calls except answering the requests it receives. No usage
statistics, no update checks, no crash reports.

**Consequences.** We learn about problems only from issues and from release download counts. If an update
check is ever added, it will be off by default and documented.
