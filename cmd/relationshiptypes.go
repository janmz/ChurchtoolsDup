package cmd

import (
	"fmt"
	"sort"

	config "github.com/janmz/churchtools-dup/internal/config"
	"github.com/spf13/cobra"
)

var relationshipTypesCmd = &cobra.Command{
	Use:   "relationship-types",
	Short: "Beziehungstypen der ChurchTools-Instanz auflisten",
	Run: func(cmd *cobra.Command, args []string) {
		exitOnError(runRelationshipTypes())
	},
}

func init() {
	rootCmd.AddCommand(relationshipTypesCmd)
}

func runRelationshipTypes() error {
	cfg, err := config.LoadOrEmpty(configPath)
	if err != nil {
		return err
	}

	client, err := connectChurchTools(cfg)
	if err != nil {
		return err
	}

	types, err := client.ListRelationshipTypes()
	if err != nil {
		return err
	}

	sort.Slice(types, func(i, j int) bool {
		if types[i].ID == types[j].ID {
			return types[i].Name < types[j].Name
		}
		return types[i].ID < types[j].ID
	})

	if len(types) == 0 {
		fmt.Println("Keine Beziehungstypen gefunden.")
		return nil
	}

	fmt.Printf("Beziehungstypen (%d):\n", len(types))
	for _, relType := range types {
		fmt.Printf("  %d  %s\n", relType.ID, relType.Name)
	}
	fmt.Printf("Instanz: %s\n", client.BaseURL())
	return nil
}
