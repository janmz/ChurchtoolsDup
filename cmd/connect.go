package cmd

import (
	"fmt"

	churchtools "github.com/janmz/churchtools-dup/internal/churchtools"
	config "github.com/janmz/churchtools-dup/internal/config"
)

func connectChurchTools(cfg config.Config) (*churchtools.Client, error) {
	client := churchtools.NewClient(cfg.BaseURL, cfg.LoginToken, cfg.Username, cfg.Password)
	if err := client.Login(); err != nil {
		return nil, err
	}
	if note := client.LoginRedirectNote(); note != "" {
		fmt.Println(note)
	}
	return client, nil
}
