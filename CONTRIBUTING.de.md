# Beitragen

Danke, dass du Masseneinladung verbesserst.

## Einrichtung

1. Go 1.22 oder neuer installieren.
2. Repository klonen.
3. `go mod download` ausführen.
4. Für lokale Tests `config.example.json` nach `config.json` kopieren (nicht committen).

## Prüfungen vor dem Einreichen

```bash
scripts/ci.sh
```

Oder unter Windows:

```powershell
scripts/ci.ps1
```

Automatisierte Tests und manuelle Abnahme: [TESTING.de.md](TESTING.de.md).

## Pull Requests

- Änderungen fokussiert halten.
- Tests für neues Verhalten ergänzen (`go test ./...`).
- `Changelog.md` und `version.go` aktualisieren.

## Commit-Nachrichten

Klare, imperative Sätze verwenden (z. B. „Dry-Run-Flag für invite ergänzen“).
