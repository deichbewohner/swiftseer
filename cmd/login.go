package cmd

import (
	"flag"
	"fmt"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/deichbewohner/swiftseer/internal/client"
	"github.com/deichbewohner/swiftseer/internal/config"
	"github.com/deichbewohner/swiftseer/internal/models"
	"github.com/deichbewohner/swiftseer/internal/ui"
)

func LoginCmd(args []string) error {
	username, group, environment, err := parseLoginFlags(args)
	if err != nil {
		return err
	}

	cfgManager, err := config.NewManager()
	if err != nil {
		return fmt.Errorf("failed to create config manager: %w", err)
	}

	cfg, err := cfgManager.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	applyFlagOverrides(cfg, username, group, environment)
	if err := ensureLoginConfig(cfg); err != nil {
		return err
	}

	env, err := config.GetEnvironment(cfg.Environment)
	if err != nil {
		return err
	}

	authClient := client.NewAuthClient(env.GetAuthTokenURL(), nil)

	if hasEnvPassword() {
		return loginWithEnvCredentials(authClient, cfgManager, cfg)
	}

	return loginInteractively(authClient, cfgManager, cfg)
}

func parseLoginFlags(args []string) (username, group, environment string, err error) {
	fs := flag.NewFlagSet("login", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Authenticate with futureEXPERT and save credentials

Usage:
  swiftseer login [flags]

Flags:
  --user, -u     Username (optional if saved in config)
  --group, -g    Group (optional if saved in config)

Examples:
  # First-time login
  swiftseer login --user Analyst1 --group group-expert

  # Login with saved config (prompts for password)
  swiftseer login

  # Non-interactive login (CI/CD)
  FUTURE_PASSWORD=secret swiftseer login --user Analyst1 --group group-expert

Environment variables:
  FUTURE_USER        Override username
  FUTURE_PASSWORD    Password (non-interactive mode)
  FUTURE_OTP         Optional OTP code for 2FA
  FUTURE_GROUP       Override group

Config: ~/.config/swiftseer/config.yaml
`)
	}
	fs.StringVar(&username, "user", "", "Username (optional if saved in config)")
	fs.StringVar(&username, "u", "", "Username (short form)")
	fs.StringVar(&group, "group", "", "Group (optional if saved in config)")
	fs.StringVar(&group, "g", "", "Group (short form)")
	fs.StringVar(&environment, "env", "", "")
	fs.StringVar(&environment, "e", "", "")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		return "", "", "", fmt.Errorf("failed to parse flags: %w", err)
	}
	return username, group, environment, nil
}

func applyFlagOverrides(cfg *config.Config, username, group, environment string) {
	if username != "" {
		cfg.Username = username
	}
	if group != "" {
		cfg.Group = group
	}
	if environment != "" {
		cfg.Environment = environment
	}
}

func ensureLoginConfig(cfg *config.Config) error {
	if cfg.Environment == "" {
		cfg.Environment = "production"
	}
	if cfg.Username == "" {
		return fmt.Errorf("username is required (use --user flag or save in config)")
	}
	if cfg.Group == "" {
		return fmt.Errorf("group is required (use --group flag or save in config)")
	}
	return nil
}

func hasEnvPassword() bool {
	return os.Getenv("FUTURE_PASSWORD") != ""
}

func loginWithEnvCredentials(
	authClient *client.AuthClient,
	cfgManager *config.Manager,
	cfg *config.Config,
) error {
	pwd := os.Getenv("FUTURE_PASSWORD")
	otp := os.Getenv("FUTURE_OTP")

	tokenResp, err := authClient.Authenticate(cfg.Username, pwd, otp)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	if err := applyTokensAndSave(cfgManager, cfg, tokenResp); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	printAuthSummary(os.Stdout, cfgManager, cfg)
	return nil
}

func loginInteractively(
	authClient *client.AuthClient,
	cfgManager *config.Manager,
	cfg *config.Config,
) error {
	fmt.Printf("\n%s\n", ui.TitleStyle.Render("futureEXPERT Login"))
	fmt.Printf("  Username: %s\n", ui.HighlightStyle.Render(cfg.Username))
	fmt.Printf("  Group: %s\n", ui.HighlightStyle.Render(cfg.Group))
	fmt.Printf("  Environment: %s\n\n", ui.HighlightStyle.Render(cfg.Environment))

	loginModel := ui.NewLoginModel(authClient, cfg.Username)
	p := tea.NewProgram(loginModel)
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("login UI error: %w", err)
	}

	tokenResp, err := finalModel.(ui.LoginModel).Result()
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	if err := applyTokensAndSave(cfgManager, cfg, tokenResp); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	printAuthSummary(os.Stdout, cfgManager, cfg)
	return nil
}

func applyTokensAndSave(cfgManager *config.Manager, cfg *config.Config, t *models.TokenResponse) error {
	cfg.UpdateToken(t.AccessToken, t.RefreshToken, t.ExpiresIn, t.RefreshExpiresIn)
	return cfgManager.Save(cfg)
}

func printAuthSummary(w io.Writer, cfgManager *config.Manager, cfg *config.Config) {
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "%s Authentication successful!\n", ui.SuccessStyle.Render("✓"))
	_, _ = fmt.Fprintf(w, "  User: %s\n", cfg.Username)
	_, _ = fmt.Fprintf(w, "  Group: %s\n", cfg.Group)
	_, _ = fmt.Fprintf(w, "  Environment: %s\n", cfg.Environment)
	_, _ = fmt.Fprintf(w, "  Refresh token expires: %s\n", cfg.RefreshExpiresAt.Format("2006-01-02 15:04:05"))
	_, _ = fmt.Fprintf(w, "\n%s %s\n\n", ui.InfoStyle.Render("Configuration saved to:"), cfgManager.GetConfigPath())
}
