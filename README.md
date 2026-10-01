# portbell

**See every HTTP request that reaches your machine, and choose the reply, live, from your terminal.**

[![CI](https://github.com/tejjasdev/portbell/actions/workflows/ci.yml/badge.svg)](https://github.com/tejjasdev/portbell/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

> **Status: pre-release (v0.1 in progress).** The core works and is tested. There is no
> published release yet, so the install options below that need a release say so.
> The project name is a working title and may change before v0.1.0.

<!-- TODO before v0.1.0: add a 10-second demo GIF here (record with VHS or asciinema). -->

portbell opens a port on your computer and shows each request that arrives: method,
path, sorted headers and a readable body. You decide what it answers with: any status,
headers, body and delay, changed while it runs. It is one small program for Windows,
macOS and Linux, with nothing to install first.

## Quick start

```sh
portbell
```

Then, in another terminal:

```sh
curl http://127.0.0.1:8080/hello
curl -X POST http://127.0.0.1:8080/hook -d '{"hello":"world"}'
```

Both requests appear on screen. Press `e` to change the reply, `?` for help, `q` to quit.

## Install

| Method | Command | Available |
| --- | --- | --- |
| Go | `go install github.com/tejjasdev/portbell/cmd/portbell@latest` | once the repository is public |
| Download | binaries for Windows, macOS and Linux on the [Releases page](https://github.com/tejjasdev/portbell/releases) | from v0.1.0 |
| Homebrew, Scoop, winget | | planned, see the [roadmap](docs/roadmap.md) |
| From source | `git clone https://github.com/tejjasdev/portbell && cd portbell && make build` | now (needs Go, see `go.mod`) |

## What you can do with it

- **Test a webhook.** See exactly what Stripe, GitHub or Razorpay sends, headers and
  signature included. Recipes: [Stripe](docs/recipes/stripe-webhook.md),
  [GitHub](docs/recipes/github-webhook.md), [Razorpay](docs/recipes/razorpay-webhook.md).
- **Test how your app handles failure.** Make the reply a `500`, a `429` or a 5-second
  delay, without touching a backend. See [test error replies](docs/recipes/test-errors.md).
- **Stand in for a backend that is not ready.** Return a fixed JSON body, with CORS
  handled: [CORS recipe](docs/recipes/cors.md).
- **Learn HTTP.** Watch the raw requests your tools and browser send.
- **Use it in scripts.** `portbell --headless` prints one JSON line per request.

## Useful flags

```text
-p, --port N          port to listen on (default 8080)
-b, --bind ADDR       address to bind (default 127.0.0.1)
-r, --response FILE   reply to send, written like an HTTP response
-s, --status CODE     reply status code
-d, --delay TIME      wait before replying, for example 500ms or 2s
    --cors            answer CORS preflight requests
    --headless        no screen: JSON lines on stdout
-c, --config FILE     config file (TOML)
```

Everything else is in the [usage guide](docs/usage.md) and the
[configuration reference](docs/configuration.md).

## Safe by default

portbell opens a network port and shows text sent by strangers, so it is careful:

- It listens on `127.0.0.1` only. Binding to `0.0.0.0` shows a red warning.
- Control characters in requests are shown as visible text like `\x1b`, so a request
  cannot rewrite your terminal.
- Copied and saved requests hide `Authorization`, `Cookie` and similar headers.
- `${VAR}` in a reply only reads environment variables that start with `PORTBELL_`
  (or that you list), so a shared reply file cannot leak your secrets.
- No telemetry, no network calls except the requests you receive.

Details: [SECURITY.md](SECURITY.md).

## What it is not

portbell does one job. These are deliberate non-goals for version 1:

- **Not an API client.** It receives requests; it does not send them.
  (Use Bruno, Posting or curl for that.)
- **Not a full mock-server framework** with a rules engine.
- **Not a tunnel or proxy.** It works well behind ngrok or similar, but does not replace them.
- **No accounts, cloud or telemetry**, ever.

## Prior art and credit

portbell is **not a new idea**. It is inspired by
[httplab](https://github.com/qustavo/httplab) by Gustavo Chaín, which pioneered the
"inspect requests and forge replies in the terminal" workflow and has not been updated
since 2024. portbell is a fresh implementation (no httplab code was copied) that aims to
be actively maintained, documented, and easy to install on every system.

Other tools solve nearby problems, and may suit you better:
see [docs/prior-art.md](docs/prior-art.md) for an honest comparison.

## Documentation

| | |
| --- | --- |
| [Getting started](docs/getting-started.md) | Install and see your first request |
| [Usage](docs/usage.md) | Every screen and action |
| [Configuration](docs/configuration.md) | Config file and flags |
| [Keybindings](docs/keybindings.md) | All keys |
| [Troubleshooting](docs/troubleshooting.md) | tmux, Windows, busy ports |
| [Architecture](docs/architecture.md) | How the code fits together |
| [Roadmap](docs/roadmap.md) | What comes next |

## Contributing

Contributions are welcome, including documentation, recipes and bug reports from
your own terminal. Start with [CONTRIBUTING.md](CONTRIBUTING.md). Look for issues labelled
[`good first issue`](https://github.com/tejjasdev/portbell/labels/good%20first%20issue).

## License

[MIT](LICENSE) © 2026 Tejas Chavan. Third-party notices: [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
