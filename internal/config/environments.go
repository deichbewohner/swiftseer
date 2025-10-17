package config

import "fmt"

type Environment struct {
	APIURL        string
	AuthRealm     string
	AuthServerURL string
}

var Environments = map[string]Environment{
	"production": {
		APIURL:        "https://api.future-forecasting.de/api/v1/",
		AuthRealm:     "future",
		AuthServerURL: "https://future-auth.prognostica.de",
	},
	"staging": {
		APIURL:        "https://api.staging.future-forecasting.de/api/v1/",
		AuthRealm:     "development",
		AuthServerURL: "https://future-auth.prognostica.de",
	},
	"development": {
		APIURL:        "https://api.dev.future-forecasting.de/api/v1/",
		AuthRealm:     "development",
		AuthServerURL: "https://future-auth.prognostica.de",
	},
}

func GetEnvironment(name string) (Environment, error) {
	env, ok := Environments[name]
	if !ok {
		return Environment{}, fmt.Errorf(
			"unknown environment: %s (valid: production, staging, development)",
			name,
		)
	}
	return env, nil
}

func (e Environment) GetAuthTokenURL() string {
	return fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", e.AuthServerURL, e.AuthRealm)
}
