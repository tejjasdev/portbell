# Recipe: see what Stripe sends

**Goal:** look at the exact headers and body of a Stripe webhook event on your own machine.

1. Start portbell:

   ```sh
   portbell -p 4242
   ```

2. Forward test events to it with the [Stripe CLI](https://docs.stripe.com/stripe-cli):

   ```sh
   stripe listen --forward-to localhost:4242/webhook
   stripe trigger payment_intent.succeeded
   ```

3. The request appears in portbell. Look at:
   - the `Stripe-Signature` header (what your server must verify),
   - the JSON body (the event).

**Reply with success.** Stripe expects a `2xx` reply. The default `200` is fine.
**Test your failure handling.** Press `e`, set the status to `500`, apply, and trigger again to see how
Stripe's CLI shows the failure.

Note: signature checks need the exact raw body. portbell shows the body as received; use `s` to save the
request, and compare with what your own server sees.
