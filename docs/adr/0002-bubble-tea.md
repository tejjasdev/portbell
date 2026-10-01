# 0002: Use Bubble Tea for the screen

**Status:** Accepted

**Context.** httplab depends on an old terminal library and pinned 2017 versions of its dependencies, which
broke its build later (httplab issue #89). Many of its other bugs were terminal-specific (tmux, backspace,
small windows).

**Decision.** Use Bubble Tea v2 with Bubbles and Lip Gloss from Charm: actively maintained, works on Windows,
widely used. Keep the screen in its own package (`internal/tui`) so it can be replaced.

**Consequences.** Bubble Tea v2 is pinned to v2.0.2 (with Bubbles v2.0.0 and Lip Gloss v2.0.0), the newest versions that build with Go 1.24, so contributors do not need the very latest Go. Newer releases of these libraries require Go 1.25 or 1.26; moving to them is a deliberate step (raise the `go` line in `go.mod` and CI together). We test the screen in several
terminals before each release. If the library is ever abandoned, only `internal/tui` changes.
