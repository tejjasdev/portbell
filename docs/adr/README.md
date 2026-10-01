# Decision records

One short file per big decision, so the reason is written down for the next contributor.
A decision that changes the core (events, config format, flags) needs a record **before** the code.

Format: **Status**, **Context**, **Decision**, **Consequences**. Keep it under one page. To change
a decision, add a new record that supersedes the old one; do not edit history.

| # | Decision | Status |
| --- | --- | --- |
| [0001](0001-go.md) | Write it in Go | Accepted |
| [0002](0002-bubble-tea.md) | Use Bubble Tea for the screen | Accepted |
| [0003](0003-internal-packages.md) | Keep all packages under `internal/` | Accepted |
| [0004](0004-toml-config.md) | TOML config with a version key | Accepted |
| [0005](0005-no-telemetry.md) | No telemetry | Accepted |
