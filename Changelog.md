# Changelog

All notable changes to this project are documented in this file.

## [1.3.2.36] - 2026-11-10 10:24:23

Umsetzung des Sicherheitsaudits vom 2026-11-10.

### Security

- Keine Zugangsdaten mehr an abgeleitete Hosts: Der Login-Fallback auf eine
  „Hauptinstanz“ wird nur noch mit ausdrücklich gesetztem `main_url`
  (`CT_MAIN_URL`) genutzt; `setup init` schlägt sie vor und übernimmt sie nur
  nach Bestätigung (vorher wurde z. B. `meine-gemeinde` → `meine.church.tools`
  angesprochen) (Nr. 1)
- `login_token` und `password` werden mit `github.com/janmz/sconfig/v3`
  rechnergebunden verschlüsselt in `config.json` gespeichert; vorhandene
  Klartextwerte werden beim ersten Start umgewandelt (Nr. 2)
- `config.json` und Dubletten-CSV nur für den aktuellen Benutzer lesbar
  (Windows-ACL ohne Vererbung, sonst `0600`) (Nr. 3)
- Gruppen nur noch über den exakten Namen (Sortiernummern-Markup wird
  ignoriert); öffentliche Gruppenanmeldung nur nach Rückfrage (Nr. 5)
- CSV-Export entschärft Zellen mit `= + - @`/Tab/CR (Formel-Injection); der
  Import stellt die Werte wieder her (Nr. 6)
- `setup token`: Option `--person-id` entfernt (keine Tokens fremder Konten);
  der eigene Token wird nicht mehr ausgegeben, sondern nach Rückfrage
  verschlüsselt gespeichert (Nr. 7)
- Nur `https://` für `base_url`/`main_url` (`http://` nur localhost);
  Weiterleitungen auf anderen Host oder anderes Schema werden abgelehnt (Nr. 9)
- Login-Token wird verdeckt eingegeben; Passworteingabe ohne Sternchen
  (verrät keine Länge mehr) (Nr. 10)
- E-Mail-Adressen in Hinweisen maskiert; Servertexte in Meldungen gekürzt und
  ohne Steuerzeichen (Nr. 11)

### Changed

- Für einen Lauf beigetretene Vorab- und Rechtegruppen werden am Ende von
  `export`/`import` wieder verlassen (auch bei Fehlern); bestehende
  Mitgliedschaften bleiben unverändert (Nr. 4, angepasst)
- Datenschutzhinweis zum Löschen der CSV nach Export und Import; neuer
  README-Abschnitt „Zugangsdaten und Datenschutz“ (Nr. 15)
- Gesamtlimit 512 MiB für alle Seiten einer Listenabfrage; Eingabezeilen auf
  4096 Bytes begrenzt (Nr. 16)
- Go-Mindestversion 1.25 (sconfig); Release-Builds und Lint mit Go 1.27 (Nr. 8)
- CI/Release: Standardrechte `contents: read`, Schreibrecht nur im
  Release-Job, Workflow-Eingaben über `env:` und Tag-Prüfung, Actions per
  Commit-SHA fixiert (Nr. 14)

### Dependencies

- `github.com/janmz/sconfig/v3` v3.0.0 (Ausnahme von der 14-Tage-Regel nach
  Rückfrage), `golang.org/x/sys` v0.29.0 jetzt direkt

## [1.3.1.35] - 2026-11-10 10:00:54

### Fixed

- Passwort-Sitzungen: schreibende REST-Aufrufe (POST/PUT/PATCH/DELETE) senden
  jetzt das CSRF-Token im Header `CSRF-Token` (wie der offizielle
  ChurchTools-JS-Client; nicht bei `GET` und `/api/login`). Neuere
  ChurchTools-Versionen lehnen Schreibzugriffe aus einer Sitzung sonst ab
- Legacy-AJAX-Fallback sendete den falschen Header `X-CSRF-Token`; er nutzt
  jetzt `CSRF-Token` und läuft über die zentrale HTTP-Schicht
  (Größenlimit, HTTP-429-Wiederholung)

## [1.3.0.34] - 2026-11-10 09:52:06

Umsetzung des Code-Reviews vom 2026-11-10 (CR-1 … CR-27).

### Fixed

- Import: `POST /persons/{id}/relationships` sendet jetzt den von der API
  verlangten Body `{otherPersonId, relationshipTypeId}`; HTTP 201 gilt als
  Erfolg. Bisher scheiterte jeder echte Import oder fiel auf die Legacy-API
  zurück (CR-1)
- Legacy-Fallback nur noch bei HTTP 404 und nur mit ausdrücklichem
  `{"status":"success"}` als Erfolg – eine HTML-/Login-Seite mit HTTP 200
  meldet keinen Scheinerfolg mehr (CR-2)
- Dubletten-Erkennung: vertauschte Vor-/Nachnamen („Hans Meier“/„Meier Hans“),
  Initialen („A. Schulz“/„Anna Schulz“) und Teil-Doppelnamen werden jetzt
  tatsächlich verglichen (gemeinsame Vorfilter-Schlüssel) (CR-3)
- Beziehungstyp: `duplicate_relationship_type.name` wird berücksichtigt;
  Reihenfolge ID → Name → ID 8 (nur wenn als Duplikat-Typ erkennbar) →
  Stichwortsuche (CR-4)
- Gruppensuche: exakter Name (ohne Groß-/Kleinschreibung) vor Teilstring;
  Mitgliedschaftsprüfung nur noch mit exaktem Namen („Duplikate (alt)“ zählt
  nicht als „Duplikate“) (CR-5)
- Passworteingabe: Umlaute bleiben erhalten (UTF-8), Leerzeichen am Rand
  werden nicht mehr abgeschnitten (Terminal, Pipe, `CT_PASSWORD`) (CR-6)
- Export: Fehler beim Nachladen einzelner Personen brechen den Export nicht
  mehr ab (Listendaten + Warnung); `delay_ms` wird eingehalten, HTTP 429 einmal
  wiederholt, Fortschritt angezeigt (CR-7)
- Gruppenbeitritt: bestehende Mitgliedschaften werden nicht mehr per PUT
  überschrieben (Kommentar/Enddatum blieben sonst nicht erhalten);
  neuer Beitritt mit `only_add` (CR-8)
- Standortauswahl speichert nur `campus_id`; Zugangsdaten aus Umgebungsvariablen
  landen nicht mehr in `config.json` (CR-9)
- `setup init` fragt vor dem Überschreiben und behält `campus_id`,
  `permission_groups` und `duplicate_relationship_type` (CR-10)
- Menüs und Hinweise gehen nach stderr, `export -o -` liefert reine CSV (CR-11)
- Import prüft die CSV vorab: doppelte Zeilen und Personen in mehreren DupIDs
  werden mit Zeilennummern gemeldet, bevor etwas geändert wird (CR-12)
- Erneute Anmeldung nach Sitzungsablauf auch bei Benutzername/Passwort (CR-13)
- Setup über Pipe: Passwort wird aus demselben Eingabepuffer gelesen; letzte
  Zeile ohne Zeilenumbruch wird akzeptiert (CR-15)
- Import-Zusammenfassung zählt bei Fehlern bereits angelegte Beziehungen mit
  („x von y bereits angelegt“) (CR-16)
- DupIDs sind bei gleichem Datenbestand stabil (CR-17)
- Erstellungsdatum wird in Ortszeit ausgegeben (vorher UTC, Vortag möglich) (CR-18)
- `--campus`: Umlaute/Akzente werden umgeschrieben (ü→ue, é→e), Ziffern bleiben
  erhalten, exakter Name hat Vorrang (CR-19)

### Changed

- `export` überschreibt eine vorhandene Ziel-Datei nur noch mit `--force` (CR-14)
- E-Mail-Adressen, die mehr als 5 Personen nutzen (z. B. `info@…`), werden
  nicht mehr zur Verknüpfung verwendet, sondern als Hinweis gemeldet; sehr große
  DupIDs erzeugen eine Warnung (CR-20)
- Import löst die Gruppe „Duplikate“ einmal vorab auf (Abbruch vor jeder
  Änderung, wenn sie fehlt) und prüft jede Beziehung nur noch einmal (CR-22)
- Vorab-Gruppen: abgelehnte Gruppen lösen keine weiteren Durchläufe mehr aus (CR-23)
- Zentrale HTTP-Schicht im Client: Größenlimits (API 32 MiB, CSV 100 MiB,
  Konfiguration 1 MiB), Fehlertexte auf ca. 2 KB gekürzt (CR-21)
- Defaults (Beziehungstyp 8, Gruppe „Duplikate“, `delay_ms` 500) zentral in
  `internal/config` (CR-25)
- CI: `gofmt`-Prüfung und `staticcheck` v0.8.2 (eigener Lint-Job mit Go 1.26);
  `.gitattributes` erzwingt LF für Go-Dateien (CR-26)

### Removed

- Ungenutzter Code aus dem Fork (Personenfilter nach Gruppe/Status/IDs,
  `promptFilterMode`, `applyDefaultCampus`, doppelte Typ-Dekodierung) (CR-24)

### Tests

- Neue Tests für Beziehungs-POST, Legacy-Fallback, vertauschte Namen/Initialen,
  stabile DupIDs, Sammel-E-Mails, Gruppensuche, `only_add`, Vorab-Gruppen,
  Teilfehler im Import, Zusammenfassung, CSV-Prüfung, Standortsuche,
  `SaveCampusID`, Re-Login per Passwort, HTTP 429, Datumszone, Passwort-UTF-8;
  Paginierungstest mit 6 statt 10 001 Anfragen (CR-27)

## [1.2.4.32] - 2026-08-19 22:17:34

### Fixed

- CLI: Fehlende Dateinamen bei `-o`/`-f` (z. B. `export -o -i` oder `-o --help`)
  liefern dieselbe vollständige Meldung wie unbekannte Optionen (`Error:` +
  `Usage` + `Flags`), ohne doppelte Fehlerzeile am Ende

## [1.2.3.30] - 2026-06-20 13:35:37

### Changed

- Dubletten-CSV: Spalte `Einwilligungsdatum` durch `Einladungsstatus` ersetzt
  (Werte `NEU`, `Eingeladen`, `Registriert` wie in ChurchToolsInvite aus
  `invitationStatus` und Kontometadaten)
- Personenanreicherung übernimmt Einladungs-/Kontofelder für den Export

### Removed

- `ConsentDate()` und die bisher leere Spalte `Einwilligungsdatum`

## [1.2.2.28] - 2026-06-20 13:23:29

### Fixed

- `export --campus`: Standortnamen werden für die Teilstring-Suche wie der
  Suchbegriff normalisiert (Bindestriche/Leerzeichen irrelevant)
- Ausgabe Vorab-Gruppen: einheitlich „Bereits Mitglied“

### Changed

- README / README.de.md: Hinweis zu Datei-Optionen (`-o` / `-f`) bei fehlendem
  Dateinamen
- `EnsurePreJoinGroups`: ungenutzte `refreshSession`-Hilfsfunktion entfernt

### Added

- Test: `ListPersonGroups` nutzt `groupId` statt Mitgliedschafts-`id`

## [1.2.2.27] - 2026-06-20 13:10:34

### Fixed

- `pre_join_groups`: noch unsichtbare Gruppen blockieren nicht mehr die restliche
  Liste; nach erfolgreichem Beitritt werden vorher nicht gefundene Gruppen erneut
  versucht
- Gruppenbeitritt: Bei `403` auf `PUT /groups/{id}/members/{personId}` wird die
  Web-GUI-Anmeldung über `GET /publicgroups/{id}/form`, `POST .../token` und
  `POST .../signup` verwendet
- `ListPersonGroups`: `groupId` statt Mitgliedschafts-`id` für die Gruppen-ID
- `export --output` / `import --csv`: Werte wie `-i` oder `--dry-run` nach `-o`/`-f`
  werden als fehlender Dateiname erkannt

### Changed

- Standard `pre_join_groups`: `ChurchTools Verwaltung,Gruppen Administration,ChurchTools Admin,Personen Administration,Personen verwalten`
- `export --campus-id` → `--campus` (ID, eindeutiger Namens-Teilstring oder `all`);
  `--all-campuses` als Alias für `--campus all`
- Ohne `--campus`: Standort des angemeldeten Nutzers bzw. `campus_id` aus der
  Config; ohne Zuordnung alle Standorte
- `README.md` / `README.de.md`: `--campus`, `--all-campuses`, Dateinamen-Prüfung

## [1.2.0.25] - 2026-06-19 11:18:05

### Added

- `export --campus-id all` (auch `alle`, `*`): Dublettensuche ohne
  Standortfilter über den gesamten Bestand

### Fixed

- `export -i` / `--interactive`: zeigt immer eine Standortauswahl (inkl.
  „Alle Standorte“), auch wenn am Benutzer bereits ein Standort hinterlegt ist

### Changed

- Standard-Vorab-Gruppe „Personen verwaltung“ → „Personen verwalten“
- README / README.de.md: Dubletten-Matching (Regeln, Normalisierung,
  Transitivität, Ausnahmen) ausführlicher beschrieben

## [1.1.3.23] - 2026-06-19 11:01:07

### Added

- Vorab-Gruppen: vor Export/Import werden konfigurierte Gruppen in fester
  Reihenfolge beigetreten, wenn noch nicht Mitglied (`pre_join_groups` in
  `config.json`, Standard: ChurchTools Admin, ChurchTools Verwaltung, Personen
  Administration, Personen verwalten)
- `setup init` fragt die kommagetrennte Gruppenliste ab
- Flags `--skip-pre-join-groups` für `export` und `import`
- Umgebungsvariable `CT_PRE_JOIN_GROUPS`

## [1.1.2.21] - 2026-06-18 23:08:23

### Changed

- Standard-Beziehungstyp für Duplikate ist **ID 8** (nicht mehr heuristische
  Auswahl z. B. ID 5 „noduplicate.name“); `config.example.json` entsprechend

### Fixed

- Import/Dry-Run: bestehende Duplikat-Beziehungen zuverlässiger erkannt (weitere
  API-Feldvarianten, Fallback über Beziehungsnamen)
- Dry-Run: prüft Gruppenmitgliedschaft „Duplikate“; bereits vorgemerkte DupIDs
  zählen in der Zusammenfassung als „übersprungen“
- Import-/Dry-Run-Zusammenfassung: „verknüpft“ nur bei tatsächlich neu angelegten
  Beziehungen; „bereits vorhanden“ und Gruppenstatus in den Meldungen

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
