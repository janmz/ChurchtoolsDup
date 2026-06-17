package cmd

import (
	"fmt"
	"sort"

	churchtools "github.com/janmz/churchtools-dup/internal/churchtools"
	config "github.com/janmz/churchtools-dup/internal/config"
	"github.com/spf13/cobra"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Angemeldeten ChurchTools-Benutzer anzeigen",
	Run: func(cmd *cobra.Command, args []string) {
		exitOnError(runWhoAmI())
	},
}

func runWhoAmI() error {
	cfg, err := config.LoadOrEmpty(configPath)
	if err != nil {
		return err
	}

	client, err := connectChurchTools(cfg)
	if err != nil {
		return err
	}

	user, err := client.WhoAmI()
	if err != nil {
		return err
	}

	fmt.Printf("Person-ID:   %d\n", user.ID)
	fmt.Printf("Name:        %s %s\n", user.FirstName, user.LastName)
	fmt.Printf("E-Mail:      %s\n", user.Email)

	if user.CampusID > 0 {
		fmt.Printf("Standort-ID: %d\n", user.CampusID)
		if name := campusDisplayName(client, user.CampusID); name != "" {
			fmt.Printf("Standort:    %s\n", name)
		}
	} else {
		fmt.Println("Standort-ID: nicht zugeordnet")
		if cfg.CampusID > 0 {
			if name := campusDisplayName(client, cfg.CampusID); name != "" {
				fmt.Printf("Standard-Standort (config): %s (ID %d)\n", name, cfg.CampusID)
			} else {
				fmt.Printf("Standard-Standort (config): ID %d\n", cfg.CampusID)
			}
		}
	}

	groups, err := client.ListPersonGroups(user.ID)
	if err != nil {
		return err
	}
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].ID == groups[j].ID {
			return churchtools.PlainGroupName(groups[i].Name) < churchtools.PlainGroupName(groups[j].Name)
		}
		return groups[i].ID < groups[j].ID
	})
	if len(groups) == 0 {
		fmt.Println("Gruppen:     (keine)")
	} else {
		fmt.Println("Gruppen:")
		for _, group := range groups {
			fmt.Printf("  %d  %s\n", group.ID, churchtools.PlainGroupName(group.Name))
		}
	}

	fmt.Printf("Instanz:     %s\n", client.BaseURL())
	return nil
}
