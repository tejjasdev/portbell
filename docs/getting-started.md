# Getting started

This page takes about two minutes. At the end you will have seen your first request.

## 1. Install

Pick one (see the [README](../README.md#install) for what is available today):

```sh
# With Go
go install github.com/tejjasdev/portbell/cmd/portbell@latest

# Or from source
git clone https://github.com/tejjasdev/portbell
cd portbell
make build          # creates ./bin/portbell
```

Or download a binary for your system from the Releases page and put it on your `PATH`.

Check it works:

```sh
portbell --version
```

## 2. Start it

```sh
portbell
```

You see a screen that says it is waiting for requests on `http://127.0.0.1:8080`.

If the port is busy, you get a clear message and a suggestion. Choose another port with
`-p 9000`.

## 3. Send a request

Open a second terminal and run:

```sh
curl http://127.0.0.1:8080/hello
```

The request appears on the left, and its details (headers and body) on the right. curl
prints the default reply: `Hello from portbell`.

## 4. Change the reply

Press `e`. The first line is the status, then headers, then a blank line, then the body:

```text
HTTP/1.1 200 OK
Content-Type: application/json

{"ok": true}
```

Press `Ctrl+S` to apply. Send the request again: curl now prints your JSON. The reply bar at
the bottom shows the current reply.

## 5. Quit

Press `q` or `Ctrl+C`.

## Next steps

- [Usage](usage.md): every screen and action
- [Recipes](recipes): webhooks, CORS, testing errors
- [Configuration](configuration.md): keep your settings in a file
- [Troubleshooting](troubleshooting.md): if something looks wrong
