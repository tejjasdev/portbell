# Recipe: a fake backend with CORS

**Goal:** your front end at `http://localhost:3000` needs an API that is not ready yet.

1. Write the reply you want in `reply.json.http`:

   ```text
   HTTP/1.1 200 OK
   Content-Type: application/json

   {"items": [{"id": 1, "name": "first"}]}
   ```

2. Start portbell with CORS switched on:

   ```sh
   portbell -p 8000 --cors -r reply.json.http
   ```

3. Point your front end at `http://127.0.0.1:8000`.

Browsers first send an `OPTIONS` preflight request for many calls. portbell answers it with `204` and the
right `Access-Control-*` headers, and shows it in the list, so you can see what the browser asked for.
Normal replies get `Access-Control-Allow-Origin` added unless your reply already sets it.

portbell answers the same reply to every path. To try different data, press `e` and change the reply.
