# Recipe: see what Razorpay sends

**Goal:** look at a Razorpay webhook (for example `payment.captured`) in test mode.

1. Start portbell: `portbell -p 8080`
2. Start a tunnel to port 8080 (ngrok, cloudflared) and copy the public HTTPS URL.
3. In the Razorpay dashboard (test mode): **Account Settings → Webhooks → Add New Webhook**. Paste the
   URL, choose the events, and set a secret.
4. Make a test payment. The webhook appears in portbell.

Look at the `X-Razorpay-Signature` header (what your server must verify using your webhook secret) and the
JSON body with the event name and payload.

Razorpay expects a `2xx` reply; the default `200` is fine. To check how your integration behaves when your
endpoint fails or is slow, press `e` and set status `500`, or set a delay of a few seconds.

Use test mode, and remove the webhook and the tunnel when you are finished.
