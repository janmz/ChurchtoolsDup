# Contributing

Thank you for improving Masseneinladung.

## Setup

1. Install Go 1.22 or newer.
2. Clone the repository.
3. Run `go mod download`.
4. Copy `config.example.json` to `config.json` for local tests (do not commit).

## Checks before submitting

```bash
scripts/ci.sh
```

Or on Windows:

```powershell
scripts/ci.ps1
```

Automated tests and manual acceptance: [TESTING.md](TESTING.md).

## Pull requests

- Keep changes focused.
- Add tests for new behaviour (`go test ./...`).
- Update `Changelog.md` and bump `version.go`.

## Commit messages

Use clear, imperative sentences (e.g. "Add dry-run flag for invite command").
