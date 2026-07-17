# Contributing to LocalAI Hub

## Development Setup

**Prerequisites:**
- Go 1.23+
- Node.js 22+
- npm

```bash
# Clone and set up
git clone https://github.com/delta574/localai-hub.git
cd localai-hub

# Build the web frontend
cd web && npm ci && npm run build && cd ..

# Build the Go binary
go build ./...
```

## Code Standards

- **Formatting:** Run `go fmt ./...` before committing.
- **Linting:** Run `go vet ./...` — all code must pass vet checks.
- **Frontend:** Run `cd web && npx svelte-check` to validate Svelte files.
- Keep changes focused. Prefer deleting code over adding it.
- No speculative abstractions, no unused exports, no dead code.

## Pull Request Workflow

1. Fork the repository.
2. Create a feature branch: `git checkout -b feat/your-feature`
3. Commit your changes (see commit conventions below).
4. Push to your fork and open a pull request against `main`.
5. Ensure CI passes (build, lint, test).

## Commit Messages

Conventional commits are preferred:

```
feat: add X
fix: correct Y
chore: bump dependency
docs: update README
refactor: extract Z
```

## Testing

- Add tests for new features or bug fixes.
- Run tests with `go test ./...` before opening a PR.
- Test coverage must not regress for non-trivial changes.

## Security

See [SECURITY.md](./SECURITY.md) for reporting vulnerabilities.
