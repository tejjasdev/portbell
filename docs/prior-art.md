# Prior art and alternatives

portbell is not a new idea, and it is not the right tool for every job. This page says what
else exists, what each is good at, and what portbell tries to do differently. Facts below were
checked against each project's public page on 2026-10-01; please open an issue if something is
out of date.

| Tool | What it is | Where it fits |
| --- | --- | --- |
| [httplab](https://github.com/qustavo/httplab) | Interactive terminal web server: inspect requests and forge replies. Written in Go. No release or commit since early 2024 | The direct inspiration for portbell. If it works for you, it is still a good tool |
| [webhook-tester](https://github.com/tarampampam/webhook-tester) | Web UI for receiving and inspecting webhooks, usually run with Docker | Better if you want a browser UI or a shared page |
| [HookTUI](https://pypi.org/project/hooktui/) | Python/Textual terminal webhook inspector, with SQLite history and extra SMTP and DNS capture | Better if you want history that survives restarts, or email and DNS capture |
| [Proxelar](https://github.com/emanuele-em/proxelar) | Rust traffic workbench: proxy, intercept, rewrite, replay, scripting | Better if you need a proxy, HTTPS interception or scripting |
| webhook.site and similar hosted services | Public URL that records requests | Better if you need a public URL with no tunnel; your data goes to a third party |
| Bruno, Posting, curl | API clients that **send** requests | The other half of the job: portbell receives, they send |
| mitmproxy | Full intercepting proxy | Better for inspecting traffic you do not control |

## What portbell tries to do

- Stay small: one job, one binary for Windows, macOS and Linux, no runtime.
- Be easy to start and easy to learn: the empty screen tells you what to do.
- Be careful by default: local-only bind, safe display of untrusted text, redaction on export.
- Be maintained and documented so it is easy to contribute to.

If you need something in the "better if" column above, please use that tool. We would rather
you have the right tool than the one we made.
