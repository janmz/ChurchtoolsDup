package duplicates

import (
	"fmt"
	"io"
	"strings"
	"time"

	churchtools "github.com/janmz/churchtools-dup/internal/churchtools"
	config "github.com/janmz/churchtools-dup/internal/config"
	csvfile "github.com/janmz/churchtools-dup/internal/csvfile"
)

// ImportResult describes the outcome for one duplicate group.
type ImportResult struct {
	DupID         int
	Primary       int
	Success       bool
	Linked        int  // relationships created (or, in a dry run, to be created)
	AlreadyLinked int  // relationships that existed before
	Vorgemerkt    bool // dry run: primary already in the duplicate group
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

// Run imports duplicate groups: link persons and add primary to the duplicate
// group. The group is resolved once before any change; when it cannot be
// found, Run fails without creating relationships.
func (r ImportRunner) Run(groups [][]csvfile.DupEntry, opts ImportOptions) ([]ImportResult, error) {
	var duplicateGroup churchtools.Group
	if !opts.DryRun && !r.SkipGroupAdd {
		group, err := r.Client.FindGroupByName(r.groupName())
		if err != nil {
			return nil, fmt.Errorf("Gruppe %q: %w (oder --skip-group-add verwenden)", r.groupName(), err)
		}
		duplicateGroup = group
	}

	results := make([]ImportResult, 0, len(groups))
	for i, group := range groups {
		if i > 0 && opts.Delay > 0 {
			time.Sleep(opts.Delay)
		}

		if len(group) < 2 {
			results = append(results, ImportResult{
				DupID:   group[0].DupID,
				Message: "übersprungen: weniger als zwei Personen",
			})
			continue
		}

		if opts.DryRun {
			results = append(results, r.dryRunGroup(group))
		} else {
			results = append(results, r.importGroup(group, duplicateGroup, opts.Delay))
		}
	}

	return results, nil
}

// dryRunGroup reports what importGroup would do without changing anything.
func (r ImportRunner) dryRunGroup(group []csvfile.DupEntry) ImportResult {
	primary := group[0].PersonID
	result := ImportResult{DupID: group[0].DupID, Primary: primary}

	others := personIDs(group[1:])
	alreadyLinked, err := r.existingDuplicateIDs(primary, group[1:])
	if err != nil {
		result.Message = fmt.Sprintf("Beziehungen prüfen fehlgeschlagen: %v", err)
		return result
	}

	inGroup := false
	if !r.SkipGroupAdd {
		inGroup, err = r.Client.PersonIsInGroup(primary, r.groupName())
		if err != nil {
			result.Message = fmt.Sprintf("Gruppenmitgliedschaft prüfen fehlgeschlagen: %v", err)
			return result
		}
	}

	result.Success = true
	result.Linked = len(others) - len(alreadyLinked)
	result.AlreadyLinked = len(alreadyLinked)
	result.Vorgemerkt = inGroup && !r.SkipGroupAdd
	result.Message = formatDryRunMessage(primary, others, alreadyLinked, r.groupName(), inGroup, r.SkipGroupAdd)
	return result
}

// importGroup links all persons of one DupID to the primary and adds the
// primary to the duplicate group. Counts are kept on failure so the summary
// reflects relationships that were already created.
func (r ImportRunner) importGroup(group []csvfile.DupEntry, duplicateGroup churchtools.Group, delay time.Duration) ImportResult {
	primary := group[0].PersonID
	result := ImportResult{DupID: group[0].DupID, Primary: primary}
	total := len(group) - 1

	fail := func(format string, args ...any) ImportResult {
		result.Message = fmt.Sprintf(format, args...)
		if result.Linked > 0 {
			result.Message += fmt.Sprintf(" (%d von %d Beziehung(en) bereits angelegt)", result.Linked, total)
		}
		return result
	}

	for _, entry := range group[1:] {
		exists, err := r.Client.DuplicateRelationshipExists(primary, entry.PersonID, r.RelType)
		if err != nil {
			return fail("Beziehung %d -> %d prüfen fehlgeschlagen: %v", primary, entry.PersonID, err)
		}
		if exists {
			result.AlreadyLinked++
			continue
		}

		if err := r.Client.LinkAsDuplicate(primary, entry.PersonID, r.RelType); err != nil {
			return fail("Beziehung %d -> %d fehlgeschlagen: %v", primary, entry.PersonID, err)
		}
		result.Linked++
		if delay > 0 {
			time.Sleep(delay)
		}
	}

	inGroup := false
	if !r.SkipGroupAdd {
		membership, err := r.Client.RequestGroupMembership(duplicateGroup.ID, primary)
		if err != nil {
			return fail("Gruppe %q: %v", r.groupName(), err)
		}
		if membership.Status == churchtools.MembershipDenied {
			msg := membership.Message
			if msg == "" {
				msg = "Mitgliedschaft abgelehnt"
			}
			return fail("Gruppe %q: %s", r.groupName(), msg)
		}
		inGroup = membership.Status == churchtools.MembershipActive
	}

	result.Success = true
	result.Message = formatImportMessage(primary, result.Linked, result.AlreadyLinked, total, inGroup, r.groupName(), r.SkipGroupAdd)
	return result
}

func (r ImportRunner) groupName() string {
	if strings.TrimSpace(r.GroupName) == "" {
		return config.DefaultDuplicateGroupName
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

func formatImportMessage(primary, linked, alreadyLinked, total int, inGroup bool, groupName string, skipGroupAdd bool) string {
	groupNote := ""
	if !skipGroupAdd {
		if inGroup {
			groupNote = fmt.Sprintf("; Person %d in Gruppe %q", primary, groupName)
		} else {
			groupNote = fmt.Sprintf("; Person %d nicht in Gruppe %q", primary, groupName)
		}
	}

	switch {
	case linked > 0 && alreadyLinked > 0:
		return fmt.Sprintf(
			"Person %d: %d Beziehung(en) angelegt, %d bereits vorhanden%s",
			primary,
			linked,
			alreadyLinked,
			groupNote,
		)
	case alreadyLinked == total:
		return fmt.Sprintf(
			"Person %d: alle %d Duplikat-Beziehung(en) bereits vorhanden%s",
			primary,
			alreadyLinked,
			groupNote,
		)
	default:
		return fmt.Sprintf(
			"Person %d mit %d Dublette(n) verknüpft%s",
			primary,
			linked,
			groupNote,
		)
	}
}

// ImportSummary aggregates import results.
type ImportSummary struct {
	OK       int
	Skipped  int
	Failed   int
	Linked   int // relationships created, including those of failed groups
	Existing int // relationships that already existed
}

// SummarizeImport counts results by status.
func SummarizeImport(results []ImportResult) ImportSummary {
	var summary ImportSummary
	for _, result := range results {
		switch ImportResultStatus(result) {
		case "ok":
			summary.OK++
		case "skipped":
			summary.Skipped++
		default:
			summary.Failed++
		}
		summary.Linked += result.Linked
		summary.Existing += result.AlreadyLinked
	}
	return summary
}

// PrintImportSummary writes one line per result and the summary to w.
func PrintImportSummary(w io.Writer, results []ImportResult) ImportSummary {
	for _, result := range results {
		label := map[string]string{"ok": "OK", "skipped": "ÜBERSPRUNGEN"}[ImportResultStatus(result)]
		if label == "" {
			label = "FEHLER"
		}
		fmt.Fprintf(w, "%s DupID %d: %s\n", label, result.DupID, result.Message)
	}

	summary := SummarizeImport(results)
	fmt.Fprintf(w,
		"\nZusammenfassung: %d OK, %d übersprungen, %d Fehler (%d verknüpft, %d bereits vorhanden)\n",
		summary.OK, summary.Skipped, summary.Failed, summary.Linked, summary.Existing,
	)
	return summary
}

// ImportResultStatus classifies a result as "ok", "skipped" or "failed".
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
	if result.Vorgemerkt {
		return "skipped"
	}
	return "ok"
}

// IsSkippedImportResult reports outcomes that are not import failures.
func IsSkippedImportResult(result ImportResult) bool {
	return ImportResultStatus(result) == "skipped"
}
