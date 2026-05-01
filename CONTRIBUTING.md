# Contributing to netglance

Thanks for your interest. This is a small project; the bar for contributions
is "make sense, don't break things, keep it tight". The notes below help us
get there with as little back-and-forth as possible.

## Reporting bugs and asking for features

Use the [issue templates](https://github.com/netglance/netglance/issues/new/choose).
For open-ended questions, please use [Discussions](https://github.com/netglance/netglance/discussions)
instead of issues.

If you're reporting a security issue, do **not** open a public issue —
see [SECURITY.md](SECURITY.md).

## Setting up a dev environment

Most things only need Docker and Node. The OPNsense plugin part needs a
FreeBSD VM (see below).

```sh
# clone, then:
make local      # backend + frontend in Docker on http://localhost:8473
make ui         # frontend with HMR; needs a backend running somewhere
make test       # go test ./...
```

For OPNsense plugin work, you need a VM with OPNsense installed —
[dev/opnsense-vm/README.md](dev/opnsense-vm/README.md) walks through it.
Then:

```sh
make dev-vm-deploy  VM_HOST=root@<vm-ip>   # cross-build + scp the daemon
make dev-plugin-sync VM_HOST=root@<vm-ip>  # rsync the plugin tree
make dev-plugin-logs VM_HOST=root@<vm-ip>  # tail logs
```

## Code style and conventions

- **Go**: standard `gofmt` + `go vet`. Errors get wrapped with `fmt.Errorf("...: %w", err)`.
  No third-party logger — use `log/slog` like the rest of the codebase.
- **TypeScript / React**: existing files set the tone. Function components,
  hooks, no class components. Tailwind for styling. No new state libraries
  unless there's a real reason.
- **Comments**: write the *why*, not the *what*. If a comment is just
  describing what the next line does, delete it. If a constant or branch
  exists because of a non-obvious constraint or past incident, *that* is
  worth a comment.
- **Tests**: Go has `go test ./...` for the few covered packages.
  We don't gatekeep PRs on coverage, but new logic with edge cases should
  come with a test.

## Pull requests

1. Open an issue first if the change is non-trivial (a new feature, an API
   break, a refactor that touches >1 package). For straightforward bug
   fixes, just send the PR.
2. One topic per PR. "drive-by improvements" make review harder — submit
   them separately.
3. Commit messages: imperative mood, type prefix (`fix:`, `feat:`, `docs:`,
   `refactor:`, `ci:`, `deploy:`), concise subject. The body, when needed,
   explains *why*.
4. Don't add features, error handling for impossible cases, or
   abstractions for hypothetical future requirements.
5. CI must pass. If a flaky test breaks your run, mention it — don't
   silently retry until green.

## Releasing (maintainers)

See the "Maintainer workflow" section in
[docs/install/opnsense-plugin.md](docs/install/opnsense-plugin.md). In short:

```sh
git tag v0.X.Y
git push --tags
# CI builds Linux/FreeBSD binaries, the Docker image,
# and the FreeBSD pkg repo on the gh-pages branch.
```

For release candidates use `vX.Y.Z-rc.N` — `pkg` orders them as
prereleases.

## License

By contributing, you agree your contributions will be licensed under the
project's [MIT license](LICENSE).
