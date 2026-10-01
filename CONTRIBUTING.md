# Contributing to portbell

Thank you for helping! This guide is short on purpose. If anything is unclear, that is a
bug in this guide: please open an issue.

## Ways to help

- **Report a bug.** Use the bug template. Many problems depend on the terminal, so tell
  us which one you use, and whether it is inside tmux or SSH.
- **Improve the docs.** A wrong or confusing sentence is a real bug.
- **Add a recipe.** One page in [`docs/recipes/`](docs/recipes) for a real task.
- **Fix an issue.** Look for [`good first issue`](https://github.com/tejjasdev/portbell/labels/good%20first%20issue)
  or `help wanted`.

For a new feature or a big change, **open an issue first**, so you do not spend time on
something we cannot accept. portbell does one job; please read the non-goals in the README.

## Set up (3 commands)

You need Go (the version in `go.mod`) and `make`.

```sh
git clone https://github.com/tejjasdev/portbell
cd portbell
make test        # runs all tests
make run         # starts the tool from source
```

Other tasks: `make help`.

## Make a change

1. Create a branch from `main`.
2. Make the change. Keep it small and focused: one idea per pull request.
3. Run `make check` (format, vet, tests). Run `make lint` if you have golangci-lint.
4. Update the docs and add a line under **Unreleased** in `CHANGELOG.md`.
5. Open a pull request and fill in the checklist.

## Rules that keep the project healthy

- **Tests.** Every bug fix comes with a test that fails without the fix.
- **Dependencies.** Prefer the standard library. A new dependency needs one sentence in
  the pull request explaining why that is not enough.
- **Packages depend one way.** `tui`, `headless` and `export` read `events`; nothing
  depends on `tui`. See [docs/architecture.md](docs/architecture.md).
- **Everything lives under `internal/`.** We do not publish a Go API before v1.0.
- **Untrusted text.** Anything from a request that reaches the screen goes through
  `internal/safe`. There is a test for this; please keep it passing.
- **Big decisions get a short record** in [`docs/adr/`](docs/adr).
- **Commit messages.** Short, in the imperative: "Fix crash on empty body". A
  [Conventional Commits](https://www.conventionalcommits.org) prefix is welcome, not required.

## Check your change in a real terminal

Screen bugs hide in specific terminals. If you change anything in `internal/tui`, try it in
at least one real terminal, resize the window, and make it very small. If you can, try Windows
Terminal or tmux too. Say what you tried in the pull request.

## Licence of contributions

portbell is MIT licensed. By opening a pull request you agree that your contribution is
licensed under the same MIT licence. There is no separate agreement to sign.

## Conduct and response time

Everyone is expected to follow the [Code of Conduct](CODE_OF_CONDUCT.md). This is a
volunteer project: we aim to reply to issues and pull requests within a week.
