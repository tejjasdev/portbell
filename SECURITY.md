# Security policy

## Reporting a vulnerability

Please **do not open a public issue** for a security problem.

Report it privately through GitHub:
[Report a vulnerability](https://github.com/tejjasdev/portbell/security/advisories/new),
or email **hey.tejjas@gmail.com** with the subject "portbell security".

Please include the version (`portbell --version`), your operating system and terminal, and
steps to reproduce. You will get a first reply within about a week. Please give us a
reasonable time to fix the problem before you share details publicly. We are happy to credit
you in the release notes if you wish.

## Supported versions

Only the latest release gets security fixes.

## How portbell is designed to be safe

portbell opens a network port and displays text from strangers in your terminal.

| Risk | What we do |
| --- | --- |
| Other people on your network reach the port | Binds to `127.0.0.1` by default. Binding elsewhere shows a visible warning |
| A request contains terminal escape sequences | Control characters are shown as visible escapes (`\x1b`) and never printed raw. JSON output is escaped |
| A huge request exhausts memory | Body size limit (default 10 MB), header size limit, read timeouts, and a history capped at 1,000 requests |
| A shared reply file leaks your environment | `${VAR}` only reads variables starting with `PORTBELL_`, or ones you list under `[env] allow` |
| Secrets end up in copied or saved files | `Authorization`, `Cookie`, `Proxy-Authorization` and `X-Api-Key` are replaced with `REDACTED` on copy and save. The screen shows full values because debugging needs them |
| A saved file overwrites or escapes a folder | Saved requests go to `./portbell-saved`, are never overwritten, and are readable only by their owner |
| Telemetry or hidden network calls | None. The program makes no network requests except answering the ones it receives |

Things to know: portbell serves plain HTTP. Anyone who can reach the port can send requests
and read the reply, so keep the default bind address unless you need otherwise.

## Supply chain

Dependencies are pinned in `go.mod`, updated by Dependabot, and kept few. GitHub Actions are
pinned to commit hashes. Releases include a `checksums.txt` file.
