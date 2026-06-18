package duplicates

import (
	"fmt"
	"strings"
	"time"

	churchtools "github.com/janmz/churchtools-dup/internal/churchtools"
	csvfile "github.com/janmz/churchtools-dup/internal/csvfile"
)

// ImportResult describes the outcome for one duplicate group.
type ImportResult struct {
	DupID         int
	Primary       int
	Success       bool
	Linked        int
	AlreadyLinked int
	Message       string
}

// ImportOptions controls duplicate import behaviour.
type ImportOptions struct {
	DryRun bool
	Delay  time.Duration
}

// ImportRunner applies duplicate merge preparation from a CSV file.
type ImportRunner struct {
	Client       *churchtools.Client
	RelType      churchtools.RelationshipType
	GroupName    string
	SkipGroupAdd bool
}

// Run imports duplicate groups: link persons and add primary to group "Duplikate".
func (r ImportRunner) Run(groups [][]csvfile.DupEntry, opts ImportOptions) ([]ImportResult, error) {
	results := make([]ImportResult, 0, len(groups))

	for i, group := range groups {
		if i > 0 && opts.Delay > 0 {
			time.Sleep(opts.Delay)
		}

		result := ImportResult{DupID: group[0].DupID}
		if len(group) < 2 {
			result.Message = "übersprungen: weniger als zwei Personen"
			results = append(results, result)
			continue
		}

		primary := group[0].PersonID
		result.Primary = primary

		if opts.DryRun {
			others := personIDs(group[1:])
			alreadyLinked, err := r.existingDuplicateIDs(primary, group[1:])
			if err != nil {
				result.Message = fmt.Sprintf("Beziehungen prüfen fehlgeschlagen: %v", err)
				results = append(results, result)
				continue
			}

			inGroup := false
			if !r.SkipGroupAdd {
				inGroup, err = r.Client.PersonIsInGroup(primary, r.groupName())
				if err != nil {
					result.Message = fmt.Sprintf("Gruppenmitgliedschaft prüfen fehlgeschlagen: %v", err)
					results = append(results, result)
					continue
				}
			}

			result.Success = true
			result.Linked = len(others) - len(alreadyLinked)
			result.AlreadyLinked = len(alreadyLinked)
			result.Message = formatDryRunMessage(primary, others, alreadyLinked, r.groupName(), inGroup, r.SkipGroupAdd)
			results = append(results, result)
			continue
		}

		linked := 0
		alreadyLinked := 0
		for _, entry := range group[1:] {
			exists, err := r.Client.DuplicateRelationshipExists(primary, entry.PersonID, r.RelType)
			if err != nil {
				result.Message = fmt.Sprintf(
					"Beziehung %d -> %d prüfen fehlgeschlagen: %v",
					primary,
					entry.PersonID,
					err,
				)
				results = append(results, result)
				goto nextGroup
			}
			if exists {
				alreadyLinked++
				continue
			}

			if err := r.Client.LinkAsDuplicate(primary, entry.PersonID, r.RelType); err != nil {
				result.Message = fmt.Sprintf(
					"Beziehung %d -> %d fehlgeschlagen: %v",
					primary,
					entry.PersonID,
					err,
				)
				results = append(results, result)
				goto nextGroup
			}
			linked++
			if opts.Delay > 0 {
				time.Sleep(opts.Delay)
			}
		}

		if linked == 0 && alreadyLinked == 0 {
			result.Message = "keine Beziehungen angelegt"
			results = append(results, result)
			goto nextGroup
		}

		if !r.SkipGroupAdd {
			membership, err := r.Client.EnsurePersonInGroup(r.groupName(), primary)
			if err != nil {
				result.Message = fmt.Sprintf("Gruppe %q: %v", r.groupName(), err)
				results = append(results, result)
				continue
			}
			if membership.Status == churchtools.MembershipDenied {
				msg := membership.Message
				if msg == "" {
					msg = "Mitgliedschaft abgelehnt"
				}
				result.Message = fmt.Sprintf("Gruppe %q: %s", r.groupName(), msg)
				results = append(results, result)
				continue
			}
		}

		result.Success = true
		result.Linked = linked
		result.AlreadyLinked = alreadyLinked
		result.Message = formatImportMessage(primary, linked, alreadyLinked, len(group)-1)
		results = append(results, result)
	nextGroup:
	}

	return results, nil
}

func (r ImportRunner) groupName() string {
	if strings.TrimSpace(r.GroupName) == "" {
		return churchtools.DuplicateGroupName
	}
	return r.GroupName
}

func personIDs(entries []csvfile.DupEntry) []int {
	ids := make([]int, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.PersonID)
	}
	return ids
}

func formatIDList(ids []int) string {
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, fmt.Sprintf("%d", id))
	}
	return strings.Join(parts, ", ")
}

func (r ImportRunner) existingDuplicateIDs(primary int, entries []csvfile.DupEntry) ([]int, error) {
	alreadyLinked := make([]int, 0, len(entries))
	for _, entry := range entries {
		exists, err := r.Client.DuplicateRelationshipExists(primary, entry.PersonID, r.RelType)
		if err != nil {
			return nil, err
		}
		if exists {
			alreadyLinked = append(alreadyLinked, entry.PersonID)
		}
	}
	return alreadyLinked, nil
}

func formatDryRunMessage(primary int, others, alreadyLinked []int, groupName string, inGroup, skipGroupAdd bool) string {
	toLink := make([]int, 0, len(others))
	linkedSet := make(map[int]struct{}, len(alreadyLinked))
	for _, id := range alreadyLinked {
		linkedSet[id] = struct{}{}
	}
	for _, id := range others {
		if _, ok := linkedSet[id]; ok {
			continue
		}
		toLink = append(toLink, id)
	}

	groupNote := ""
	if !skipGroupAdd {
		if inGroup {
			groupNote = fmt.Sprintf("; Person %d bereits in Gruppe %q", primary, groupName)
		} else {
			groupNote = fmt.Sprintf("; Person %d würde in Gruppe %q aufgenommen", primary, groupName)
		}
	}

	switch {
	case len(alreadyLinked) > 0 && len(toLink) == 0:
		return fmt.Sprintf(
			"Dry-Run: Person %d ist bereits mit %s verknüpft%s",
			primary,
			formatIDList(alreadyLinked),
			groupNote,
		)
	case len(alreadyLinked) > 0:
		return fmt.Sprintf(
			"Dry-Run: Person %d mit %s verknüpfen (%s bereits verknüpft)%s",
			primary,
			formatIDList(toLink),
			formatIDList(alreadyLinked),
			groupNote,
		)
	default:
		return fmt.Sprintf(
			"Dry-Run: würde Person %d mit %s verknüpfen%s",
			primary,
			formatIDList(toLink),
			groupNote,
		)
	}
}

func formatImportMessage(primary, linked, alreadyLinked, total int) string {
	switch {
	case linked > 0 && alreadyLinked > 0:
		return fmt.Sprintf(
			"Person %d: %d Beziehung(en) angelegt, %d bereits vorhanden",
			primary,
			linked,
			alreadyLinked,
		)
	case alreadyLinked == total:
		return fmt.Sprintf(
			"Person %d: alle %d Duplikat-Beziehung(en) bereits vorhanden",
			primary,
			alreadyLinked,
		)
	default:
		return fmt.Sprintf(
			"Person %d mit %d Dublette(n) verknüpft",
			primary,
			linked,
		)
	}
}

// PrintImportSummary writes import results to stdout.
func PrintImportSummary(results []ImportResult) {
	ok, skipped, failed := 0, 0, 0
	linkedTotal, existingTotal := 0, 0

	for _, result := range results {
		linkedTotal += result.Linked
		existingTotal += result.AlreadyLinked

		switch ImportResultStatus(result) {
		case "ok":
			ok++
			fmt.Printf("OK DupID %d: %s\n", result.DupID, result.Message)
		case "skipped":
			skipped++
			fmt.Printf("ÜBERSPRUNGEN DupID %d: %s\n", result.DupID, result.Message)
		default:
			failed++
			fmt.Printf("FEHLER DupID %d: %s\n", result.DupID, result.Message)
		}
	}

	fmt.Printf(
		"\nZusammenfassung: %d OK, %d übersprungen, %d Fehler (%d verknüpft, %d bereits vorhanden)\n",
		ok, skipped, failed, linkedTotal, existingTotal,
	)
}

func ImportResultStatus(result ImportResult) string {
	if !result.Success {
		if strings.Contains(result.Message, "übersprungen") {
			return "skipped"
		}
		return "failed"
	}
	if result.Linked == 0 && result.AlreadyLinked > 0 {
		return "skipped"
	}
	return "ok"
}

// IsSkippedImportResult reports outcomes that are not import failures.
func IsSkippedImportResult(result ImportResult) bool {
	return ImportResultStatus(result) == "skipped"
}

