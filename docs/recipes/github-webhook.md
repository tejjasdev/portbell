# Recipe: see what GitHub sends

**Goal:** look at a GitHub webhook delivery. GitHub needs a public URL, so you need a tunnel to your machine.

1. Start portbell: `portbell -p 8080`
2. Start a tunnel that forwards to port 8080 (for example ngrok or cloudflared) and copy the public URL.
3. In your repository: **Settings → Webhooks → Add webhook**. Paste the public URL, choose `application/json`,
   and pick the events you want.
4. Make something happen (push a commit, open an issue). The delivery appears in portbell.

Useful things to look at: the `X-GitHub-Event` header (the event type), `X-GitHub-Delivery` (a unique id),
`X-Hub-Signature-256` (the signature your server should verify), and the JSON body.

GitHub expects a quick `2xx` reply, which the default `200` gives. To see what GitHub shows for a failing
endpoint, press `e` and set the status to `500`.

Remove the webhook when you are done: a tunnel exposes your port to the internet.
