# Changelog

All notable changes to this project are documented in this file.

## [1.1.1.18] - 2026-06-18 22:55:18

### Fixed

- Import/Dry-Run: bestehende Duplikat-Beziehungen zuverlässiger erkannt (weitere
  API-Feldvarianten, Fallback über Beziehungsnamen)
- Dry-Run: prüft jetzt auch, ob die Primärperson bereits in der Gruppe
  „Duplikate“ ist (statt nur „Gruppe prüfen“)

## [1.1.0.16] - 2026-06-18 21:37:55

### Fixed

- CSV-Import: UTF-8-BOM am Dateianfang (Excel, eigener Export) wird erkannt;
  Fehler „csv benötigt eine DupID-spalte“ bei korrekter Kopfzeile behoben
- CSV-Import: Trennzeichen `,`, `;` oder Tab werden automatisch erkannt;
  Anführungszeichen für Felder mit Komma werden beim Einlesen berücksichtigt
- CSV-Export: schreibt immer mit Komma; Felder mit Komma werden in `"…"` gesetzt

## [1.1.0.15] - 2026-06-17 18:21:56

### Removed

- Reste des Einladungs-Workflows (`internal/invite`, Einladungs-CSV,
  `invite`-API, E-Mail-Sync, Personen-Status-API)
- Lokale Export-Dateien mit Produktivdaten (`dup.csv`, `dup-test.csv`)
- Veraltete VS-Code-Launch-Konfiguration für `invite`

### Changed

- `TESTING.md` / `TESTING.de.md` auf Dubletten-Workflow aktualisiert

## [1.1.0.14] - 2026-06-17 18:07:52

### Changed

- Nutzertexte: deutsche Groß-/Kleinschreibung vereinheitlicht (Nomen, OK/Fehler,
  DupID, Person, Beziehung, Gruppe usw.)
- README.de.md und README.md auf Dubletten-Workflow (export/import) aktualisiert

## [1.1.0.13] - 2026-06-17 17:59:45

### Fixed

- Import-Zusammenfassung: DupID-Gruppen ohne neue Verknüpfung (nur bereits
  vorhandene Beziehungen) zählen als „übersprungen“, nicht als „ok“; Summe
  zeigt verknüpft/bereits vorhanden getrennt

## [1.1.0.12] - 2026-06-17 17:55:25

### Added

- Befehl `relationship-types`: alle Beziehungstypen mit ID und Name auflisten

## [1.1.0.11] - 2026-06-17 17:54:04

### Fixed

- Import: bestehende Dubletten-Beziehungen werden per
  `GET /persons/{id}/relationships` erkannt und übersprungen (z. B. Person
  10005 ↔ 10006); Dry-Run meldet „bereits verknüpft“

## [1.1.0.10] - 2026-06-17 17:45:47

### Fixed

- Import: nur noch `edit relations` prüfen (nicht `administer persons`); keine
  irreführende Meldung „weiterhin nicht aktiv“ nach aktiver Gruppenmitgliedschaft
- Beziehungstyp: bei mehreren Treffern Priorität `Duplikat` > `Dublette` >
  `Duplicate`; optional `duplicate_relationship_type` in `config.json`
  (id/name); Fehlermeldung listet Kandidaten mit IDs

## [1.1.0.9] - 2026-06-17 17:39:44

### Fixed

- Berechtigungsgruppen: Standard-Fallbacks `Personen Administration` und
  `Personen bearbeiten`; Suche probiert mehrere Namen (`FindGroupByNames`)
- Beziehungstypen: korrekter API-Pfad `/person/relationshiptypes` (Import/Dry-Run)

## [1.1.0.8] - 2026-06-17 17:32:39

### Added

- `whoami`: Gruppenmitgliedschaften des angemeldeten Benutzers mit ID und Name
  (`GET /persons/{id}/groups`)

## [1.1.0.7] - 2026-06-17 16:48:15

### Changed

- Stadtvergleich über `compareCity`: Normalisierung und Namenszusätze
  (`am`, `an der`, `Trebur/Astheim` usw.) werden intern verglichen; Aufrufer
  nutzen nicht mehr vorab normalisierte Stadtstrings
- Blocking-Keys verwenden nur noch den Hauptortnamen aus `normalizeCity`

## [1.1.0.6] - 2026-06-17 16:20:41

### Changed

- Dubletten-Erkennung: Paare mit gemeinsamer E-Mail und gleicher Adresse
  (Straße, Stadt, Standort), aber unterschiedlichem Vornamen werden nicht mehr
  als Dubletten gelistet (z. B. Ehepaare mit gemeinsamer Mailbox)
- Straßenvergleich: `str.`/`Straße`/`Strasse` werden gleich behandelt;
  nach Punkten wird ein Leerzeichen ergänzt; Leerzeichen und Bindestriche im
  Straßennamen sind beim Vergleich gleichwertig

## [1.1.0.5] - 2026-06-17 16:08:33

### Fixed

- Erstellungsdatum: ChurchTools liefert `meta.createdDate` (nicht `createdAt`);
  Ausgabe als `DD.MM.YYYY`
- Einwilligungsdatum: Datumsformatierung vereinheitlicht; leer wenn in
  ChurchTools keine Datenschutz-Einwilligung hinterlegt ist

## [1.1.0.4] - 2026-06-17 15:57:09

### Added

- CSV-Export: Spalten **E-Mail** und **Standort** ergänzt

### Fixed

- Erstellungsdatum und Einwilligungsdatum: Personendetails werden für
  Dubletten nachgeladen (`GET /persons/{id}`), da die Listen-API die Felder
  oft nicht liefert
- Erweiterte Feld-Erkennung für Adresse, Campus, Erstellungsdatum und
  Datenschutz-Einwilligung

## [1.1.0.3] - 2026-06-17 15:45:14

### Changed

- Dubletten-Erkennung Stufe 2: Vorname + Stadt + Straße (statt nur Vorname +
  Stadt)

## [1.1.0.2] - 2026-06-17 15:41:41

### Added

- Dubletten-Erkennung für einen Standort im Gesamtbestand (`export`): gleiche
  E-Mail, Vorname+Stadt+Straße oder Vorname+Nachname inkl. üblicher Abweichungen
- CSV-Format: DupID, ID, Vorname, Nachname, Straße, Stadt, Erstellungsdatum,
  Einwilligungsdatum
- Import (`import`): verbleibende Dubletten per Beziehungsmanagement verknüpfen,
  erster Eintrag wird in Gruppe „Duplikate“ aufgenommen
- Fuzzy-Abgleich: weggelassene Vornamen, Bindestrich/Leerzeichen, Initialen,
  Teil-Doppelnamen, vertauschte Vor-/Nachnamen

### Removed

- Befehl `invite` und Einladungs-Export (Projektzweck: Dubletten statt Einladungen)

## [1.0.0.1] - 2026-06-17 15:18:27

### Changed

- Projekt von `churchtools-invite` als Ausgangspunkt für `churchtools-dup`
  übernommen: Datei- und Modul-Umbenennungen (`churchtools-dup.go`,
  `churchtools-dup.code-workspace`, Go-Modul `github.com/janmz/churchtools-dup`)

## [2.3.2.32] - 2026-06-17 10:28:25

### Fixed

- `embed-windows-icon.sh`: `go-winres` für den CI-Host installieren (nicht mit
  `GOOS=windows` — sonst `go-winres.exe`, auf Linux nicht ausführbar)

## [2.3.0.30] - 2026-06-17 10:25:39

### Fixed

- Release-Build (Windows): `embed-windows-icon.sh` ruft `go-winres` über
  `$(go env GOPATH)/bin` auf (CI-PATH enthält dieses Verzeichnis oft nicht)

## [2.2.1.24] - 2026-06-17 10:04:31

### Fixed

- HTTP-401-Behandlung: höchstens ein Re-Login-Versuch pro API-Aufruf (keine
  unbegrenzte Rekursion bei dauerhaftem 401)
- Paginierung: Abbruch nach maximal 10.000 Seiten (Schutz vor Endlosschleifen
  bei fehlerhaften API-Antworten)

## [2.2.1.23] - 2026-06-17 09:47:50

### Added

- `TESTING.md` / `TESTING.de.md`: Teststrategie, `go test ./...`, Abgrenzung zu
  manueller Prüfung gegen echte ChurchTools-Instanzen
- Zusätzliche Unit-Tests: Einladungs-Logik (Live-Einladung, E-Mail-Konflikt,
  Sync bei 403), CLI-Export-Hilfen, Terminal-Passwort (Pipe)

### Changed

- README: Hinweis, warum bloßes `go test` im Root keine Tests ausführt

## [2.2.0.21] - 2026-06-17 09:12:47

### Changed

- Bereits eingeladene Personen werden nur noch übersprungen, wenn die E-Mail aus
  der CSV mit ChurchTools übereinstimmt; bei abweichender Adresse erfolgen
  E-Mail-Update und erneute Einladung (ohne `--reinvite`)

## [2.2.0.20] - 2026-06-17 09:03:23

### Added

- OAuth-Bridge für Nebeninstanzen: Login auf Zentralinstanz, dann
  `oauthclients/…/startlogin` mit Redirect-Folge; API-Session auf der
  konfigurierten Nebeninstanz; `MeAPIToken()` via `/api/person/me/apitoken`
- `setup init` holt nach Passwort-Login bevorzugt den API-Token der Nebeninstanz

### Changed

- Passwort-Login auf Nebeninstanzen nutzt nicht mehr nur die Zentral-URL für
  API-Aufrufe, sondern den vollständigen OAuth-Flow (README aktualisiert)

## [2.1.3.19] - 2026-06-17 08:38:57

### Added

- `setup init`: nur Instanzname (z. B. `meine-gemeinde`) statt voller URL;
  Passwort-Eingabe mit `*`-Maskierung (Windows/Linux/macOS, `golang.org/x/term`)

### Fixed

- `CT_BASE_URL` und `base_url` als Instanzname werden in der vollständige URL
  übersetzt (`Validate` wirkte bisher nicht auf die geladene Config)

## [2.1.3.18] - 2026-06-17 08:33:32

### Fixed

- Hauptinstanz-Fallback auch für Login-Token und CSRF-Abruf (Token gilt oft nur
  auf `haupt.church.tools`, nicht auf `haupt-neben.church.tools`); Session-
  Cookies beim Instanzwechsel nicht mehr verworfen

## [2.1.2.16] - 2026-06-17 08:20:05

### Added

- Login mit Benutzername/Passwort: bei URL-Muster `haupt-neben.church.tools`
  automatischer Versuch auf der Hauptinstanz `haupt.church.tools`; Hinweis bei
  erfolgreichem Wechsel (README: Haupt- und Nebeninstanz)

## [2.1.1.14] - 2026-06-16 22:12:13

### Fixed

- Bereits eingeladene/registrierte Personen werden über `invitationStatus`
  erkannt (`accepted`, `pending`); die bisherigen Felder (`isSystemUser` etc.)
  liefert ChurchTools in Personendetails oft gar nicht mit

## [2.1.1.13] - 2026-06-16 22:06:04

### Fixed

- `invite --dry-run`: bereits eingeladene Personen werden erkannt (ChurchTools
  liefert u. a. `isSystemUser` als Zahl); Ausgabe
  `dry-run: würde überspringen: …`

## [2.1.1.12] - 2026-06-16 21:56:10

### Added

- `whoami`: Standort-ID immer ausgeben (eigene Zeile)
- `config.json`: Feld `campus_id` als Standard-Standort

### Changed

- `export`: ohne `--all-campuses` auf Standort des Nutzers einschränken; fehlt
  dieser, `campus_id` aus config oder einmalige interaktive Auswahl mit
  Speicherung in config

## [2.1.0.10] - 2026-06-16 21:46:08

### Fixed

- Release-Workflow: Artefakt-Upload nutzte ungültiges Glob `dist/*.{tar.gz,zip}`
  (leere Release-Assets); explizite Pfade, Prüfung und `workflow_dispatch` zum
  Nachbauen bestehender Tags

## [2.0.0.8] - 2026-06-16 21:37:33

### Added

- GitHub Actions Release-Workflow: bei Tag `v*` werden Binaries für Linux,
  macOS (amd64/arm64) und Windows gebaut und als Release-Assets veröffentlicht

## [2.0.0.6] - 2026-06-16 20:03:39

### Fixed

- CI: `scripts/ci.sh` baut wieder das Root-Modul (`.`) statt veralteten Pfad
  `./cmd/churchtools-invite`
- README-Badges und Repository-Links auf `janmz/ChurchToolsInvite` korrigiert
  (Go-Version-, Release- und Build-Status-Badge)

## [1.0.6.4] - 2026-06-16 19:43:26

### Removed

- Befehl `validate` entfernt (redundant zu `invite --dry-run`)

### Changed

- README: Prüflauf nur noch über `invite --dry-run` dokumentiert
- Flag-Beschreibung `--dry-run` präzisiert

## [1.0.6.3] - 2026-06-16 19:40:57

### Changed

- Flag `--skip-invited` durch `--reinvite` ersetzt: bereits eingeladene Personen
  werden standardmäßig übersprungen; `--reinvite` lädt erneut ein
- README: ausführliche Erläuterung von `validate` und Vergleich mit
  `invite --dry-run`

## [1.0.6.2] - 2026-06-16 18:44:49

### Fixed

- JSON-Parsing für `privacyPolicyAgreement`: ChurchTools liefert das Feld teils
  als Array statt Objekt (`whoami`, Personendetails); `--skip-invited` schlägt
  damit nicht mehr fehl

### Changed

- README.md und README.de.md: Layout wie wp_plugin_releaser (Badges, Sprachboxen,
  Installation, Lizenz/Kontakt/Changelog-Abschnitte)

## [1.0.6.1] - 2026-06-16 18:07:57

### Added

- Flag `--skip-invited` für `invite` und `validate`: bereits eingeladene
  Personen überspringen (Erkennung über `isSystemUser`, `cmsUserId`,
  `acceptedsecurity`, Datenschutz-Einwilligung)

## [1.0.5.1] - 2026-06-16 17:36:34

### Added

- Automatische Gruppenanfrage beim `export` und `invite` (E-Mail-Sync): fehlende
  Rechte `export data` bzw. `write access` lösen Mitgliedschaftsanfrage für die
  konfigurierten Gruppen aus (Standard: „Personen exportieren“ / „Personen
  bearbeiten“)
- Konfiguration `permission_groups` in `config.json`
- Flag `--skip-permission-request` zum Deaktivieren der automatischen Anfrage

## [1.0.4.1] - 2026-06-16 17:31:39

### Changed

- Einladungen über REST-API `POST /persons/{id}/invite` statt Legacy-AJAX
  (`invitePersonToSystem`)
- Export nutzt standardmäßig den Standort (`campusId`) des angemeldeten Nutzers
- `--all-campuses` deaktiviert den automatischen Standort-Filter

### Fixed

- E-Mail-Sync bei fehlender Berechtigung (403): Hinweis und Einladung trotzdem
  an die ChurchTools-Adresse (statt Abbruch)

## [1.0.3.1] - 2026-06-16 17:27:46

### Added

- Export: Standortauswahl (`--campus-id`) und Filter nach Personenstatus
  (`--status-id`) oder Gruppe (`--group-id`)
- `export --interactive` / `-i`: Standort wählen, danach optional filtern
  (alle Personen, Status oder Gruppe am Standort)
- ChurchTools-API: `/campuses`, `/statuses`, `/groups` mit
  `campus_ids[]`-Filter für Personen

## [1.0.2.2] - 2026-06-16 15:36:03

### Changed

- Alle Go-Pakete auf `package ChurchToolsInvite` umbenannt (Einstiegspunkt
  `cmd/masseneinladung/main.go` bleibt `package main`)
- Import-Aliase für eindeutige Paketreferenzen beibehalten

## [1.0.2.2] - 2026-06-16 15:29:17

### Added

- `export`-Befehl: Personenliste als CSV (`id,vorname,nachname,email`) aus
  ChurchTools exportieren, optional gefiltert nach `--group-id`
- UTF-8-BOM für Excel-kompatible CSV-Dateien

## [1.0.1] - 2026-06-16 15:28:20

### Added

- E-Mail-Sync aus CSV/Excel: abweichende Adresse wird primär gesetzt, bisherige
  ChurchTools-Adresse bleibt als zusätzliche erhalten (PATCH /persons/{id})
- Flag `--no-sync-email` zum Deaktivieren des E-Mail-Syncs

## [1.0.0] - 2026-06-16 15:22:22

### Added

- Initial Go CLI for ChurchTools mass invitations from CSV
- ChurchTools REST and legacy AJAX client with login token / password auth
- Setup commands: `init`, `test`, `token`, `permissions`
- Commands: `invite`, `validate`, `whoami`, `--dry-run`
- Example CSV, config templates, tests and CI workflow
