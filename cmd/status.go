package cmd

import (
	"fmt"
	"time"

	"github.com/deichbewohner/swiftseer/internal/config"
	"github.com/deichbewohner/swiftseer/internal/ui"
)

func StatusCmd(args []string) error {
    cfgManager, err := config.NewManager()
	if err != nil {
		return fmt.Errorf("failed to create config manager: %w", err)
	}

    cfg, err := cfgManager.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Printf("\n%s\n", ui.TitleStyle.Render("Configuration Status"))
	fmt.Printf("  Config file: %s\n\n", ui.InfoStyle.Render(cfgManager.GetConfigPath()))

    fmt.Println(ui.HighlightStyle.Render("User Configuration:"))
	if cfg.Username != "" {
		fmt.Printf("  Username: %s\n", cfg.Username)
	} else {
		fmt.Printf("  Username: %s\n", ui.InfoStyle.Render("(not set)"))
	}

	if cfg.Group != "" {
		fmt.Printf("  Group: %s\n", cfg.Group)
	} else {
		fmt.Printf("  Group: %s\n", ui.InfoStyle.Render("(not set)"))
	}

	if cfg.Environment != "" {
		fmt.Printf("  Environment: %s\n", cfg.Environment)
	} else {
		fmt.Printf("  Environment: %s\n", ui.InfoStyle.Render("(not set)"))
	}

	fmt.Println()

    fmt.Println(ui.HighlightStyle.Render("Authentication:"))
	if cfg.RefreshToken == "" {
		fmt.Printf("  Status: %s\n", ui.ErrorStyle.Render("Not logged in"))
		fmt.Printf("\n  %s\n\n", ui.InfoStyle.Render("Run 'swiftseer login' to authenticate"))
	} else if cfg.IsRefreshTokenValid() {
		timeLeft := time.Until(cfg.RefreshExpiresAt).Round(time.Second)
		fmt.Printf("  Status: %s\n", ui.SuccessStyle.Render("Logged in"))
		fmt.Printf("  Refresh token expires: %s (%s remaining)\n",
			cfg.RefreshExpiresAt.Format("2006-01-02 15:04:05"),
			ui.HighlightStyle.Render(timeLeft.String()))
		fmt.Println()
	} else {
		fmt.Printf("  Status: %s\n", ui.ErrorStyle.Render("Refresh token expired"))
		fmt.Printf("  Expired: %s\n", cfg.RefreshExpiresAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("\n  %s\n", ui.InfoStyle.Render("Run 'swiftseer login' to re-authenticate"))
		fmt.Println()
	}

	return nil
}
