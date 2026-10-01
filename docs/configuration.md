# Configuration

portbell works with no configuration. A config file is optional, and flags always win.

## Where the file is looked up

1. The file given with `-c FILE` (an error if it does not exist).
2. `./portbell.toml` in the current folder.
3. `portbell/config.toml` in your user config folder (`~/.config` on Linux, `~/Library/Application Support`
   on macOS, `%AppData%` on Windows).

**Order of precedence:** built-in defaults, then the config file, then flags.

Unknown keys are an error (so a typo is caught instead of silently ignored).

## All settings

```toml
# portbell.toml: every key shown with its default
version = 1                 # config format version; lets future releases upgrade old files

[server]
bind = "127.0.0.1"          # "0.0.0.0" opens the port to your network (a warning is shown)
port = 8080
max_body_bytes = 10485760   # largest request body kept (10 MB)
cors = false                # answer CORS preflight requests and add CORS headers
history = 1000              # requests kept in memory (1 to 100000)

[reply]
status = 200
headers = ["Content-Type: text/plain; charset=utf-8"]   # each "Name: value"
body = "Hello from portbell\n"
delay = "0s"                # for example "250ms" or "2s"

[env]
allow = []                  # extra environment variables ${VAR} may read, besides PORTBELL_*

[export]
redact = true               # hide Authorization, Cookie and similar headers when copying or saving
```

## Flags

| Flag | Overrides |
| --- | --- |
| `-p`, `--port` | `server.port` |
| `-b`, `--bind` | `server.bind` |
| `--cors` | `server.cors` |
| `--max-body` | `server.max_body_bytes` |
| `-s`, `--status` | `reply.status` |
| `-d`, `--delay` | `reply.delay` |
| `-r`, `--response FILE` | the whole reply, except the delay |
| `-c`, `--config FILE` | which config file to read |

A port of `0` lets the system choose a free port; the chosen address is shown on screen.

## Compatibility

Before version 1.0 a minor release may change a setting; every change is listed in the
[changelog](../CHANGELOG.md). From 1.0, settings are not renamed or removed without a
deprecation period of at least two minor releases. A config file written by a newer release
is refused with a clear message instead of being half-read.
