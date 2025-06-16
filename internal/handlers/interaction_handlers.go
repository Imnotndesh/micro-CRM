package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"micro-CRM/internal/models"
	"micro-CRM/internal/utils"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// CreateInteraction handles the creation of a new interaction.
func (c *CRMHandlers) CreateInteraction(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(models.UserIDContextKey).(int)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var interaction models.Interaction
	if err := json.NewDecoder(r.Body).Decode(&interaction); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	interaction.UserID = userID // Assign the authenticated user's ID

	db := c.DB

	// Validate contact_id belongs to the user
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM contacts WHERE id = ? AND user_id = ?)", interaction.ContactID, userID).Scan(&exists)
	if err != nil || !exists {
		utils.RespondError(w, http.StatusForbidden, "Contact not found or does not belong to the user")
		return
	}

	stmt, err := db.Prepare(`INSERT INTO interactions (user_id, contact_id, type, description, interaction_at) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		log.Printf("Error preparing statement: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer stmt.Close()

	// If interaction_at is not provided, use current timestamp
	if interaction.InteractionAt == nil || *interaction.InteractionAt == "" {
		now := time.Now().Format(time.RFC3339)
		interaction.InteractionAt = &now
	}

	result, err := stmt.Exec(
		interaction.UserID,
		interaction.ContactID,
		interaction.Type,
		interaction.Description,
		interaction.InteractionAt,
	)
	if err != nil {
		log.Printf("Error inserting interaction: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Failed to create interaction")
		return
	}

	id, _ := result.LastInsertId()
	interaction.ID = int(id)
	interaction.CreatedAt = time.Now().Format(time.RFC3339)

	utils.RespondJSON(w, http.StatusCreated, interaction)
}

// GetInteraction retrieves a single interaction by ID.
func (c *CRMHandlers) GetInteraction(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(models.UserIDContextKey).(int)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	vars := mux.Vars(r)
	interactionID, err := strconv.Atoi(vars["id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid interaction ID")
		return
	}

	db := c.DB
	var interaction models.Interaction
	err = db.QueryRow(`SELECT id, user_id, contact_id, type, description, interaction_at, created_at FROM interactions WHERE id = ? AND user_id = ?`,
		interactionID, userID).Scan(
		&interaction.ID, &interaction.UserID, &interaction.ContactID, &interaction.Type,
		&interaction.Description, &interaction.InteractionAt, &interaction.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		utils.RespondError(w, http.StatusNotFound, "Interaction not found or unauthorized")
		return
	}
	if err != nil {
		log.Printf("Error querying interaction: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	utils.RespondJSON(w, http.StatusOK, interaction)
}

// ListInteractions retrieves all interactions for the authenticated user (or filtered by contact_id).
func (c *CRMHandlers) ListInteractions(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(models.UserIDContextKey).(int)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	db := c.DB
	query := `SELECT id, user_id, contact_id, type, description, interaction_at, created_at FROM interactions WHERE user_id = ?`
	args := []interface{}{userID}

	// Optional filtering by contact_id
	contactIDStr := r.URL.Query().Get("contact_id")
	if contactIDStr != "" {
		contactID, err := strconv.Atoi(contactIDStr)
		if err != nil {
			utils.RespondError(w, http.StatusBadRequest, "Invalid contact_id parameter")
			return
		}
		query += ` AND contact_id = ?`
		args = append(args, contactID)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Error querying interactions: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	var interactions []models.Interaction
	for rows.Next() {
		var interaction models.Interaction
		if err := rows.Scan(
			&interaction.ID, &interaction.UserID, &interaction.ContactID, &interaction.Type,
			&interaction.Description, &interaction.InteractionAt, &interaction.CreatedAt,
		); err != nil {
			log.Printf("Error scanning interaction row: %v", err)
			continue
		}
		interactions = append(interactions, interaction)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error iterating interaction rows: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	utils.RespondJSON(w, http.StatusOK, interactions)
}

// UpdateInteraction updates an existing interaction.
func (c *CRMHandlers) UpdateInteraction(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(models.UserIDContextKey).(int)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	vars := mux.Vars(r)
	interactionID, err := strconv.Atoi(vars["id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid interaction ID")
		return
	}

	var interaction models.Interaction
	if err := json.NewDecoder(r.Body).Decode(&interaction); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	interaction.ID = interactionID // Ensure the ID from the URL is used

	db := c.DB

	// Validate contact_id belongs to the user if provided in payload
	if interaction.ContactID != 0 { // 0 is default int value, indicates not set by JSON
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM contacts WHERE id = ? AND user_id = ?)", interaction.ContactID, userID).Scan(&exists)
		if err != nil || !exists {
			utils.RespondError(w, http.StatusForbidden, "Contact not found or does not belong to the user")
			return
		}
	}

	stmt, err := db.Prepare(`UPDATE interactions SET contact_id = ?, type = ?, description = ?, interaction_at = ? WHERE id = ? AND user_id = ?`)
	if err != nil {
		log.Printf("Error preparing statement: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer stmt.Close()

	result, err := stmt.Exec(
		interaction.ContactID,
		interaction.Type,
		interaction.Description,
		interaction.InteractionAt,
		interaction.ID,
		userID,
	)
	if err != nil {
		log.Printf("Error updating interaction: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Failed to update interaction")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		utils.RespondError(w, http.StatusNotFound, "Interaction not found or unauthorized to update")
		return
	}

	utils.RespondJSON(w, http.StatusOK, interaction)
}

// DeleteInteraction deletes an interaction.
func (c *CRMHandlers) DeleteInteraction(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(models.UserIDContextKey).(int)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	vars := mux.Vars(r)
	interactionID, err := strconv.Atoi(vars["id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid interaction ID")
		return
	}

	db := c.DB
	result, err := db.Exec("DELETE FROM interactions WHERE id = ? AND user_id = ?", interactionID, userID)
	if err != nil {
		log.Printf("Error deleting interaction: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Failed to delete interaction")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		utils.RespondError(w, http.StatusNotFound, "Interaction not found or unauthorized to delete")
		return
	}

	utils.RespondJSON(w, http.StatusNoContent, nil)
}
