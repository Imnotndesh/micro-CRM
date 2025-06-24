package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"golang.org/x/crypto/bcrypt"
	"log"
	"micro-CRM/internal/logger"
	"micro-CRM/internal/models"
	"micro-CRM/internal/utils"
	"net/http"
)

type CRMHandlers struct {
	DB  *sql.DB
	Log logger.Logger
}

// RegisterUser handles user registration.
func (c *CRMHandlers) RegisterUser(w http.ResponseWriter, r *http.Request) {
	var payload models.UserRegistrationPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	c.Log.Info("Registering user call from : ", r.RemoteAddr)
	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Failed to process password")
		return
	}

	db := c.DB
	stmt, err := db.Prepare("INSERT INTO users (username, email, password_hash, first_name, last_name) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		log.Printf("Error preparing statement: %v", err)
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

	utils.RespondJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "User registered successfully",
		"token":   token,
		"user_id": userID,
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
	err := db.QueryRow("SELECT id, username, password_hash, email, first_name, last_name, created_at, updated_at FROM users WHERE username = ?", payload.Username).Scan(
		&user.ID, &user.Username, &user.PasswordHash, &user.Email, &user.FirstName, &user.LastName, &user.CreatedAt, &user.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		utils.RespondError(w, http.StatusUnauthorized, "Invalid username or password")
		return
	}
	if err != nil {
		log.Printf("Error querying user: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Database error")
		return
	}

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
		"user":    user, // Optionally return user details
	})
}
