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

// CreateCompany handles the creation of a new company.
func (c *CRMHandlers) CreateCompany(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(models.UserIDContextKey).(int)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var company models.Company
	if err := json.NewDecoder(r.Body).Decode(&company); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	company.UserID = userID // Assign the authenticated user's ID

	db := c.DB
	stmt, err := db.Prepare("INSERT INTO companies (user_id, name, website, industry, address, phone_number, pipeline_stage) VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		log.Printf("Error preparing statement: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer stmt.Close()

	result, err := stmt.Exec(
		company.UserID,
		company.Name,
		company.Website,
		company.Industry,
		company.Address,
		company.PhoneNumber,
		company.PipelineStage,
	)
	if err != nil {
		log.Printf("Error inserting company: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Failed to create company")
		return
	}

	id, _ := result.LastInsertId()
	company.ID = int(id)
	company.CreatedAt = time.Now().Format(time.RFC3339)
	company.UpdatedAt = company.CreatedAt

	utils.RespondJSON(w, http.StatusCreated, company)
}

// GetCompany retrieves a single company by ID.
func (c *CRMHandlers) GetCompany(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(models.UserIDContextKey).(int)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	vars := mux.Vars(r)
	companyID, err := strconv.Atoi(vars["id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid company ID")
		return
	}

	db := c.DB
	var company models.Company
	err = db.QueryRow("SELECT id, user_id, name, website, industry, address, phone_number, created_at, updated_at, pipeline_stage FROM companies WHERE id = ? AND user_id = ?",
		companyID, userID).Scan(
		&company.ID, &company.UserID, &company.Name, &company.Website, &company.Industry,
		&company.Address, &company.PhoneNumber, &company.CreatedAt, &company.UpdatedAt, &company.PipelineStage,
	)
	if errors.Is(err, sql.ErrNoRows) {
		utils.RespondError(w, http.StatusNotFound, "Company not found or unauthorized")
		return
	}
	if err != nil {
		log.Printf("Error querying company: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	utils.RespondJSON(w, http.StatusOK, company)
}

// ListCompanies retrieves all companies for the authenticated user.
func (c *CRMHandlers) ListCompanies(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(models.UserIDContextKey).(int)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	db := c.DB
	rows, err := db.Query("SELECT id, user_id, name, website, industry, address, phone_number, created_at, updated_at, pipeline_stage FROM companies WHERE user_id = ?", userID)
	if err != nil {
		log.Printf("Error querying companies: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	var companies []models.Company
	for rows.Next() {
		var company models.Company
		if err := rows.Scan(
			&company.ID, &company.UserID, &company.Name, &company.Website, &company.Industry,
			&company.Address, &company.PhoneNumber, &company.CreatedAt, &company.UpdatedAt, &company.PipelineStage,
		); err != nil {
			log.Printf("Error scanning company row: %v", err)
			continue
		}
		companies = append(companies, company)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error iterating company rows: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	utils.RespondJSON(w, http.StatusOK, companies)
}

// UpdateCompany updates an existing company.
func (c *CRMHandlers) UpdateCompany(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(models.UserIDContextKey).(int)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	vars := mux.Vars(r)
	companyID, err := strconv.Atoi(vars["id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid company ID")
		return
	}

	var company models.Company
	if err := json.NewDecoder(r.Body).Decode(&company); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	company.ID = companyID // Ensure the ID from the URL is used

	db := c.DB
	stmt, err := db.Prepare(`UPDATE companies SET name = ?, website = ?, industry = ?, address = ?, phone_number = ?, pipeline_stage = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND user_id = ?`)
	if err != nil {
		log.Printf("Error preparing statement: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer stmt.Close()

	result, err := stmt.Exec(
		company.Name,
		company.Website,
		company.Industry,
		company.Address,
		company.PhoneNumber,
		company.PipelineStage,
		company.ID,
		userID,
	)
	if err != nil {
		log.Printf("Error updating company: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Failed to update company")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		utils.RespondError(w, http.StatusNotFound, "Company not found or unauthorized to update")
		return
	}

	// Retrieve updated company to return
	company.UpdatedAt = time.Now().Format(time.RFC3339) // Update timestamp
	utils.RespondJSON(w, http.StatusOK, company)
}

// DeleteCompany deletes a company.
func (c *CRMHandlers) DeleteCompany(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(models.UserIDContextKey).(int)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	vars := mux.Vars(r)
	companyID, err := strconv.Atoi(vars["id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid company ID")
		return
	}

	db := c.DB
	result, err := db.Exec("DELETE FROM companies WHERE id = ? AND user_id = ?", companyID, userID)
	if err != nil {
		log.Printf("Error deleting company: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Failed to delete company")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		utils.RespondError(w, http.StatusNotFound, "Company not found or unauthorized to delete")
		return
	}

	utils.RespondJSON(w, http.StatusNoContent, nil) // 204 No Content for successful deletion
}
