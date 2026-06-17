# Sicherheitsrichtlinie

## Unterstützte Versionen

| Version | Unterstützt |
| --- | --- |
| 0.1.x | ja |

## Schwachstellen melden

Sicherheitsprobleme bitte vertraulich an den Repository-Maintainer melden.
Keine öffentlichen Issues für Credential-Leaks oder Exploit-Details.

## Umgang mit Geheimnissen

- `config.json`, `.env` und Login-Tokens niemals committen.
- In CI `config.example.json` und Umgebungsvariablen verwenden.
- Lokale Config-Dateien mit restriktiven Rechten speichern (0600).

## API-Zugangsdaten

ChurchTools-Login-Tokens gewähren vollen Zugriff als der jeweilige Benutzer.
Tokens rotieren, wenn sie möglicherweise offengelegt wurden.
