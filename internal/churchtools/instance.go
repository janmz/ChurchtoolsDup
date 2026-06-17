package churchtools

import (
	"fmt"
	"net/url"
	"strings"
)

const churchToolsHostSuffix = ".church.tools"

var mainInstanceURLForLogin = MainInstanceURL

// MainInstanceURL derives the main ChurchTools instance from a sub-instance URL.
// Example: https://gemeinde-unterstadt.church.tools -> https://gemeinde.church.tools
func MainInstanceURL(instanceURL string) (string, bool) {
	parsed, err := url.Parse(normalizeInstanceURL(instanceURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", false
	}

	host := strings.ToLower(parsed.Hostname())
	if !strings.HasSuffix(host, churchToolsHostSuffix) {
		return "", false
	}

	subdomain := strings.TrimSuffix(host, churchToolsHostSuffix)
	dash := strings.Index(subdomain, "-")
	if dash <= 0 || dash >= len(subdomain)-1 {
		return "", false
	}

	mainHost := subdomain[:dash] + churchToolsHostSuffix
	if mainHost == host {
		return "", false
	}

	mainURL := *parsed
	mainURL.Host = mainHost
	if parsed.Port() != "" {
		mainURL.Host = mainHost + ":" + parsed.Port()
	}
	mainURL.Path = ""
	mainURL.RawPath = ""
	mainURL.RawQuery = ""
	mainURL.Fragment = ""

	return strings.TrimSuffix(mainURL.String(), "/"), true
}

// MainInstanceLoginNote explains that login succeeded on the main instance.
func MainInstanceLoginNote(configuredURL, mainURL string) string {
	return fmt.Sprintf(
		"Hinweis: Anmeldung auf der Hauptinstanz %s erfolgreich (konfiguriert: %s).",
		mainURL,
		configuredURL,
	)
}

// SubInstanceOAuthLoginNote explains OAuth bridging from central to sub-instance.
func SubInstanceOAuthLoginNote(subURL, centralURL string) string {
	return fmt.Sprintf(
		"Hinweis: Anmeldung über OAuth von Zentralinstanz %s für Nebeninstanz %s.",
		centralURL,
		subURL,
	)
}

func normalizeInstanceURL(raw string) string {
	url := strings.TrimSpace(raw)
	url = strings.TrimSuffix(url, "/")
	url = strings.TrimSuffix(url, "/api")
	return url
}
