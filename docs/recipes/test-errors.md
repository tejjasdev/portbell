# Recipe: test how your app handles errors

**Goal:** see what your app does on a `500`, a `429`, a slow reply, or broken JSON, without changing a backend.

Point your app at `http://127.0.0.1:8080`, then change the reply while it runs.

| Test | Start with | Or press `e` and set |
| --- | --- | --- |
| Server error | `portbell -s 500` | `500 Internal Server Error` |
| Rate limited | `portbell -r rate-limited.http` | `429`, header `Retry-After: 5` |
| Slow reply | `portbell -d 3s` | the delay field to `3s` |
| Timeout | `portbell -d 60s` (longer than your client's timeout) | delay `60s` |
| Broken JSON | `portbell -r broken.http` | body `{"id": 1,` |
| Empty reply | `portbell -s 204` | `204 No Content` |

`rate-limited.http` for example:

```text
HTTP/1.1 429 Too Many Requests
Retry-After: 5
Content-Type: application/json

{"error": "slow down"}
```

During a delay, the request shows `…` in the list. If your client gives up first, it shows `✗`
(client disconnected), which is a good way to confirm your timeout works.

Not yet supported: closing the connection without any reply (planned for v0.3).
