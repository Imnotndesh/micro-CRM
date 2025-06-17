package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"micro-CRM/internal/logger"
	"micro-CRM/internal/models"
	"micro-CRM/internal/utils"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

const (
	maxUploadSize = 10 << 20
	uploadDir     = "./uploads"
)

func ensureUploadsDir(c logger.Logger) {
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		c.Info("Creating upload directory")
		if err = os.Mkdir(uploadDir, 0755); err != nil {
			c.Fatal("Unable to create upload directory")
		}
	}
}
func (c *CRMHandlers) UploadFileHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(models.UserIDContextKey).(int)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	// 1. Uploads Dir Check
	c.Log.Debug("UploadFile: received request to upload file")
	ensureUploadsDir(c.Log)

	// 2. Limit the size of the uploaded file
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		c.Log.Warn("UploadFile: Max upload size exceeded or invalid multipart form: ", err)
		utils.RespondError(w, http.StatusBadRequest, fmt.Sprintf("File too large. Max size is %dMB", maxUploadSize/(1<<20)))
		return
	}

	// 3. Get the file from the form data
	file, handler, err := r.FormFile("file") // "file" is the name of the input field in the form
	if err != nil {
		c.Log.Warn("UploadFile: Error retrieving file from form: %v", err)
		utils.RespondError(w, http.StatusBadRequest, "Error retrieving file from form. Make sure the input field is named 'file'.")
		return
	}
	defer file.Close() // Ensure the uploaded file is closed

	// 4. Validate file type (simple example: only allow certain MIME types)
	// You should implement more robust content-type sniffing if security is critical
	allowedMIMETypes := map[string]bool{
		"image/jpeg":         true,
		"image/png":          true,
		"application/pdf":    true,
		"text/plain":         true,
		"application/msword": true, // .doc
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true, // .docx
		"application/vnd.ms-excel": true, // .xls
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true, // .xlsx
	}

	fileType := handler.Header.Get("Content-Type")
	if !allowedMIMETypes[fileType] {
		c.Log.Warn("UploadFile: Disallowed file type uploaded: %s", fileType)
		utils.RespondError(w, http.StatusBadRequest, "Unsupported file type. Allowed: JPEG, PNG, PDF, TXT, Word, Excel.")
		return
	}

	// 5. Create a unique filename on the server to prevent conflicts and ensure safety
	filename := handler.Filename
	ext := filepath.Ext(filename)
	base := filename[:len(filename)-len(ext)]
	uniqueFilename := fmt.Sprintf("%s-%d%s", base, time.Now().UnixNano(), ext) // Add nanosecond timestamp for uniqueness
	storagePath := filepath.Join(uploadDir, uniqueFilename)

	// 6. Create the destination file on the server
	dst, err := os.Create(storagePath)
	if err != nil {
		c.Log.Error("UploadFile: Error creating destination file %s: %v", storagePath, err)
		utils.RespondError(w, http.StatusInternalServerError, "Could not save file on server")
		return
	}
	defer dst.Close() // Ensure the destination file is closed

	// 7. Copy the uploaded file to the destination
	fileSize, err := io.Copy(dst, file) // io.Copy returns the number of bytes copied
	if err != nil {
		c.Log.Error("UploadFile: Error copying file to destination %s: %v", storagePath, err)
		utils.RespondError(w, http.StatusInternalServerError, "Error saving file")
		return
	}

	c.Log.Info("UploadFile: Successfully saved file: %s (Size: %d bytes)", uniqueFilename, fileSize)

	// 8. Extract optional metadata from form fields
	contactIDStr := r.FormValue("contact_id")
	var contactID *int
	if contactIDStr != "" {
		id, err := strconv.Atoi(contactIDStr)
		if err != nil {
			c.Log.Warn("UploadFile: Invalid contact_id in form: ", err)
			utils.RespondError(w, http.StatusBadRequest, "Invalid contact_id format")
			return
		}
		contactID = &id
	}

	companyIDStr := r.FormValue("company_id")
	var companyID *int
	if companyIDStr != "" {
		id, err := strconv.Atoi(companyIDStr)
		if err != nil {
			c.Log.Warn("UploadFile: Invalid company_id in form: %v", err)
			utils.RespondError(w, http.StatusBadRequest, "Invalid company_id format")
			return
		}
		companyID = &id
	}

	// 9. Validate contact_id or company_id belongs to the user if provided
	if contactID != nil && *contactID != 0 {
		var exists bool
		err := c.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM contacts WHERE id = ? AND user_id = ?)", *contactID, userID).Scan(&exists)
		if err != nil || !exists {
			c.Log.Warn("UploadFile: Contact ID %d not found or does not belong to user %d", *contactID, userID)
			utils.RespondError(w, http.StatusForbidden, "Associated contact not found or does not belong to the user")
			return
		}
	}
	if companyID != nil && *companyID != 0 {
		var exists bool
		err := c.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM companies WHERE id = ? AND user_id = ?)", *companyID, userID).Scan(&exists)
		if err != nil || !exists {
			c.Log.Warn("UploadFile: Company ID %d not found or does not belong to user %d", *companyID, userID)
			utils.RespondError(w, http.StatusForbidden, "Associated company not found or does not belong to the user")
			return
		}
	}

	// 10. Create the database record
	fileRecord := models.File{
		UserID:      userID,
		ContactID:   contactID,
		CompanyID:   companyID,
		FileName:    filename,                  // Original filename
		StoragePath: storagePath,               // Unique path on server
		FileType:    &fileType,                 // MIME type from header
		FileSize:    intPointer(int(fileSize)), // Convert int64 to int, use helper
	}

	stmt, err := c.DB.Prepare(`INSERT INTO files (user_id, contact_id, company_id, file_name, storage_path, file_type, file_size) VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		c.Log.Error("UploadFile: Error preparing statement: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer stmt.Close()

	result, err := stmt.Exec(
		fileRecord.UserID,
		fileRecord.ContactID,
		fileRecord.CompanyID,
		fileRecord.FileName,
		fileRecord.StoragePath,
		fileRecord.FileType,
		fileRecord.FileSize,
	)
	if err != nil {
		c.Log.Error("UploadFile: Error inserting file record: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Failed to create file record")
		return
	}

	id, _ := result.LastInsertId()
	fileRecord.ID = int(id)
	now := time.Now().Format(time.RFC3339)
	fileRecord.UploadedAt = now

	c.Log.Info("UploadFile: File record created successfully for file %s", fileRecord.FileName)
	utils.RespondJSON(w, http.StatusCreated, fileRecord)
}
func intPointer(i int) *int {
	return &i
}

// CreateFile handles the creation of a new file record (metadata only).
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
