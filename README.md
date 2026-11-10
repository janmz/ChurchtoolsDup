# ChurchTools_Dup

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)](https://golang.org)
[![Release](https://img.shields.io/badge/Release-GitHub-0077B6)](https://github.com/janmz/ChurchToolsDup/releases)
[![License: MIT (Modified)](https://img.shields.io/badge/License-MIT--Modified-blue.svg)](LICENSE)
[![Support: CFI-Kinderhilfe](https://img.shields.io/badge/Support-CFI--Kinderhilfe-0077B6?logo=heart)](https://cfi-kinderhilfe.de/jetzt-spenden?q=VAYACTDUP)
[![Build Status](https://github.com/janmz/ChurchToolsDup/actions/workflows/ci.yml/badge.svg)](https://github.com/janmz/ChurchToolsDup/actions/workflows/ci.yml)

<p align="center">
  <a href="README.de.md"><img src="https://img.shields.io/badge/🇩🇪-Deutsch-555?style=for-the-badge" alt="Deutsch"></a>
  <img src="https://img.shields.io/badge/🇺🇸-English-0077B6?style=for-the-badge" alt="English (current)">
</p>

**churchtools-dup** is a Go CLI to **detect, review, and mark duplicates** in
ChurchTools:

- Find duplicate persons for a campus across the full person list and export CSV
- Clean up the CSV manually (remove false positives; keep row order per DupID)
- Link remaining pairs in relationship management and add primaries to group
  “Duplikate”
- Skip pairs that already have a duplicate relationship in ChurchTools

## Features

### Detection (`export`)

Search runs across the **full person database**. By default, a group is exported
when **at least one person** belongs to the selected campus (`--campus ID` or name
substring, or interactive). With `--campus all` or `--all-campuses`, the campus
filter is skipped and **all** duplicates in the database are exported. Without
`--campus`, your user's campus or `campus_id` from config is used; if none is
set, all campuses apply.

#### Pipeline

1. **E-mail phase:** persons with the same (normalized) e-mail are linked.
2. **Name phase:** additional pairs are checked via first name, last name, city,
   and street.
3. **Transitivity:** overlapping pairs merge into **one DupID** (union-find). If
   A↔B and B↔C match, A, B, and C share one group.

#### When two persons count as duplicates

| Rule | Condition |
| --- | --- |
| E-mail | Same e-mail address |
| Address + first name | Same street, same city (fuzzy), matching first names |
| Name | Same first and last name, including **swapped** first/last |

**E-mail exception:** same e-mail, same address, **different** first names on
the **same campus** are treated as a shared mailbox (couple, etc.) and are
**not** linked.

**Shared addresses:** an e-mail address used by more than 5 persons (e.g.
`info@…` or a placeholder) is not used for linking; the export lists it as a
hint on stderr. DupIDs with 10 or more persons produce a warning.

#### Normalization and fuzzy matching

- **E-mail:** lowercased, trimmed
- **Names:** lowercased, hyphens as spaces, punctuation ignored
- **First names:** exact match, subset (`Jan Oliver` ↔ `Jan`), or initials
  (`Jan O.` ↔ `Jan Oliver`)
- **Last names:** exact match or subset for compound names (`Müller-Schmidt`
  ↔ `Müller`)
- **City:** main locality and suffix parsed separately; variants like
  `Frankfurt`, `Frankfurt am Main`, `Frankfurt/M.` match; different places
  (`Frankfurt a.d. Oder` vs. `Frankfurt am Main`) do not
- **Street:** `ß`→`ss`, `Str.`/`Straße`/`Strasse` unified, punctuation and
  extra spaces ignored (`Klarstr.` = `Klarstraße`)

Candidates are pre-filtered with **blocking keys** (first-name initial + city +
street, or first-name initial + each last-name token, also with first and last
name swapped) so not every person is compared to every other. DupIDs are
numbered by the smallest person ID and stay stable for the same data.

#### What is not exported

- Singletons with no partner in the group
- Groups where **no** person belongs to the selected campus
- Couples/shared mailbox (see exception above)
- Links that rely only on an e-mail address shared by more than 5 persons

### Import (`import`)

- First row per **DupID** becomes the primary person
- Other rows in the same DupID are linked as duplicates
- Existing duplicate relationships are detected and skipped
- Primary is added to group **Duplikate** (optional)

### Other commands

- `whoami` – logged-in user, campus, group memberships
- `relationship-types` – list relationship type IDs and names
- `setup` – configuration, connection test, token, permission hints

## Requirements

- Go 1.25+ (build from source)
- ChurchTools access (login token or username/password)
- **Export:** **export data** permission (group “Personen exportieren”)
- **Import:** **edit relations** permission (group “Personen Administration” or
  “Personen bearbeiten”)

## Installation

### Binary download

[Releases](https://github.com/janmz/ChurchToolsDup/releases)

### Go Install

```bash
go install github.com/janmz/churchtools-dup@latest
```

### Build from Source

```bash
git clone https://github.com/janmz/ChurchToolsDup.git
cd ChurchToolsDup
go build -o churchtools-dup .
```

## Usage

### Quick Start

```bash
cp config.example.json config.json
# edit config.json or run setup init

./churchtools-dup setup test
./churchtools-dup export -o dup.csv -i
```

Edit the CSV (remove wrong rows; first row per DupID stays primary), then:

```bash
./churchtools-dup import -f dup.csv --dry-run
./churchtools-dup import -f dup.csv
```

Global option: `-c config.json` for an alternate config path.

### Path flags (`-o` / `-f`)

If the file name is missing and another flag follows directly (e.g.
`export -o -i` or `import -f --dry-run`), the tool reports a clear error
instead of treating the option as a file path.

### Relationship type

```bash
./churchtools-dup relationship-types
```

Set the duplicate type ID in `config.json`, e.g.:

```json
"duplicate_relationship_type": { "id": 8 }
```

## Configuration

| Variable / field | Description |
| --- | --- |
| `CT_BASE_URL` | Instance name or full URL (`https://` only; `http://` only for localhost) |
| `main_url` / `CT_MAIN_URL` | Main (central) instance for a sub-instance login; only used when set explicitly |
| `CT_LOGIN_TOKEN` | API login token |
| `CT_USERNAME` / `CT_PASSWORD` | Alternative to token |
| `delay_ms` | Delay between API calls in ms (default: 500) |
| `campus_id` | Default campus when user has none |
| `pre_join_groups` | Comma-separated groups to join in order before export/import (default in `config.example.json`; set `-` to disable) |
| `CT_PRE_JOIN_GROUPS` | Environment override for `pre_join_groups` |
| `permission_groups.edit_persons` | Group for import rights |
| `permission_groups.export_persons` | Group for export |
| `duplicate_relationship_type.id` | Relationship type ID (must exist when set) |
| `duplicate_relationship_type.name` | Relationship type name (used when no ID is set) |

Without ID and name, type **8** is used if its name looks like a duplicate
type (Duplikat/Dublette/Duplicate); otherwise the type is found by these
keywords.

## CSV format (duplicates)

```csv
DupID,ID,Vorname,Nachname,E-Mail,Straße,Stadt,Standort,Erstellungsdatum,Einladungsstatus
```

- **DupID** – group id; rows with the same DupID belong together
- **ID** – ChurchTools person id (required)
- **Einladungsstatus** – `NEU`, `Eingeladen`, or `Registriert` (from ChurchTools
  `invitationStatus` and account metadata)
- Delete rows to discard matches; **first row** per DupID is the import primary

### Import output

- **OK** – at least one new duplicate link created
- **ÜBERSPRUNGEN** (skipped) – all links already existed or fewer than two persons
- **FEHLER** (error) – API or permission failure

Summary includes counts for linked vs. already existing.

## Commands

| Command | Purpose |
| --- | --- |
| `setup init` | Create `config.json` interactively |
| `setup test` | Test login and connection |
| `setup token` | Fetch your own login token and store it encrypted in `config.json` |
| `setup permissions` | Permission hints for export/import |
| `whoami` | User, campus, groups, instance URL |
| `relationship-types` | List relationship types with ID and name |
| `export -o FILE` | Export duplicate CSV (default `duplikate.csv`, `-` = stdout; hints go to stderr) |
| `export --force` | Overwrite an existing output file (otherwise the export stops) |
| `export -i` | Choose campus interactively (always prompts, incl. “All campuses”) |
| `export --campus VALUE` | Campus by ID or unique name substring; only needed for a different campus |
| `export --campus all` | Duplicate search across all campuses (no filter) |
| `export --all-campuses` | Alias for `--campus all` |
| `import -f FILE` | Import edited CSV |
| `import -f FILE --dry-run` | Simulate without changes |
| `import -f FILE --skip-group-add` | Do not add to group “Duplikate” |
| `import -f FILE --skip-permission-request` | Skip group membership requests |
| `export --skip-pre-join-groups` | Skip pre-join groups before export |
| `import -f FILE --skip-pre-join-groups` | Skip pre-join groups before import |

## Development

See [TESTING.md](TESTING.md). Windows release builds embed `vaya.ico` via
[go-winres](https://github.com/tc-hib/go-winres).

```bash
scripts/ci.sh      # or scripts/ci.ps1: gofmt, go vet, staticcheck, tests, build
go test ./...
go build -o churchtools-dup .
```

`staticcheck` (v0.8.2) needs Go 1.26 or newer; on older toolchains set
`SKIP_STATICCHECK=1`.

## Credentials and privacy

- **Encrypted secrets:** `login_token` and `password` in `config.json` are
  encrypted with [sconfig](https://github.com/janmz/sconfig) on first start
  (`login_token_secure`, `secure_password`; the plaintext field then shows a
  marker). The encryption is bound to this computer – on another PC enter the
  secrets again. To change a secret, write the new plaintext value into
  `login_token` / `password`.
- `config.json` must be located below the program directory, the current
  working directory or your home directory (sconfig rule).
- `config.json` and exported CSV files are readable only by the current user
  (Windows ACL, `0600` elsewhere). Environment variables (`CT_*`) are never
  written to the file.
- **No fallback to guessed hosts:** credentials are only sent to `base_url`
  and, if configured, `main_url`. `setup init` suggests a main instance for
  names like `church-site` but uses it only after confirmation. Redirects to
  another host or scheme are refused.
- **Temporary groups:** groups from `pre_join_groups` and permission groups
  that the tool joins for a run are left again at the end; memberships you
  had before stay untouched. Joining via the public group signup form is
  only done after confirmation.
- `setup token` stores only **your own** token, encrypted; it is never printed.
- The CSV contains personal data: delete it after review and import (the
  tool reminds you). Cells starting with `= + - @` are prefixed with `'`
  against formula injection; the import removes the prefix again.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

Modified MIT license – see [LICENSE](LICENSE). Credit **Jan Neuhaus, VAYA
Consulting**. No warranty.

## Support

[CFI-Kinderhilfe](https://cfi-kinderhilfe.de/jetzt-spenden?q=VAYACTDUP)

## Contact

**Author:** Jan Neuhaus, VAYA Consulting –
[VAYA Consulting](https://vaya-consulting.de/development?q=GITHUB)

**Repository:** [https://github.com/janmz/ChurchToolsDup](https://github.com/janmz/ChurchToolsDup)

## Changelog

See [Changelog.md](Changelog.md).
