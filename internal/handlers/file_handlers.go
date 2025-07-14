package handlers

import (
	"context"
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
	"strings"
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

	// Validate file type (simple example: only allow certain MIME types)
	allowedMIMETypes := map[string]bool{
		"image/jpeg":         true,
		"image/png":          true,
		"image/svg+xml":      true,
		"application/pdf":    true,
		"text/plain":         true,
		"text/xml":           true,
		"application/xml":    true,
		"application/msword": true, // .doc
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true, // .docx
		"application/vnd.ms-excel": true, // .xls
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true, // .xlsx
	}

	/// 4. Determine MIME type
	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil && err != io.EOF {
		c.Log.Warn("UploadFile: Unable to read file for content type detection: %v", err)
		utils.RespondError(w, http.StatusBadRequest, "Failed to read file content")
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		c.Log.Warn("UploadFile: Failed to rewind file after reading header: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Failed to process file")
		return
	}

	// Sniff MIME type from content
	sniffedType := http.DetectContentType(buffer)
	fileType := sniffedType
	if idx := strings.Index(fileType, ";"); idx != -1 {
		fileType = strings.TrimSpace(fileType[:idx])
	}

	// Fallback to file extension if needed
	ext := strings.ToLower(filepath.Ext(handler.Filename))
	if sniffedType == "application/octet-stream" || sniffedType == "text/plain" {
		switch ext {
		case ".svg":
			fileType = "image/svg+xml"
		case ".docx":
			fileType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
		case ".xlsx":
			fileType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		case ".xls":
			fileType = "application/vnd.ms-excel"
		case ".doc":
			fileType = "application/msword"
		case ".pdf":
			fileType = "application/pdf"
		}
	}

	if !allowedMIMETypes[fileType] {
		c.Log.Warn("UploadFile: Disallowed file type uploaded: %s", fileType)
		utils.RespondError(w, http.StatusBadRequest, "Unsupported file type.")
		return
	}

	// 5. Create a unique filename on the server to prevent conflicts and ensure safety
	rawFilename := handler.Filename
	cleanFilename := filepath.Base(rawFilename)   // removes any path traversal
	base := utils.SanitizeFilename(cleanFilename) // we'll create this function next

	ext = filepath.Ext(base)
	name := base[:len(base)-len(ext)]
	uniqueFilename := fmt.Sprintf("%s-%d%s", name, time.Now().UnixNano(), ext)
	// Add nanosecond timestamp for uniqueness
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

	// Extract optional metadata
	contactIDStr := r.FormValue("contact_id")
	var contactID *int
	if contactIDStr != "" {
		id, err := strconv.Atoi(contactIDStr)
		if err != nil {
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
			utils.RespondError(w, http.StatusBadRequest, "Invalid company_id format")
			return
		}
		companyID = &id
	}

	interactionIDStr := r.FormValue("interaction_id")
	var interactionID *int
	if interactionIDStr != "" {
		id, err := strconv.Atoi(interactionIDStr)
		if err != nil {
			utils.RespondError(w, http.StatusBadRequest, "Invalid interaction_id format")
			return
		}
		interactionID = &id
	}

	// Check if at least one was provided
	if contactID == nil && companyID == nil && interactionID == nil {
		utils.RespondError(w, http.StatusBadRequest, "At least one of contact_id, company_id, or interaction_id must be provided")
		return
	}

	// 9. Validate ownership to user
	if contactID != nil {
		if err := utils.ValidateOwnership(c.DB, "contacts", *contactID, userID); err != nil {
			utils.RespondError(w, http.StatusForbidden, err.Error())
			return
		}
	}

	if companyID != nil {
		if err := utils.ValidateOwnership(c.DB, "companies", *companyID, userID); err != nil {
			utils.RespondError(w, http.StatusForbidden, err.Error())
			return
		}
	}

	if interactionID != nil {
		if err := utils.ValidateOwnership(c.DB, "interactions", *interactionID, userID); err != nil {
			utils.RespondError(w, http.StatusForbidden, err.Error())
			return
		}
	}

	// 10. Create the database record
	fileRecord := models.File{
		UserID:        userID,
		ContactID:     contactID,
		CompanyID:     companyID,
		FileName:      cleanFilename,
		StoragePath:   storagePath,
		FileType:      &fileType,
		FileSize:      intPointer(int(fileSize)),
		InteractionID: interactionID,
	}

	stmt, err := c.DB.Prepare(`INSERT INTO files (user_id, contact_id, company_id, interaction_id, file_name, storage_path, file_type, file_size) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		c.Log.Error("UploadFile: Error preparing statement: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer stmt.Close()
	result, err := stmt.Exec(fileRecord.UserID, fileRecord.ContactID, fileRecord.CompanyID, fileRecord.InteractionID, fileRecord.FileName, fileRecord.StoragePath, fileRecord.FileType, fileRecord.FileSize)
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

//// CreateFile handles the creation of a new file record (metadata only).
//func (c *CRMHandlers) CreateFile(w http.ResponseWriter, r *http.Request) {
//	userID, ok := r.Context().Value(models.UserIDContextKey).(int)
//	if !ok {
//		utils.RespondError(w, http.StatusUnauthorized, "User not authenticated")
//		return
//	}
//
//	var file models.File
//	if err := json.NewDecoder(r.Body).Decode(&file); err != nil {
//		utils.RespondError(w, http.StatusBadRequest, "Invalid request payload")
//		return
//	}
//	file.UserID = userID // Assign the authenticated user's ID
//
//	db := c.DB
//
//	// Validate contact_id or company_id belongs to the user if provided
//	if file.ContactID != nil && *file.ContactID != 0 {
//		if err := utils.ValidateOwnership(c.DB, "contacts", *file.ContactID, userID); err != nil {
//			utils.RespondError(w, http.StatusForbidden, err.Error())
//			return
//		}
//	}
//	if file.CompanyID != nil && *file.CompanyID != 0 {
//		if err := utils.ValidateOwnership(c.DB, "contacts", *file.CompanyID, userID); err != nil {
//			utils.RespondError(w, http.StatusForbidden, err.Error())
//			return
//		}
//	}
//
//	stmt, err := db.Prepare(`INSERT INTO files (user_id, contact_id, company_id, file_name, storage_path, file_type, file_size) VALUES (?, ?, ?, ?, ?, ?, ?)`)
//	if err != nil {
//		log.Printf("Error preparing statement: %v", err)
//		utils.RespondError(w, http.StatusInternalServerError, "Database error")
//		return
//	}
//	defer stmt.Close()
//
//	result, err := stmt.Exec(
//		file.UserID,
//		file.ContactID,
//		file.CompanyID,
//		file.FileName,
//		file.StoragePath,
//		file.FileType,
//		file.FileSize,
//	)
//	if err != nil {
//		log.Printf("Error inserting file: %v", err)
//		utils.RespondError(w, http.StatusInternalServerError, "Failed to create file record")
//		return
//	}
//
//	id, _ := result.LastInsertId()
//	file.ID = int(id)
//	file.UploadedAt = time.Now().Format(time.RFC3339)
//
//	utils.RespondJSON(w, http.StatusCreated, file)
//}
//

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
	err = db.QueryRow(`SELECT id, user_id, contact_id, company_id, file_name, storage_path, file_type, file_size, uploaded_at, interaction_id FROM files WHERE id = ? AND user_id = ?`, fileID, userID).
		Scan(&file.ID, &file.UserID, &file.ContactID, &file.CompanyID, &file.FileName, &file.StoragePath, &file.FileType, &file.FileSize, &file.UploadedAt, &file.InteractionID)
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
	query := `SELECT id, user_id, contact_id, company_id, file_name, storage_path, file_type, file_size, uploaded_at,interaction_id FROM files WHERE user_id = ?`
	args := []interface{}{userID}

	// Filtering by contact_id
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
	// Filtering by interaction_id
	interactionIDStr := r.URL.Query().Get("interaction_id")
	if interactionIDStr != "" {
		interactionID, err := strconv.Atoi(interactionIDStr)
		if err != nil {
			utils.RespondError(w, http.StatusBadRequest, "Invalid interaction_id parameter")
			return
		}
		query += " AND interaction_id = ?"
		args = append(args, interactionID)
	}

	// Filtering by company_id
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
			&file.StoragePath, &file.FileType, &file.FileSize, &file.UploadedAt, &file.InteractionID,
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

	var payload struct {
		FileName      string `json:"file_name"`
		ContactID     *int   `json:"contact_id,omitempty"`
		CompanyID     *int   `json:"company_id,omitempty"`
		InteractionID *int   `json:"interaction_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Optional validation
	if payload.FileName == "" {
		utils.RespondError(w, http.StatusBadRequest, "File name is required")
		return
	}

	// Validate ownership of contact_id and company_id
	if payload.ContactID != nil && *payload.ContactID != 0 {
		var exists bool
		err := c.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM contacts WHERE id = ? AND user_id = ?)", *payload.ContactID, userID).Scan(&exists)
		if err != nil || !exists {
			utils.RespondError(w, http.StatusForbidden, "Associated contact not found or does not belong to the user")
			return
		}
	}
	if payload.CompanyID != nil && *payload.CompanyID != 0 {
		var exists bool
		err := c.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM companies WHERE id = ? AND user_id = ?)", *payload.CompanyID, userID).Scan(&exists)
		if err != nil || !exists {
			utils.RespondError(w, http.StatusForbidden, "Associated company not found or does not belong to the user")
			return
		}
	}
	if payload.InteractionID != nil && *payload.InteractionID != 0 {
		var exists bool
		err = c.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM interactions WHERE id = ? AND user_id = ?)", *payload.InteractionID, userID).Scan(&exists)
		if err != nil || !exists {
			utils.RespondError(w, http.StatusForbidden, "Associated interaction not found or does not belong to the user")
			return
		}
	}

	stmt, err := c.DB.Prepare(`
		UPDATE files
		SET contact_id = ?, company_id = ?, file_name = ?, interaction_id = ?
		WHERE id = ? AND user_id = ?
	`)
	if err != nil {
		c.Log.Error("UpdateFile: Prepare failed: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer stmt.Close()

	result, err := stmt.Exec(
		payload.ContactID,
		payload.CompanyID,
		payload.FileName,
		payload.InteractionID,
		fileID,
		userID,
	)
	if err != nil {
		c.Log.Error("UpdateFile: Exec failed: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Failed to update file record")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		utils.RespondError(w, http.StatusNotFound, "File not found or unauthorized to update")
		return
	}

	utils.RespondJSON(w, http.StatusOK, map[string]string{"status": "updated file"})
}
func (c *CRMHandlers) CleanupOrphanedFiles(w http.ResponseWriter, r *http.Request) {
	err := utils.CleanOrphanedFiles(c.DB, uploadDir)
	if err != nil {
		c.Log.Error("CleanupOrphanedFiles: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Failed to clean orphaned files")
		return
	}
	utils.RespondJSON(w, http.StatusOK, map[string]string{"status": "cleanup completed"})
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
	// Clear out orphaned files
	go func() {
		c.Log.Info("Running Files cleanup")
		err = utils.CleanOrphanedFiles(c.DB, uploadDir)
		if err != nil {
			return
		}
	}()
}

var downloadSemaphore = make(chan struct{}, 100) // Max 100 concurrent downloads

func (c *CRMHandlers) DownloadFileHandler(w http.ResponseWriter, r *http.Request) {
	// Add timeout context
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()

	// Rate limiting with context cancellation
	select {
	case downloadSemaphore <- struct{}{}:
		defer func() { <-downloadSemaphore }()
	case <-ctx.Done():
		http.Error(w, "Request cancelled", http.StatusRequestTimeout)
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	if id == "" {
		http.Error(w, "Invalid file ID", http.StatusBadRequest)
		return
	}

	// Context-aware database query
	var fileName, storagePath, fileType string
	var fileSize int64

	query := `SELECT file_name, storage_path, file_type, file_size FROM files WHERE id = ?`
	row := c.DB.QueryRowContext(ctx, query, id)

	err := row.Scan(&fileName, &storagePath, &fileType, &fileSize)
	if err != nil {
		if err == context.Canceled || err == context.DeadlineExceeded {
			return
		}
		if err == sql.ErrNoRows {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Check if client disconnected before file operations
	select {
	case <-ctx.Done():
		return
	default:
	}

	// Sanitize filename for header safety
	sanitizedName := strings.ReplaceAll(fileName, "\"", "")
	sanitizedName = strings.ReplaceAll(sanitizedName, "\n", "")
	sanitizedName = strings.ReplaceAll(sanitizedName, "\r", "")

	// Validate and set content type
	contentType := fileType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Basic content type validation for security
	safeTypes := map[string]bool{
		"application/pdf": true, "application/zip": true, "application/octet-stream": true,
		"image/jpeg": true, "image/png": true, "image/gif": true, "text/plain": true,
		"application/msword": true, "application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	}

	if !safeTypes[contentType] {
		contentType = "application/octet-stream"
	}

	// Set download headers
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", sanitizedName))
	w.Header().Set("Content-Type", contentType)
	if fileSize > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(fileSize, 10))
	}

	// Use http.ServeFile - highly optimized for concurrent access
	// Handles range requests, ETags, caching, and proper connection management
	http.ServeFile(w, r.WithContext(ctx), storagePath)
}

// Refactor everything below this line
func isViewableFileType(fileType string) bool {
	viewableTypes := map[string]bool{
		// PDF files
		"application/pdf": true,

		// Image files
		"image/jpeg":    true,
		"image/jpg":     true,
		"image/png":     true,
		"image/gif":     true,
		"image/webp":    true,
		"image/svg+xml": true,
		"image/bmp":     true,
		"image/tiff":    true,

		// Video files
		"video/mp4":       true,
		"video/mpeg":      true,
		"video/quicktime": true,
		"video/x-msvideo": true, // .avi
		"video/webm":      true,
		"video/ogg":       true,
		"video/3gpp":      true,
		"video/x-ms-wmv":  true,
		"video/x-flv":     true,
	}

	return viewableTypes[fileType]
}

var viewerSemaphore = make(chan struct{}, 100)

func (c *CRMHandlers) ViewFileHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()

	select {
	case viewerSemaphore <- struct{}{}:
		defer func() { <-viewerSemaphore }()
	case <-ctx.Done():
		http.Error(w, "Request cancelled", http.StatusRequestTimeout)
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		http.Error(w, "Invalid file ID", http.StatusBadRequest)
		return
	}

	var fileName, storagePath, fileType string
	var fileSize int64
	query := `SELECT file_name, storage_path, file_type, file_size FROM files WHERE id = ?`
	row := c.DB.QueryRowContext(ctx, query, id)

	err := row.Scan(&fileName, &storagePath, &fileType, &fileSize)
	if err != nil {
		if err == context.Canceled || err == context.DeadlineExceeded {
			return
		}
		if err == sql.ErrNoRows {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	select {
	case <-ctx.Done():
		return
	default:
	}

	// Determine content type with extension fallback
	contentType := fileType
	if !isViewableFileType(contentType) {
		ext := strings.ToLower(filepath.Ext(fileName))
		extensionToType := map[string]string{
			".pdf": "application/pdf", ".jpg": "image/jpeg", ".jpeg": "image/jpeg",
			".png": "image/png", ".gif": "image/gif", ".webp": "image/webp",
			".svg": "image/svg+xml", ".bmp": "image/bmp", ".tiff": "image/tiff",
			".tif": "image/tiff", ".mp4": "video/mp4", ".mpeg": "video/mpeg",
			".mpg": "video/mpeg", ".mov": "video/quicktime", ".avi": "video/x-msvideo",
			".webm": "video/webm", ".ogv": "video/ogg", ".3gp": "video/3gpp",
			".wmv": "video/x-ms-wmv", ".flv": "video/x-flv",
		}
		if ct, exists := extensionToType[ext]; exists {
			contentType = ct
		} else {
			http.Error(w, "File type not supported for viewing", http.StatusUnsupportedMediaType)
			return
		}
	}

	sanitizedName := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(fileName, "\"", ""), "\n", ""), "\r", "")

	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", sanitizedName))
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")

	if fileSize > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(fileSize, 10))
	}
	http.ServeFile(w, r.WithContext(ctx), storagePath)
}
