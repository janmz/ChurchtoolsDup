package cmd

import (
	"fmt"
	"os"
	"strings"

	churchtools "github.com/janmz/churchtools-dup/internal/churchtools"
	config "github.com/janmz/churchtools-dup/internal/config"
)

func connectChurchTools(cfg config.Config) (*churchtools.Client, error) {
	client := churchtools.NewClient(cfg.BaseURL, cfg.LoginTokenPassword, cfg.Username, cfg.Password)
	client.SetMainInstanceURL(cfg.MainURL)
	client.SetPublicSignupConfirm(newPublicSignupConfirm())
	if err := client.Login(); err != nil {
		return nil, err
	}
	if note := client.LoginRedirectNote(); note != "" {
		// stderr: stdout may carry CSV data (export -o -).
		fmt.Fprintln(os.Stderr, note)
	}
	return client, nil
}

// newPublicSignupConfirm asks once per group before a person is signed up
// through the public group signup form; the answer is reused for that group.
// Without an answer (e.g. piped input) nothing is signed up.
func newPublicSignupConfirm() churchtools.PublicSignupConfirm {
	answers := make(map[int]bool)
	return func(groupID, personID int) bool {
		if answer, ok := answers[groupID]; ok {
			return answer
		}
		line, err := readLine(fmt.Sprintf(
			"Direkter Beitritt zu Gruppe ID %d nicht erlaubt. Person %d über die öffentliche Gruppenanmeldung eintragen (gilt für diese Gruppe in diesem Lauf)? [j/N]: ",
			groupID, personID))
		answers[groupID] = err == nil && strings.EqualFold(strings.TrimSpace(line), "j")
		return answers[groupID]
	}
}

// leaveTemporaryGroups removes the user from groups joined during this run
// (pre-join and permission groups the user was not a member of before).
func leaveTemporaryGroups(client *churchtools.Client) {
	for _, result := range client.LeaveJoinedGroups() {
		if result.Err != nil {
			fmt.Fprintf(os.Stderr, "Warnung: Gruppe ID %d konnte nicht verlassen werden – bitte manuell austreten: %v\n", result.GroupID, result.Err)
			continue
		}
		fmt.Fprintf(os.Stderr, "Gruppe ID %d wieder verlassen (nur für diesen Lauf beigetreten).\n", result.GroupID)
	}
}
