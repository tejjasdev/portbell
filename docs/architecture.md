# Architecture

portbell is built around one idea: **a stream of request and reply events**. Every other part
only reads from that stream or writes to it. This keeps the parts small, testable, and
replaceable.

```text
HTTP clients <--> server --publish--> events --> tui       (screen)
                    ^                      |---> headless  (JSON lines)
                    |                      '---> export    (curl, files)
                responder <--- config (defaults)
                    ^
                    '------- tui (edited reply)
```

## Packages

All code is under `internal/`, so nothing is a public Go API before v1.0.

| Package | Job | Depends on |
| --- | --- | --- |
| `cmd/portbell` | Entry point. Only calls `app.Run` | `app` |
| `internal/app` | Reads flags and config, wires everything, runs screen or headless mode | all below |
| `internal/events` | `Request`, `Response`, `Event` types and the in-memory `Store` that publishes updates | none |
| `internal/server` | HTTP listener: records requests, applies limits and CORS, waits for the delay, sends the reply | `events`, `responder` |
| `internal/responder` | The current reply (`Spec`), validation, `${VAR}` expansion, the reply file format | `events`, `brand` |
| `internal/config` | TOML config: load, validate, defaults, discovery | `responder`, `events`, `brand` |
| `internal/tui` | The terminal screen (Bubble Tea) | `events`, `responder`, `export`, `safe` |
| `internal/headless` | JSON-lines output | `events` |
| `internal/export` | Curl command, saved request files, redaction, safe file names | `events`, `safe` |
| `internal/safe` | Makes untrusted text safe to print in a terminal | none |
| `internal/brand` | The project name in one place | none |
| `internal/version` | Build version, set at link time | none |

## How one request travels

1. A client sends a request to the server.
2. The server applies the size limit, stores the request as an event, and (with `--cors`)
   answers preflight requests itself.
3. It asks the responder for the reply, which is the one set in the screen or the defaults.
4. It waits for the delay, and stops early if the client disconnects (the event is marked aborted).
5. It sends the reply and stores it on the same event.
6. The screen, headless mode and export see the event update.

## Rules

- `tui`, `headless` and `export` never import `server`. A new front end (a web view, a log
  file writer) subscribes to events and needs no change in `server`.
- Each request runs in its own goroutine. The event store is the only shared state, guarded
  by a mutex. A slow subscriber loses updates rather than slowing the server down.
- Everything from a request that reaches a terminal passes through `internal/safe`.
- The store keeps the most recent 1,000 events (configurable) so memory stays bounded.

## Where to add things

| I want to... | Change |
| --- | --- |
| Add a new output (file logger, web view) | A new package that calls `Store.Subscribe()` |
| Add a new source of replies (rules file, script) | `internal/responder`, behind the same `Spec` |
| Add a flag or config key | `internal/app` and `internal/config`, plus `docs/configuration.md` |
| Change what the screen shows | `internal/tui` |

Big decisions are recorded in [`adr/`](adr).
