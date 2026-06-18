# ChurchTools_Dup

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://golang.org)
[![Release](https://img.shields.io/badge/Release-GitHub-0077B6)](https://github.com/janmz/ChurchToolsDup/releases)
[![Lizenz: MIT (modifiziert)](https://img.shields.io/badge/Lizenz-MIT--Modified-blue.svg)](LICENSE)
[![Unterstützung: CFI-Kinderhilfe](https://img.shields.io/badge/Unterstützung-CFI--Kinderhilfe-0077B6?logo=heart)](https://cfi-kinderhilfe.de/jetzt-spenden?q=VAYACTDUP)
[![Build Status](https://github.com/janmz/ChurchToolsDup/actions/workflows/ci.yml/badge.svg)](https://github.com/janmz/ChurchToolsDup/actions/workflows/ci.yml)

<p align="center">
  <img src="https://img.shields.io/badge/🇩🇪-Deutsch-0077B6?style=for-the-badge" alt="Deutsch (aktuell)">
  <a href="README.md"><img src="https://img.shields.io/badge/🇺🇸-English-555?style=for-the-badge" alt="English"></a>
</p>

**churchtools-dup** ist ein Go-CLI zur **Erkennung, Prüfung und Vormerkung von
Dubletten** in ChurchTools:

- Dubletten im Gesamtbestand für einen Standort finden und als CSV exportieren
- CSV manuell bereinigen (falsche Treffer entfernen, Reihenfolge pro DupID
  beachten)
- Verbleibende Paare im Beziehungsmanagement verknüpfen und Originale in die
  Gruppe „Duplikate“ aufnehmen
- Bereits bestehende Duplikat-Beziehungen werden erkannt und übersprungen

Typischer Anwendungsfall: Nach einem Datenimport oder bei gewachsenen Beständen
Dubletten sichtbar machen, vor der Zusammenführung in Ruhe prüfen und
strukturiert vormerken.

## Funktionen

### Dubletten-Erkennung (`export`)

Treffer werden im **gesamten Personenbestand** gesucht; mindestens ein Eintrag
pro Gruppe muss zum gewählten Standort gehören. Erkennungsregeln (vereinfacht):

1. Gleiche E-Mail (außer Ehepaare/gemeinsame Mailbox: gleiche E-Mail und
   Adresse, aber unterschiedlicher Vorname)
2. Gleicher Vorname + Stadt + Straße (mit toleranter Schreibweise)
3. Gleicher Vorname + Nachname (inkl. vertauschter oder abweichender Namen)

### Import (`import`)

- Pro **DupID** wird der **erste CSV-Eintrag** als Original (Primary) behandelt
- Weitere Einträge derselben DupID werden als Duplikate verknüpft
- Bestehende Duplikat-Beziehungen werden nicht erneut angelegt
- Das Original wird in die Gruppe **Duplikate** aufgenommen (abschaltbar)

### Weitere Befehle zur Kontrolle und Konfiguration

- `whoami` – angemeldeter Benutzer, Standort, Gruppenmitgliedschaften
- `relationship-types` – alle Beziehungstypen mit ID und Name
- `setup` – Konfiguration, Verbindungstest, Token, Berechtigungshinweise

## Voraussetzungen

- Go 1.22+ (zum Bauen aus dem Quellcode)
- ChurchTools-Zugang (Login-Token oder Benutzername/Passwort)
- **Export:** Berechtigung **export data** (Gruppe „Personen exportieren“)
- **Import:** Berechtigung **edit relations** (Gruppe „Personen Administration“
  oder „Personen bearbeiten“)

## Installation

### Binary herunterladen

[Releases](https://github.com/janmz/ChurchToolsDup/releases) – Archiv
entpacken, `churchtools-dup.exe` (Windows) bzw. `churchtools-dup` ausführen.

### Go Install

```bash
go install github.com/janmz/churchtools-dup@latest
```

### Aus Quellcode bauen

```bash
git clone https://github.com/janmz/ChurchToolsDup.git
cd ChurchToolsDup
go build -o churchtools-dup.exe .
```

## Verwendung

### Schnellstart

```bash
copy config.example.json config.json
# config.json anpassen oder setup init

.\churchtools-dup.exe setup test
.\churchtools-dup.exe export -o dup.csv -i
```

CSV bearbeiten (falsche DupID-Zeilen löschen; erster Eintrag pro DupID bleibt
Original), dann Dry-Run und Import:

```bash
.\churchtools-dup.exe import -f dup.csv --dry-run
.\churchtools-dup.exe import -f dup.csv
```

Globale Option: `-c config.json` für einen anderen Konfigurationspfad.

### Beziehungstyp ermitteln

```bash
.\churchtools-dup.exe relationship-types
```

Die ID des Duplikat-Typs in `config.json` eintragen, z. B.:

```json
"duplicate_relationship_type": { "id": 8 }
```

## Konfiguration

Kopiere `config.example.json` nach `config.json` oder nutze Umgebungsvariablen:

| Variable / Feld | Beschreibung |
| --- | --- |
| `CT_BASE_URL` | Instanzname (z. B. `meine-gemeinde`) oder volle URL |
| `CT_LOGIN_TOKEN` | API-Login-Token |
| `CT_USERNAME` / `CT_PASSWORD` | Alternative zum Token |
| `delay_ms` | Pause zwischen API-Aufrufen in Millisekunden (Standard: 500) |
| `campus_id` | Standard-Standort, wenn der Benutzer keinen hat |
| `permission_groups.edit_persons` | Gruppe für Import-Rechte (Standard: Personen Administration, Fallback Personen bearbeiten) |
| `permission_groups.export_persons` | Gruppe für Export (Standard: Personen exportieren) |
| `duplicate_relationship_type.id` | Beziehungstyp-ID für Duplikat-Verknüpfung (Standard: **8**) |
| `duplicate_relationship_type.name` | Beziehungstyp-Name (nur Fallback, wenn ID 8 fehlt) |

Login-Token beschaffen:

```bash
.\churchtools-dup.exe setup init
.\churchtools-dup.exe setup token
```

### Haupt- und Nebeninstanz (OAuth)

Bei Mandanten mit Zentral- und Nebeninstanz unterstützt das Tool den
OAuth-Übergang (siehe englische README oder `setup init`). API-Aufrufe laufen
über die konfigurierte (Neben-)Instanz.

## CSV-Format (Dubletten)

```csv
DupID,ID,Vorname,Nachname,E-Mail,Straße,Stadt,Standort,Erstellungsdatum,Einwilligungsdatum
1,10001,Anna,Beispiel,anna.beispiel@example.org,Lindenweg 4,Musterstadt,Standort Nord,20.05.2026,
1,10002,Anna,Beispiel,,Lindenweg 4,Musterstadt am Main,Standort Süd,20.05.2026,
```

- **DupID** – Gruppennummer; alle Zeilen mit gleicher DupID gehören zusammen
- **ID** – ChurchTools-Personen-ID (Pflicht)
- Weitere Spalten zur manuellen Prüfung; beim Import maßgeblich sind DupID und ID
- Zeilen einer DupID löschen, um Treffer zu verwerfen
- **Erster Eintrag** einer DupID wird beim Import zum Original

### Import-Ausgabe

- **OK** – mindestens eine neue Duplikat-Beziehung angelegt
- **ÜBERSPRUNGEN** – alle Beziehungen der Gruppe waren bereits vorhanden (oder
  weniger als zwei Personen in der DupID)
- **FEHLER** – API- oder Berechtigungsproblem

Zusammenfassung mit Anzahl verknüpft / bereits vorhanden.

## Befehle

| Befehl | Zweck |
| --- | --- |
| `setup init` | Interaktive `config.json` anlegen |
| `setup test` | Login und Verbindung testen |
| `setup token` | Login-Token anzeigen |
| `setup permissions` | Berechtigungshinweise für Dubletten-Export/Import |
| `whoami` | Angemeldeter Benutzer, Standort, Gruppen, Instanz-URL |
| `relationship-types` | Beziehungstypen mit ID und Name auflisten |
| `export -o DATEI` | Dubletten-CSV exportieren (Standard: `duplikate.csv`, `-` = stdout) |
| `export -i` | Standort interaktiv wählen |
| `export --campus-id ID` | Dubletten-Suche für diesen Standort |
| `export --skip-permission-request` | Keine Gruppenanfrage bei fehlenden Export-Rechten |
| `import -f DATEI` | Bearbeitete CSV importieren |
| `import -f DATEI --dry-run` | Prüfen ohne Änderungen in ChurchTools |
| `import -f DATEI --delay-ms MS` | Pause zwischen API-Aufrufen |
| `import -f DATEI --skip-group-add` | Gruppe „Duplikate“ nicht befüllen |
| `import -f DATEI --skip-permission-request` | Keine Gruppenanfrage bei fehlenden Rechten |

## Entwicklung

Unter Windows wird bei Releases `vaya.ico` per
[go-winres](https://github.com/tc-hib/go-winres) eingebettet.

**Tests:** `go test ./...` – Details in [TESTING.de.md](TESTING.de.md).

```bash
go test ./...
go vet ./...
go build -o churchtools-dup.exe .
```

## Contributing

Beiträge sind willkommen – bitte [CONTRIBUTING.de.md](CONTRIBUTING.de.md) lesen.

## Lizenz

Modifizierte MIT-Lizenz (siehe [LICENSE](LICENSE)). Namensnennung **Jan Neuhaus,
VAYA Consulting** und Link zum Repository erforderlich. **Keine Gewährleistung.**

## Unterstützung

[CFI-Kinderhilfe](https://cfi-kinderhilfe.de/jetzt-spenden?q=VAYACTDUP)

## Kontakt

**Autor:** Jan Neuhaus, VAYA Consulting –
[VAYA Consulting](https://vaya-consulting.de/development?q=GITHUB)

**Repository:** [https://github.com/janmz/ChurchToolsDup](https://github.com/janmz/ChurchToolsDup)

## Changelog

Siehe [Changelog.md](Changelog.md).
