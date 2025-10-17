package cmd

import (
	"fmt"

	"github.com/deichbewohner/swiftseer/internal/config"
	"github.com/deichbewohner/swiftseer/internal/ui"
)

func LogoutCmd(args []string) error {
    cfgManager, err := config.NewManager()
    if err != nil {
        return fmt.Errorf("failed to create config manager: %w", err)
    }

    if err := cfgManager.Clear(); err != nil {
        return fmt.Errorf("failed to clear config: %w", err)
    }

	fmt.Printf("%s Logged out successfully\n", ui.SuccessStyle.Render("✓"))
	fmt.Printf("%s %s\n", ui.InfoStyle.Render("Config file removed:"), cfgManager.GetConfigPath())

	return nil
}
