package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	churchtools "github.com/janmz/churchtools-dup/internal/churchtools"
	config "github.com/janmz/churchtools-dup/internal/config"
	"github.com/janmz/churchtools-dup/internal/termio"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "ChurchTools-Verbindung und Berechtigungen einrichten",
}

var setupInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Interaktive Erstellung der config.json",
	Run: func(cmd *cobra.Command, args []string) {
		exitOnError(runSetupInit())
	},
}

var setupTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Verbindung und Login testen",
	Run: func(cmd *cobra.Command, args []string) {
		exitOnError(runSetupTest())
	},
}

var setupTokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Login-Token für die eigene Person abrufen",
	Run: func(cmd *cobra.Command, args []string) {
		exitOnError(runSetupToken())
	},
}

var setupPermissionsCmd = &cobra.Command{
	Use:   "permissions",
	Short: "Berechtigungen für Dubletten prüfen",
	Run: func(cmd *cobra.Command, args []string) {
		exitOnError(runSetupPermissions())
	},
}

var tokenPersonID int

func init() {
	setupCmd.AddCommand(setupInitCmd)
	setupCmd.AddCommand(setupTestCmd)
	setupCmd.AddCommand(setupTokenCmd)
	setupCmd.AddCommand(setupPermissionsCmd)

	setupTokenCmd.Flags().IntVar(&tokenPersonID, "person-id", 0, "Person-ID (Standard: angemeldeter Benutzer)")
}

func runSetupInit() error {
	reader := bufio.NewReader(os.Stdin)
	cfg := config.Config{DelayMS: 500}

	fmt.Print("ChurchTools-Instanz (z. B. meine-gemeinde): ")
	instanceName, _ := reader.ReadString('\n')
	baseURL, err := config.BaseURLFromInstanceName(instanceName)
	if err != nil {
		return err
	}
	cfg.BaseURL = baseURL

	fmt.Print("Login-Methode [token/password]: ")
	method, _ := reader.ReadString('\n')
	method = strings.ToLower(strings.TrimSpace(method))

	switch method {
	case "password", "passwort", "pw":
		fmt.Print("Benutzername: ")
		user, _ := reader.ReadString('\n')
		cfg.Username = strings.TrimSpace(user)

		pass, err := termio.ReadPassword("Passwort: ")
		if err != nil {
			return fmt.Errorf("Passwort einlesen: %w", err)
		}
		cfg.Password = strings.TrimSpace(pass)
	default:
		fmt.Print("Login-Token: ")
		token, _ := reader.ReadString('\n')
		cfg.LoginToken = strings.TrimSpace(token)
	}

	client, err := connectChurchTools(cfg)
	if err != nil {
		return fmt.Errorf("verbindungstest fehlgeschlagen: %w", err)
	}

	user, err := client.WhoAmI()
	if err != nil {
		return err
	}

	fmt.Printf("Login erfolgreich als %s %s (%s)\n", user.FirstName, user.LastName, user.Email)

	if cfg.LoginToken == "" {
		token, err := client.MeAPIToken()
		if err != nil || token == "" {
			token, err = client.LoginToken(user.ID)
		}
		if err == nil && token != "" {
			fmt.Println("\nOptional: Login-Token für dauerhafte API-Nutzung gefunden.")
			fmt.Print("Token in config.json speichern? [j/N]: ")
			answer, _ := reader.ReadString('\n')
			if strings.EqualFold(strings.TrimSpace(answer), "j") {
				cfg.LoginToken = token
				cfg.Username = ""
				cfg.Password = ""
			}
		}
	}

	fmt.Printf("Gruppen vor Export/Import (kommagetrennt) [%s]: ", config.DefaultPreJoinGroups)
	preJoinInput, _ := reader.ReadString('\n')
	preJoinInput = strings.TrimSpace(preJoinInput)
	if preJoinInput != "" {
		cfg.PreJoinGroups = preJoinInput
	} else {
		cfg.PreJoinGroups = config.DefaultPreJoinGroups
	}

	if err := config.Save(configPath, cfg); err != nil {
		return err
	}

	fmt.Printf("Konfiguration gespeichert: %s\n", configPath)
	return nil
}

func runSetupTest() error {
	cfg, err := config.LoadOrEmpty(configPath)
	if err != nil {
		return err
	}

	client, err := connectChurchTools(cfg)
	if err != nil {
		return fmt.Errorf("Login fehlgeschlagen: %w", err)
	}
	if err := client.Ping(); err != nil {
		return err
	}

	user, err := client.WhoAmI()
	if err != nil {
		return err
	}

	fmt.Println("Verbindung OK")
	fmt.Printf("Angemeldet als: %s %s (%s)\n", user.FirstName, user.LastName, user.Email)
	return nil
}

func runSetupToken() error {
	cfg, err := config.LoadOrEmpty(configPath)
	if err != nil {
		return err
	}

	client, err := connectChurchTools(cfg)
	if err != nil {
		return err
	}

	personID := tokenPersonID
	if personID == 0 {
		user, err := client.WhoAmI()
		if err != nil {
			return err
		}
		personID = user.ID
	}

	token, err := client.LoginToken(personID)
	if err != nil {
		return err
	}

	fmt.Printf("Login-Token für Person %d:\n%s\n", personID, token)
	fmt.Println("\nTrage den Token in config.json unter login_token ein oder setze CT_LOGIN_TOKEN.")
	return nil
}

func runSetupPermissions() error {
	cfg, err := config.LoadOrEmpty(configPath)
	if err != nil {
		return err
	}

	client, err := connectChurchTools(cfg)
	if err != nil {
		return err
	}

	perms, err := client.GlobalPermissions()
	if err != nil {
		return fmt.Errorf("Berechtigungen konnten nicht geladen werden: %w", err)
	}

	found := churchtools.FindRelationPermissions(perms)
	fmt.Println("Benötigte Berechtigungen (laut ChurchTools-Dokumentation):")
	for _, hint := range churchtools.PermissionHints {
		fmt.Printf("  - %s\n", hint)
	}

	fmt.Println("\nGefundene Beziehungs-/Personen-bezogene Einträge in /permissions/global:")
	if len(found) == 0 {
		fmt.Println("  (keine expliziten Treffer – prüfe Rechte manuell in ChurchTools)")
	} else {
		for _, item := range found {
			fmt.Printf("  - %s\n", item)
		}
	}

	fmt.Println("\nHinweis: Für `setup token` wird zusätzlich die Berechtigung zum Lesen")
	fmt.Println("des Login-Tokens benötigt (Profil → Berechtigungen / API-Token).")
	return nil
}
