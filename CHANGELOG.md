# Changelog

All notable changes are listed here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project follows
[Semantic Versioning](https://semver.org/). Before 1.0.0, a minor version may change
behaviour; such changes are always listed here.

## [Unreleased]

### Added

- Terminal screen with a request list, a detail pane and a reply editor.
- `--headless` mode that prints one JSON line per request (schema 1).
- Reply from flags (`-s`, `-d`), from a file written like an HTTP response (`-r`), or from a
  TOML config file.
- CORS switch (`--cors`) that answers preflight requests.
- Copy a request as a curl command, and save a request to a file, with secrets hidden.
- Safe display of untrusted text: control characters are shown as visible escapes.
- Reply statuses must be 200-599: interim 1xx codes cannot be sent as a final reply.
- Safe defaults: local-only bind with a warning otherwise, body size limit, timeouts.
