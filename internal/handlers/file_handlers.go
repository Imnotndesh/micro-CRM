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

// CreateFile handles the creation of a new file record (metadata only).
// Actual file upload/storage is out of scope for this API.
func (c *CRMHandlers) CreateFile(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(models.UserIDContextKey).(int)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var file models.File
	if err := json.NewDecoder(r.Body).Decode(&file); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	file.UserID = userID // Assign the authenticated user's ID

	db := c.DB

	// Validate contact_id or company_id belongs to the user if provided
	if file.ContactID != nil && *file.ContactID != 0 {
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM contacts WHERE id = ? AND user_id = ?)", *file.ContactID, userID).Scan(&exists)
		if err != nil || !exists {
			utils.RespondError(w, http.StatusForbidden, "Associated contact not found or does not belong to the user")
			return
		}
	}
	if file.CompanyID != nil && *file.CompanyID != 0 {
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM companies WHERE id = ? AND user_id = ?)", *file.CompanyID, userID).Scan(&exists)
		if err != nil || !exists {
			utils.RespondError(w, http.StatusForbidden, "Associated company not found or does not belong to the user")
			return
		}
	}

	stmt, err := db.Prepare(`INSERT INTO files (user_id, contact_id, company_id, file_name, storage_path, file_type, file_size) VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		log.Printf("Error preparing statement: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer stmt.Close()

	result, err := stmt.Exec(
		file.UserID,
		file.ContactID,
		file.CompanyID,
		file.FileName,
		file.StoragePath,
		file.FileType,
		file.FileSize,
	)
	if err != nil {
		log.Printf("Error inserting file: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Failed to create file record")
		return
	}

	id, _ := result.LastInsertId()
	file.ID = int(id)
	file.UploadedAt = time.Now().Format(time.RFC3339)

	utils.RespondJSON(w, http.StatusCreated, file)
}

// GetFile retrieves a single file record by ID.
func (c *CRMHandlers) GetFile(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(models.UserIDContextKey).(int)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	vars := mux.Vars(r)
	fileID, err := strconv.Atoi(vars["id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid file ID")
		return
	}

	db := c.DB
	var file models.File
	err = db.QueryRow(`SELECT id, user_id, contact_id, company_id, file_name, storage_path, file_type, file_size, uploaded_at FROM files WHERE id = ? AND user_id = ?`,
		fileID, userID).Scan(
		&file.ID, &file.UserID, &file.ContactID, &file.CompanyID, &file.FileName,
		&file.StoragePath, &file.FileType, &file.FileSize, &file.UploadedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		utils.RespondError(w, http.StatusNotFound, "File not found or unauthorized")
		return
	}
	if err != nil {
		log.Printf("Error querying file: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	utils.RespondJSON(w, http.StatusOK, file)
}

// ListFiles retrieves all file records for the authenticated user (or filtered).
func (c *CRMHandlers) ListFiles(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(models.UserIDContextKey).(int)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	db := c.DB
	query := `SELECT id, user_id, contact_id, company_id, file_name, storage_path, file_type, file_size, uploaded_at FROM files WHERE user_id = ?`
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

	// Optional filtering by company_id
	companyIDStr := r.URL.Query().Get("company_id")
	if companyIDStr != "" {
		companyID, err := strconv.Atoi(companyIDStr)
		if err != nil {
			utils.RespondError(w, http.StatusBadRequest, "Invalid company_id parameter")
			return
		}
		query += ` AND company_id = ?`
		args = append(args, companyID)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Error querying files: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	var files []models.File
	for rows.Next() {
		var file models.File
		if err := rows.Scan(
			&file.ID, &file.UserID, &file.ContactID, &file.CompanyID, &file.FileName,
			&file.StoragePath, &file.FileType, &file.FileSize, &file.UploadedAt,
		); err != nil {
			log.Printf("Error scanning file row: %v", err)
			continue
		}
		files = append(files, file)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error iterating file rows: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	utils.RespondJSON(w, http.StatusOK, files)
}

// UpdateFile updates an existing file record.
func (c *CRMHandlers) UpdateFile(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(models.UserIDContextKey).(int)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	vars := mux.Vars(r)
	fileID, err := strconv.Atoi(vars["id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid file ID")
		return
	}

	var file models.File
	if err := json.NewDecoder(r.Body).Decode(&file); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	file.ID = fileID // Ensure the ID from the URL is used

	db := c.DB

	// Validate contact_id or company_id belongs to the user if provided in payload
	if file.ContactID != nil && *file.ContactID != 0 {
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM contacts WHERE id = ? AND user_id = ?)", *file.ContactID, userID).Scan(&exists)
		if err != nil || !exists {
			utils.RespondError(w, http.StatusForbidden, "Associated contact not found or does not belong to the user")
			return
		}
	}
	if file.CompanyID != nil && *file.CompanyID != 0 {
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM companies WHERE id = ? AND user_id = ?)", *file.CompanyID, userID).Scan(&exists)
		if err != nil || !exists {
			utils.RespondError(w, http.StatusForbidden, "Associated company not found or does not belong to the user")
			return
		}
	}

	stmt, err := db.Prepare(`UPDATE files SET contact_id = ?, company_id = ?, file_name = ?, storage_path = ?, file_type = ?, file_size = ? WHERE id = ? AND user_id = ?`)
	if err != nil {
		log.Printf("Error preparing statement: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer stmt.Close()

	result, err := stmt.Exec(
		file.ContactID,
		file.CompanyID,
		file.FileName,
		file.StoragePath,
		file.FileType,
		file.FileSize,
		file.ID,
		userID,
	)
	if err != nil {
		log.Printf("Error updating file: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Failed to update file record")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		utils.RespondError(w, http.StatusNotFound, "File not found or unauthorized to update")
		return
	}

	utils.RespondJSON(w, http.StatusOK, file)
}

// DeleteFile deletes a file record.
func (c *CRMHandlers) DeleteFile(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(models.UserIDContextKey).(int)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	vars := mux.Vars(r)
	fileID, err := strconv.Atoi(vars["id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid file ID")
		return
	}

	db := c.DB
	result, err := db.Exec("DELETE FROM files WHERE id = ? AND user_id = ?", fileID, userID)
	if err != nil {
		log.Printf("Error deleting file: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Failed to delete file record")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		utils.RespondError(w, http.StatusNotFound, "File not found or unauthorized to delete")
		return
	}

	utils.RespondJSON(w, http.StatusNoContent, nil)
}
