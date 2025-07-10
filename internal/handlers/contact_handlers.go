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

// CreateContact handles the creation of a new contact.
func (c *CRMHandlers) CreateContact(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(models.UserIDContextKey).(int)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var contact models.Contact
	if err := json.NewDecoder(r.Body).Decode(&contact); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	contact.UserID = userID // Assign the authenticated user's ID

	db := c.DB
	stmt, err := db.Prepare(`INSERT INTO contacts (user_id, company_id, first_name, last_name, email, phone_number, job_title, notes, last_interaction_at, next_action_at, next_action_description, pipeline_stage) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		log.Printf("Error preparing statement: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer stmt.Close()

	result, err := stmt.Exec(
		contact.UserID,
		contact.CompanyID,
		contact.FirstName,
		contact.LastName,
		contact.Email,
		contact.PhoneNumber,
		contact.JobTitle,
		contact.Notes,
		contact.LastInteractionAt,
		contact.NextActionAt,
		contact.NextActionDescription,
		contact.PipelineStage,
	)
	if err != nil {
		log.Printf("Error inserting contact: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Failed to create contact")
		return
	}

	id, _ := result.LastInsertId()
	contact.ID = int(id)
	contact.CreatedAt = time.Now().Format(time.RFC3339)
	contact.UpdatedAt = contact.CreatedAt

	utils.RespondJSON(w, http.StatusCreated, contact)
}

// GetContact retrieves a single contact by ID.
func (c *CRMHandlers) GetContact(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(models.UserIDContextKey).(int)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	vars := mux.Vars(r)
	contactID, err := strconv.Atoi(vars["id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid contact ID")
		return
	}
	var contact models.Contact
	query := `
		SELECT 
			id, user_id, company_id, first_name, last_name, email,
			phone_number, job_title, notes, created_at, updated_at,
			last_interaction_at, next_action_at, next_action_description, pipeline_stage
		FROM contacts 
		WHERE id = ? AND user_id = ?
	`

	err = c.DB.QueryRow(query, contactID, userID).Scan(
		&contact.ID, &contact.UserID, &contact.CompanyID, &contact.FirstName, &contact.LastName, &contact.Email,
		&contact.PhoneNumber, &contact.JobTitle, &contact.Notes, &contact.CreatedAt, &contact.UpdatedAt,
		&contact.LastInteractionAt, &contact.NextActionAt, &contact.NextActionDescription, &contact.PipelineStage,
	)

	if errors.Is(err, sql.ErrNoRows) {
		utils.RespondError(w, http.StatusNotFound, "Contact not found or unauthorized")
		return
	}
	if err != nil {
		log.Printf("Error querying contact: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	utils.RespondJSON(w, http.StatusOK, contact)
}

// ListContacts retrieves all contacts for the authenticated user.
func (c *CRMHandlers) ListContacts(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(models.UserIDContextKey).(int)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	query := `
		SELECT 
			id, user_id, company_id, first_name, last_name, email,
			phone_number, job_title, notes, created_at, updated_at,
			last_interaction_at, next_action_at, next_action_description, pipeline_stage
		FROM contacts
		WHERE user_id = ?
	`

	rows, err := c.DB.Query(query, userID)
	if err != nil {
		log.Printf("Error querying contacts: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	var contacts []models.Contact

	for rows.Next() {
		var contact models.Contact
		err := rows.Scan(
			&contact.ID,
			&contact.UserID,
			&contact.CompanyID,
			&contact.FirstName,
			&contact.LastName,
			&contact.Email,
			&contact.PhoneNumber,
			&contact.JobTitle,
			&contact.Notes,
			&contact.CreatedAt,
			&contact.UpdatedAt,
			&contact.LastInteractionAt,
			&contact.NextActionAt,
			&contact.NextActionDescription,
			&contact.PipelineStage,
		)
		if err != nil {
			log.Printf("Error scanning contact row: %v", err)
			continue // skip corrupted row
		}
		contacts = append(contacts, contact)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Row iteration error: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Database iteration error")
		return
	}

	utils.RespondJSON(w, http.StatusOK, contacts)
}

// UpdateContact updates an existing contact.
func (c *CRMHandlers) UpdateContact(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(models.UserIDContextKey).(int)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	vars := mux.Vars(r)
	contactID, err := strconv.Atoi(vars["id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid contact ID")
		return
	}

	var contact models.Contact
	if err := json.NewDecoder(r.Body).Decode(&contact); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	contact.ID = contactID // Ensure the ID from the URL is used

	db := c.DB
	stmt, err := db.Prepare(`UPDATE contacts SET company_id = ?, first_name = ?, last_name = ?, email = ?, phone_number = ?, job_title = ?, notes = ?, last_interaction_at = ?, next_action_at = ?, next_action_description = ?, pipeline_stage = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND user_id = ?`)
	if err != nil {
		log.Printf("Error preparing statement: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer stmt.Close()

	result, err := stmt.Exec(
		contact.CompanyID,
		contact.FirstName,
		contact.LastName,
		contact.Email,
		contact.PhoneNumber,
		contact.JobTitle,
		contact.Notes,
		contact.LastInteractionAt,
		contact.NextActionAt,
		contact.NextActionDescription,
		contact.PipelineStage,
		contact.ID,
		userID,
	)
	if err != nil {
		log.Printf("Error updating contact: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Failed to update contact")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		utils.RespondError(w, http.StatusNotFound, "Contact not found or unauthorized to update")
		return
	}

	contact.UpdatedAt = time.Now().Format(time.RFC3339)
	utils.RespondJSON(w, http.StatusOK, contact)
}

// DeleteContact deletes a contact.
func (c *CRMHandlers) DeleteContact(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(models.UserIDContextKey).(int)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	vars := mux.Vars(r)
	contactID, err := strconv.Atoi(vars["id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid contact ID")
		return
	}

	db := c.DB
	result, err := db.Exec("DELETE FROM contacts WHERE id = ? AND user_id = ?", contactID, userID)
	if err != nil {
		log.Printf("Error deleting contact: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Failed to delete contact")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		utils.RespondError(w, http.StatusNotFound, "Contact not found or unauthorized to delete")
		return
	}

	utils.RespondJSON(w, http.StatusNoContent, nil)
}
