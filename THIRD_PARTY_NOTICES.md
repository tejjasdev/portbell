# Third-party notices

portbell is licensed under the MIT License (see [LICENSE](LICENSE)).

## Inspiration: httplab

portbell is inspired by [httplab](https://github.com/qustavo/httplab) by
Gustavo Chaín (MIT License, Copyright (c) 2017 Gustavo Chaín), an interactive
terminal web server for inspecting HTTP requests and forging responses.

**No httplab source code, text or assets were copied into portbell.** It is a
new implementation that shares the same idea. If that ever changes, the
original MIT notice will be added here, as the MIT License requires.

## Dependencies

All direct dependencies are MIT licensed. Their full license texts ship with
their source code and are available through `go mod download`.

| Module | Used for | License |
| --- | --- | --- |
| [charm.land/bubbletea/v2](https://github.com/charmbracelet/bubbletea) | Terminal UI framework | MIT, Charmbracelet, Inc. |
| [charm.land/bubbles/v2](https://github.com/charmbracelet/bubbles) | Text input and text area widgets | MIT, Charmbracelet, Inc. |
| [charm.land/lipgloss/v2](https://github.com/charmbracelet/lipgloss) | Terminal styling | MIT, Charmbracelet, Inc. |
| [github.com/charmbracelet/x/ansi](https://github.com/charmbracelet/x) | Width-aware text cutting | MIT, Charmbracelet, Inc. |
| [github.com/BurntSushi/toml](https://github.com/BurntSushi/toml) | Config file parsing | MIT, TOML authors |

Run `go list -m all` for the complete list including indirect dependencies.
