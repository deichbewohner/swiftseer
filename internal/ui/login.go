package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/deichbewohner/swiftseer/internal/client"
	"github.com/deichbewohner/swiftseer/internal/models"
)

type LoginModel struct {
	authClient *client.AuthClient
	username   string

    passwordInput textinput.Model
    otpInput      textinput.Model

    step   int // 0 = password, 1 = OTP (if needed)
    err    error
    done   bool
    result *models.TokenResponse
}

func NewLoginModel(authClient *client.AuthClient, username string) LoginModel {
    passwordInput := textinput.New()
    passwordInput.Placeholder = "Enter your password"
    passwordInput.EchoMode = textinput.EchoPassword
    passwordInput.EchoCharacter = '•'
    passwordInput.Width = 40
    passwordInput.Focus()
    otpInput := textinput.New()
    otpInput.Placeholder = "Enter OTP (or press Enter to skip)"
    otpInput.CharLimit = 6
    otpInput.Width = 40

	return LoginModel{
		authClient:    authClient,
		username:      username,
		passwordInput: passwordInput,
		otpInput:      otpInput,
		step:          0,
	}
}

func (m LoginModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m LoginModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.done = true
			m.err = fmt.Errorf("cancelled by user")
			return m, tea.Quit

		case tea.KeyEnter:
			switch m.step {
			case 0:
				password := m.passwordInput.Value()
				if password == "" {
					return m, nil
				}
				tokenResp, err := m.authClient.Authenticate(m.username, password, "")
				if err != nil {
					if strings.Contains(strings.ToLower(err.Error()), "otp") ||
						strings.Contains(strings.ToLower(err.Error()), "totp") ||
						strings.Contains(strings.ToLower(err.Error()), "2fa") {
						m.step = 1
						m.otpInput.Focus()
						return m, textinput.Blink
					}
					m.done = true
					m.err = err
					return m, tea.Quit
				}
				m.done = true
				m.result = tokenResp
				return m, tea.Quit

			case 1:
				password := m.passwordInput.Value()
				otp := m.otpInput.Value()
				tokenResp, err := m.authClient.Authenticate(m.username, password, otp)
				if err != nil {
					m.done = true
					m.err = err
					return m, tea.Quit
				}

				m.done = true
				m.result = tokenResp
				return m, tea.Quit
			}
		}

	case error:
		m.err = msg
		m.done = true
		return m, tea.Quit
	}

	// Update the focused input
	if m.step == 0 {
		m.passwordInput, cmd = m.passwordInput.Update(msg)
	} else {
		m.otpInput, cmd = m.otpInput.Update(msg)
	}

	return m, cmd
}

func (m LoginModel) View() string {
	if m.done {
		return ""
	}

	var b strings.Builder

	switch m.step {
	case 0:
		b.WriteString(fmt.Sprintf("Password for %s: ", HighlightStyle.Render(m.username)))
		b.WriteString(m.passwordInput.View())
		b.WriteString("\n")
	case 1:
		b.WriteString("OTP (optional): ")
		b.WriteString(m.otpInput.View())
		b.WriteString("\n")
		b.WriteString(InfoStyle.Render("Press Enter to skip if not using 2FA"))
		b.WriteString("\n")
	}

	return b.String()
}

func (m LoginModel) Result() (*models.TokenResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}
