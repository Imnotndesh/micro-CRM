package utils

import (
	"micro-CRM/internal/models"
	"os"
)

func GetAllOidcParams() models.OidcConfig {
	return models.OidcConfig{
		IssuerUrl:    os.Getenv("OIDC_ISSUER"),
		ClientID:     os.Getenv("OIDC_CLIENT_ID"),
		ClientSecret: os.Getenv("OIDC_CLIENT_SECRET"),
		RedirectUri:  os.Getenv("OIDC_REDIRECT_URI"),
		LogoutUrl:    os.Getenv("OIDC_LOGOUT_URL"),
	}
}
func IsOidcMissing(p models.OidcConfig) bool {
	return p.IssuerUrl == "" || p.ClientID == "" || p.ClientSecret == "" || p.RedirectUri == "" || p.LogoutUrl == ""
}
