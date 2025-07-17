// internal/handlers/oidc.go
package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"micro-CRM/internal/models"
	"micro-CRM/internal/oidc"
	"micro-CRM/internal/utils"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

func (c *CRMHandlers) FindOrCreateUserByEmail(email, fullName string) (*models.User, error) {
	db := c.DB

	// Try to find existing user
	var user models.User
	query := "SELECT * FROM users WHERE email = ? LIMIT 1"
	err := db.QueryRow(query, email).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.Role, &user.PhoneNumber, &user.FirstName, &user.Status,
		&user.LastName, &user.CreatedAt, &user.UpdatedAt,
	)
	user.PasswordHash = ""

	if err == nil {
		return &user, nil // found existing user
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err // unexpected DB error
	}

	// If not found, create new user
	firstName, lastName := utils.SplitName(fullName)
	now := time.Now().Format(time.RFC3339)

	// Use email prefix as username fallback
	username := strings.Split(email, "@")[0]

	// Insert new user
	insertQuery := `
	INSERT INTO users (username, email, password_hash, role, phone_number, first_name, last_name, status, created_at, updated_at)
	VALUES (?, ?, ?, 'employee', 'none', ?, ?, 'active', ?, ?)
	`

	res, err := db.Exec(insertQuery, username, email, "oidc_login_placeholder", firstName, lastName, now, now)
	if err != nil {
		return nil, fmt.Errorf("insert user failed: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("retrieving last insert ID failed: %w", err)
	}

	user = models.User{
		ID:           int(id),
		Username:     username,
		Email:        email,
		PasswordHash: "oidc_login_placeholder",
		Role:         "employee",
		PhoneNumber:  "none",
		FirstName:    firstName,
		LastName:     lastName,
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	return &user, nil
}

func (c *CRMHandlers) OIDCLoginHandler(w http.ResponseWriter, r *http.Request) {
	state, err := utils.GenerateOIDCState()
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to generate random state: "+err.Error())
		return
	}
	http.Redirect(w, r, oidc.OauthConfig.AuthCodeURL(state), http.StatusFound)
}

func (c *CRMHandlers) OIDCCallbackHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	code := r.URL.Query().Get("code")

	token, err := oidc.OauthConfig.Exchange(ctx, code)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to exchange token: "+err.Error())
		return
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		utils.RespondError(w, http.StatusInternalServerError, "No id_token field in token")
		return
	}

	idToken, err := oidc.Verifier.Verify(ctx, rawIDToken)
	if err != nil {
		utils.RespondJSON(w, http.StatusInternalServerError, "Failed to verify ID Token: "+err.Error())
		return
	}

	var claims struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to parse claims: "+err.Error())
		return
	}

	// 🔐 Find or create user in your database
	user, err := c.FindOrCreateUserByEmail(claims.Email, claims.Name)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "User creation failed: "+err.Error())
		return
	}
	// 🔑 Generate your JWT
	jwtToken, err := utils.GenerateJWT(user.ID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "JWT generation failed: "+err.Error())
		return
	}

	// save token to datastore
	err = c.TokenStore.SaveIDToken(user.ID, rawIDToken, token.Expiry)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to save ID Token to token storage: "+err.Error())
	}
	userJson, err := json.Marshal(user)
	if err != nil {
		http.Error(w, "Failed to serialize user", http.StatusInternalServerError)
		return
	}

	// Redirect to React app with token and user info
	redirectURL := fmt.Sprintf("http://localhost:5173/oidc/callback?token=%s&user=%s", jwtToken, url.QueryEscape(string(userJson)))
	http.Redirect(w, r, redirectURL, http.StatusFound)
}
func (c *CRMHandlers) OIDCLogoutHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to fetch user ID from context")
		return
	}
	token, err := c.TokenStore.GetIDToken(userID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to fetch ID Token from token storage: "+err.Error())
		return
	}
	endSessionURL := os.Getenv("OIDC_LOGOUT_URI")
	postLogoutRedirect := os.Getenv("WEB_UI_BASE_URL") + "/login"

	// Some providers expect id_token_hint (optional)
	// If you have access to the ID token, include it:
	logoutURL := fmt.Sprintf("%s?id_token_hint=%s&post_logout_redirect_uri=%s", endSessionURL, token, url.QueryEscape(postLogoutRedirect))

	//logoutURL := fmt.Sprintf("%s?post_logout_redirect_uri=%s", endSessionURL, url.QueryEscape(postLogoutRedirect))
	http.Redirect(w, r, logoutURL, http.StatusFound)
}
