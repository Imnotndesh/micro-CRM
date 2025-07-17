package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"golang.org/x/crypto/bcrypt"
	"log"
	"micro-CRM/internal/logger"
	"micro-CRM/internal/models"
	"micro-CRM/internal/tokenstore"
	"micro-CRM/internal/utils"
	"net/http"
)

type CRMHandlers struct {
	DB         *sql.DB
	Log        logger.Logger
	TokenStore *tokenstore.BuntDBTokenStore
}

// RegisterUser handles user registration.
func (c *CRMHandlers) RegisterUser(w http.ResponseWriter, r *http.Request) {
	var payload models.UserRegistrationPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	// Hash the password
	hashedPassword, err := utils.GeneratePassword(payload.Password)
	if err != nil {
		c.Log.Warn("Error hashing password: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Failed to process password")
		return
	}

	db := c.DB
	stmt, err := db.Prepare("INSERT INTO users (username, email, password_hash, first_name, last_name) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		c.Log.Warn("Error preparing statement: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer stmt.Close()

	result, err := stmt.Exec(payload.Username, payload.Email, string(hashedPassword), payload.FirstName, payload.LastName)
	if err != nil {
		log.Printf("Error inserting user: %v", err)
		// Check for unique constraint violation
		if err.Error() == "UNIQUE constraint failed: users.username" || err.Error() == "UNIQUE constraint failed: users.email" {
			utils.RespondError(w, http.StatusConflict, "Username or Email already exists")
		} else {
			utils.RespondError(w, http.StatusInternalServerError, "Failed to register user")
		}
		return
	}

	userID, _ := result.LastInsertId()
	token, err := utils.GenerateJWT(int(userID))
	if err != nil {
		log.Printf("Error generating JWT: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Failed to generate authentication token")
		return
	}
	var responseUser models.User
	err = db.QueryRow("SELECT id,first_name,last_name,username,email,role,phone_number,created_at,status FROM users WHERE id = ?", userID).Scan(&responseUser.ID, &responseUser.FirstName, &responseUser.LastName, &responseUser.Username, &responseUser.Email, &responseUser.Role, &responseUser.PhoneNumber, &responseUser.CreatedAt, &responseUser.Status)
	if err != nil {
		c.Log.Warn("Error querying user: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if errors.Is(err, sql.ErrNoRows) {
		c.Log.Info("User not found")
		utils.RespondError(w, http.StatusNotFound, "User not found")
		return
	}
	utils.RespondJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "User registered successfully",
		"token":   token,
		"user":    responseUser,
	})
}

// LoginUser handles user login and JWT generation.
func (c *CRMHandlers) LoginUser(w http.ResponseWriter, r *http.Request) {
	var payload models.UserLoginPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	db := c.DB
	c.Log.Info("User login request")
	var user models.User
	err := db.QueryRow("SELECT id,first_name,last_name,username,email,role,phone_number,created_at,password_hash FROM users WHERE username = ?", payload.Username).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Username, &user.Email, &user.Role, &user.PhoneNumber, &user.CreatedAt, &user.PasswordHash)
	if errors.Is(err, sql.ErrNoRows) {
		utils.RespondError(w, http.StatusUnauthorized, "Invalid username or password")
		return
	}
	if err != nil {
		log.Printf("Error querying user: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if user.Status == "inactive" {
		c.Log.Info("User is inactive")
		utils.RespondJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"message": "User is inactive, Contact administrator to configure your user",
		})
		return
	}
	c.Log.Info("User is active")

	// Compare password hash
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(payload.Password))
	if err != nil {
		utils.RespondError(w, http.StatusUnauthorized, "Invalid username or password")
		return
	}

	token, err := utils.GenerateJWT(user.ID)
	if err != nil {
		log.Printf("Error generating JWT: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Failed to generate authentication token")
		return
	}

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Login successful",
		"token":   token,
		"user":    user,
	})
}
