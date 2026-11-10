package cmd

import (
	"fmt"
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
	Short: "Eigenen Login-Token abrufen und verschlüsselt in config.json speichern",
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

func init() {
	setupCmd.AddCommand(setupInitCmd)
	setupCmd.AddCommand(setupTestCmd)
	setupCmd.AddCommand(setupTokenCmd)
	setupCmd.AddCommand(setupPermissionsCmd)
}

// runSetupInit asks for connection settings. An existing config file is
// loaded first (after confirmation) and only the asked fields are replaced,
// so campus_id, permission_groups and duplicate_relationship_type survive.
func runSetupInit() error {
	cfg := config.Default()
	if config.Exists(configPath) {
		answer, err := readLine(fmt.Sprintf("%s existiert bereits. Verbindungsdaten neu eingeben (übrige Einstellungen bleiben erhalten)? [j/N]: ", configPath))
		if err != nil {
			return err
		}
		if !strings.EqualFold(strings.TrimSpace(answer), "j") {
			return fmt.Errorf("abgebrochen – %s unverändert", configPath)
		}
		existing, err := config.LoadFile(configPath)
		if err != nil {
			return err
		}
		cfg = existing
	}

	instanceName, err := readLine("ChurchTools-Instanz (z. B. meine-gemeinde): ")
	if err != nil {
		return err
	}
	baseURL, err := config.BaseURLFromInstanceName(instanceName)
	if err != nil {
		return err
	}
	cfg.BaseURL = baseURL

	mainURL, err := askMainInstanceURL(baseURL, cfg.MainURL)
	if err != nil {
		return err
	}
	cfg.MainURL = mainURL

	method, err := readLine("Login-Methode [token/password]: ")
	if err != nil {
		return err
	}

	switch strings.ToLower(strings.TrimSpace(method)) {
	case "password", "passwort", "pw":
		user, err := readLine("Benutzername: ")
		if err != nil {
			return err
		}
		pass, err := termio.ReadPassword("Passwort: ", stdinReader)
		if err != nil {
			return fmt.Errorf("Passwort einlesen: %w", err)
		}
		// The password is stored verbatim; spaces may be part of it.
		cfg.Username = strings.TrimSpace(user)
		cfg.Password = pass
		cfg.LoginTokenPassword = ""
	default:
		// Read hidden like a password: the token grants full account access.
		token, err := termio.ReadPassword("Login-Token: ", stdinReader)
		if err != nil {
			return fmt.Errorf("Login-Token einlesen: %w", err)
		}
		cfg.LoginTokenPassword = strings.TrimSpace(token)
		cfg.Username = ""
		cfg.Password = ""
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

	if cfg.LoginTokenPassword == "" {
		token, err := client.MeAPIToken()
		if err != nil || token == "" {
			token, err = client.LoginToken(user.ID)
		}
		if err == nil && token != "" {
			fmt.Println("\nOptional: Login-Token für dauerhafte API-Nutzung gefunden.")
			answer, _ := readLine("Token verschlüsselt in config.json speichern (statt Passwort)? [j/N]: ")
			if strings.EqualFold(strings.TrimSpace(answer), "j") {
				cfg.LoginTokenPassword = token
				cfg.Username = ""
				cfg.Password = ""
			}
		}
	}

	currentPreJoin := cfg.PreJoinGroups
	if strings.TrimSpace(currentPreJoin) == "" {
		currentPreJoin = config.DefaultPreJoinGroups
	}
	preJoinInput, _ := readLine(fmt.Sprintf("Gruppen vor Export/Import (kommagetrennt, - = keine) [%s]: ", currentPreJoin))
	preJoinInput = strings.TrimSpace(preJoinInput)
	if preJoinInput != "" {
		cfg.PreJoinGroups = preJoinInput
	} else {
		cfg.PreJoinGroups = currentPreJoin
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

// runSetupToken fetches the login token of the logged-in user and stores it
// encrypted in the config file. The token is never printed, and tokens of
// other persons cannot be requested (that would take over their account).
func runSetupToken() error {
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
	token, err := client.LoginToken(user.ID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(token) == "" {
		return fmt.Errorf("ChurchTools hat keinen Login-Token geliefert")
	}

	fmt.Printf("Login-Token für %s %s (Person %d) abgerufen.\n", user.FirstName, user.LastName, user.ID)
	answer, _ := readLine(fmt.Sprintf("Token verschlüsselt in %s speichern (Benutzername/Passwort werden dort entfernt)? [j/N]: ", configPath))
	if !strings.EqualFold(strings.TrimSpace(answer), "j") {
		fmt.Println("Nicht gespeichert.")
		return nil
	}

	// Load the file without environment values, so nothing from CT_* is
	// written to disk.
	fileCfg := config.Default()
	if config.Exists(configPath) {
		if fileCfg, err = config.LoadFile(configPath); err != nil {
			return err
		}
	} else {
		fileCfg.BaseURL = cfg.BaseURL
		fileCfg.MainURL = cfg.MainURL
	}
	fileCfg.LoginTokenPassword = token
	fileCfg.Username = ""
	fileCfg.Password = ""
	fileCfg.SecurePassword = ""
	if err := config.Save(configPath, fileCfg); err != nil {
		return err
	}
	fmt.Printf("Login-Token verschlüsselt in %s gespeichert.\n", configPath)
	return nil
}

// askMainInstanceURL asks for the main (central) instance of a sub-instance.
// A suggestion derived from the instance name is shown with its host, but
// only used after explicit confirmation: the derived host may belong to a
// different church, and credentials are sent there on login fallback.
func askMainInstanceURL(baseURL, current string) (string, error) {
	suggestion, ok := churchtools.MainInstanceURL(baseURL)
	if !ok && current == "" {
		return "", nil
	}

	fmt.Println("\nNebeninstanz: Schlägt die Anmeldung fehl, kann sich das Programm über die")
	fmt.Println("Hauptinstanz eurer Gemeinde anmelden. Zugangsdaten gehen dann an diese Adresse.")
	if ok {
		fmt.Printf("Vorschlag (nur übernehmen, wenn das eure Hauptinstanz ist): %s\n", suggestion)
	}
	prompt := "Hauptinstanz-URL ([j] = Vorschlag übernehmen, leer = keine"
	if current != "" {
		prompt += ", [=] = " + current + " behalten"
	}
	answer, err := readLine(prompt + "): ")
	if err != nil {
		return "", err
	}

	answer = strings.TrimSpace(answer)
	switch {
	case answer == "":
		return "", nil
	case answer == "=" && current != "":
		return current, nil
	case strings.EqualFold(answer, "j") && ok:
		return suggestion, nil
	default:
		return config.NormalizeBaseURL(answer), nil
	}
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
