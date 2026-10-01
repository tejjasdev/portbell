# Troubleshooting

## "port 8080 is already in use"

Another program uses that port. Choose another: `portbell -p 9000`. To see what uses it:
`lsof -i :8080` (macOS, Linux) or `netstat -ano | findstr :8080` (Windows).

## "Window too small"

portbell needs at least 60 columns by 14 rows. Make the window larger.

## The screen looks broken in tmux or screen

Make sure `TERM` inside tmux is `tmux-256color` or `screen-256color`, and that your outer
terminal is a modern one. If it still looks wrong, please open a bug and include your
`tmux -V` and terminal name. These are exactly the reports we want.

## Colours or symbols look wrong on Windows

Use Windows Terminal rather than the old console window. If `●`, `▸` or `…` show as boxes,
your font lacks them; switch to a font such as Cascadia Mono.

## "c" does not copy to the clipboard

Copy uses the terminal's own clipboard feature (OSC 52). Some terminals need it enabled; tmux
needs `set -g set-clipboard on`. Over SSH it works only if your local terminal allows it.
Use `s` to save the request to a file instead.

## "standard output is not a terminal"

You are piping or redirecting the output. The screen needs a terminal. Use `--headless` for
JSON lines.

## Requests do not arrive

- Check the address on the first line of the screen and use exactly that.
- A request from another computer needs `-b 0.0.0.0` (read the warning first) and a firewall that allows the port.
- Webhooks from the internet need a public URL: run a tunnel such as ngrok or cloudflared
  that forwards to your portbell port.

## The reply I set is not used

A new reply applies to the **next** request. Check the reply bar at the bottom of the screen
for the reply that is active.

## Still stuck

Open a [bug report](https://github.com/tejjasdev/portbell/issues/new/choose). The form asks
for your version, operating system and terminal, which helps a lot.
