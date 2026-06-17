# ChurchTools_Dup

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://golang.org)
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

Searches the **full person database**; at least one person per group must
belong to the selected campus. Rules (simplified):

1. Same e-mail (except shared-mailbox couples: same e-mail and address,
   different first name)
2. Same first name + city + street (fuzzy matching)
3. Same first name + last name (including swapped or variant names)

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

- Go 1.22+ (build from source)
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
| `CT_BASE_URL` | Instance name or full URL |
| `CT_LOGIN_TOKEN` | API login token |
| `CT_USERNAME` / `CT_PASSWORD` | Alternative to token |
| `delay_ms` | Delay between API calls in ms (default: 500) |
| `campus_id` | Default campus when user has none |
| `permission_groups.edit_persons` | Group for import rights |
| `permission_groups.export_persons` | Group for export |
| `duplicate_relationship_type.id` | Relationship type ID (optional) |
| `duplicate_relationship_type.name` | Relationship type name (optional) |

## CSV format (duplicates)

```csv
DupID,ID,Vorname,Nachname,E-Mail,Straße,Stadt,Standort,Erstellungsdatum,Einwilligungsdatum
```

- **DupID** – group id; rows with the same DupID belong together
- **ID** – ChurchTools person id (required)
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
| `setup token` | Show login token |
| `setup permissions` | Permission hints for export/import |
| `whoami` | User, campus, groups, instance URL |
| `relationship-types` | List relationship types with ID and name |
| `export -o FILE` | Export duplicate CSV (default `duplikate.csv`) |
| `export -i` | Choose campus interactively |
| `export --campus-id ID` | Campus for duplicate search |
| `import -f FILE` | Import edited CSV |
| `import -f FILE --dry-run` | Simulate without changes |
| `import -f FILE --skip-group-add` | Do not add to group “Duplikate” |
| `import -f FILE --skip-permission-request` | Skip group membership requests |

## Development

See [TESTING.md](TESTING.md). Windows release builds embed `vaya.ico` via
[go-winres](https://github.com/tc-hib/go-winres).

```bash
go test ./...
go vet ./...
go build -o churchtools-dup .
```

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
