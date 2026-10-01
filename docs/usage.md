# Usage

## The screen

```text
 portbell ● listening on http://127.0.0.1:8080
 ▸ Requests (3)          │ Request #3
 12:01:03 POST /hook 200 │ POST /hook HTTP/1.1
 12:01:09 GET  /ping 200 │ from 127.0.0.1:50422 at 12:01:15 · body 31 B · took 2ms
 12:01:15 GET  /ping 200 │
                         │ Headers
                         │   Content-Type: application/json
                         │   Host: 127.0.0.1:8080
                         │ ...
 Reply: 200 OK · delay 0s · body 20 B
 Tab pane · ↑↓ move · e edit reply · c copy curl · s save request · ? help · q quit
```

- **Request list** (left): newest at the bottom. The number at the end of a line is the
  reply status; `…` means the reply is not sent yet (for example during a delay), and `✗`
  means the client went away before the reply.
- **Detail pane** (right): the request line, the headers sorted by name, the body, and the
  reply that was sent.
- **Reply bar**: the reply that will be sent to the next request.
- **Hint bar**: the keys that work right now.

When you are looking at the newest request, the screen follows new requests. Move up to stay
on an older one.

## Body display

- JSON is indented.
- Form bodies (`application/x-www-form-urlencoded`) show one `key = value` per line.
- Binary bodies show a hex dump of the first bytes.
- Large bodies are cut on screen (the full body is kept for copy and save, up to the size
  limit). A note tells you when something was cut.

Control characters from a request are never printed raw; you see `\x1b` instead.

## Editing the reply

Press `e`. The editor shows the reply as an HTTP message. `Tab` switches between the text and the
delay field. `Ctrl+S` applies, `Esc` cancels. If the text is wrong, the editor stays open and
tells you what to fix, and the old reply stays in use.

The new reply applies to the **next** request. It does not change replies already sent.

You can also set the reply before starting:

```sh
portbell -s 503 -d 2s                  # a slow 503
portbell -r examples/reply-json.http   # reply from a file
```

A reply file is a plain HTTP response (see [`examples/`](../examples)). It carries no delay;
set that with `-d`, the config file, or the editor.

## Placeholders

`${PORTBELL_NAME}` in a header or body is replaced with the environment variable of that name:

```sh
PORTBELL_ORDER_ID=42 portbell -r reply.http
```

Only variables that start with `PORTBELL_`, or that you list under `[env] allow` in the
config file, are read. Anything else stays as written. This stops a shared reply file from
leaking your secrets.

## Copy and save

- `c` copies the request as a `curl` command. This uses a terminal feature (OSC 52) that most
  modern terminals support; some need it switched on, and tmux needs `set -g set-clipboard on`.
  If nothing arrives on your clipboard, use `s` instead.
- `s` saves the request as a file in `./portbell-saved`. Files are never overwritten.

The curl command uses POSIX shell quoting (macOS, Linux, Git Bash, WSL). Binary or
cut-off bodies are left out and noted.

Both hide `Authorization`, `Cookie`, `Proxy-Authorization` and `X-Api-Key` values (shown as
`REDACTED`). Turn this off with `redact = false` under `[export]` in the config file.

## CORS

`--cors` makes portbell answer browser preflight (`OPTIONS`) requests and add CORS headers to
replies. The preflight request still appears in the list. See the [CORS recipe](recipes/cors.md).

## Headless mode

```sh
portbell --headless | jq .
```

No screen is drawn. Each finished request is one JSON line on standard output; messages go to
standard error. Fields:

```json
{
  "schema": 1,
  "id": 1,
  "time": "2026-10-01T12:00:00Z",
  "duration_ms": 2,
  "request":  { "method": "POST", "target": "/hook", "proto": "HTTP/1.1", "host": "127.0.0.1:8080",
                "remote": "127.0.0.1:50422", "headers": [{"name": "...", "value": "..."}],
                "body": { "text": "...", "size": 15 } },
  "response": { "status": 200, "headers": [], "body": { "text": "...", "size": 20 } }
}
```

A body is `text` when it is valid UTF-8, otherwise `base64`. `truncated` is `true` when the body
was cut at the size limit; `aborted` is `true` when the client left before the reply. New
fields may be added; existing fields are never renamed or removed within a schema number.

## Exit codes

| Code | Meaning |
| --- | --- |
| 0 | Normal exit |
| 1 | Could not run, for example the port is busy |
| 2 | Bad flags or config file |
