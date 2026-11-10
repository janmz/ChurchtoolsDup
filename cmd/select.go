package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/janmz/churchtools-dup/internal/termio"
)

// stdinReader is the single buffered reader for all interactive input, so
// lines buffered by one prompt are not lost for the next (piped input).
var stdinReader = bufio.NewReader(os.Stdin)

// readLine prints prompt to stderr and returns the next stdin line without
// the line break. A last line without line break is accepted.
func readLine(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	return termio.ReadLine(stdinReader, termio.MaxLineBytes)
}

type menuItem struct {
	id   int
	name string
}

// promptMenu shows a numbered menu on stderr (stdout may carry CSV data) and
// returns the chosen item's id.
func promptMenu(title string, items []menuItem, allowSkip bool) (int, error) {
	if len(items) == 0 {
		return 0, fmt.Errorf("keine Einträge für %s", title)
	}

	fmt.Fprintf(os.Stderr, "\n%s:\n", title)
	for i, item := range items {
		fmt.Fprintf(os.Stderr, "  [%d] %s (ID %d)\n", i+1, item.name, item.id)
	}
	if allowSkip {
		fmt.Fprintln(os.Stderr, "  [0] Kein zusätzlicher Filter")
	}

	line, err := readLine("Auswahl: ")
	if err != nil {
		return 0, err
	}

	choice, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil {
		return 0, fmt.Errorf("ungültige Auswahl")
	}
	if allowSkip && choice == 0 {
		return 0, nil
	}
	if choice < 1 || choice > len(items) {
		return 0, fmt.Errorf("Auswahl außerhalb des gültigen Bereichs")
	}
	return items[choice-1].id, nil
}
