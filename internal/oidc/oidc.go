package oidc

import (
	"context"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
	"micro-CRM/internal/utils"
)

var (
	Provider    *oidc.Provider
	Verifier    *oidc.IDTokenVerifier
	OauthConfig *oauth2.Config
)

func InitOIDC(ctx context.Context) error {
	var (
		err    error
		params = utils.GetAllOidcParams()
	)

	Provider, err = oidc.NewProvider(ctx, params.IssuerUrl)
	if err != nil {
		return err
	}

	Verifier = Provider.Verifier(&oidc.Config{
		ClientID: params.ClientID,
	})

	OauthConfig = &oauth2.Config{
		ClientID:     params.ClientID,
		ClientSecret: params.ClientSecret,
		RedirectURL:  params.RedirectUri,
		Endpoint:     Provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}
	return nil
}
