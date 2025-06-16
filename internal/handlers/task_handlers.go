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

// CreateTask handles the creation of a new task.
func (c *CRMHandlers) CreateTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(models.UserIDContextKey).(int)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var task models.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	task.UserID = userID // Assign the authenticated user's ID

	db := c.DB

	// Validate contact_id belongs to the user if provided
	if task.ContactID != nil && *task.ContactID != 0 {
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM contacts WHERE id = ? AND user_id = ?)", *task.ContactID, userID).Scan(&exists)
		if err != nil || !exists {
			utils.RespondError(w, http.StatusForbidden, "Contact not found or does not belong to the user")
			return
		}
	}

	stmt, err := db.Prepare(`INSERT INTO tasks (user_id, contact_id, title, description, due_date, status, priority) VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		log.Printf("Error preparing statement: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer stmt.Close()

	result, err := stmt.Exec(
		task.UserID,
		task.ContactID,
		task.Title,
		task.Description,
		task.DueDate,
		task.Status,
		task.Priority,
	)
	if err != nil {
		log.Printf("Error inserting task: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Failed to create task")
		return
	}

	id, _ := result.LastInsertId()
	task.ID = int(id)
	task.CreatedAt = time.Now().Format(time.RFC3339)
	task.UpdatedAt = task.CreatedAt

	utils.RespondJSON(w, http.StatusCreated, task)
}

// GetTask retrieves a single task by ID.
func (c *CRMHandlers) GetTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(models.UserIDContextKey).(int)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	vars := mux.Vars(r)
	taskID, err := strconv.Atoi(vars["id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid task ID")
		return
	}

	db := c.DB
	var task models.Task
	err = db.QueryRow(`SELECT id, user_id, contact_id, title, description, due_date, status, priority, created_at, updated_at FROM tasks WHERE id = ? AND user_id = ?`,
		taskID, userID).Scan(
		&task.ID, &task.UserID, &task.ContactID, &task.Title, &task.Description,
		&task.DueDate, &task.Status, &task.Priority, &task.CreatedAt, &task.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		utils.RespondError(w, http.StatusNotFound, "Task not found or unauthorized")
		return
	}
	if err != nil {
		log.Printf("Error querying task: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	utils.RespondJSON(w, http.StatusOK, task)
}

// ListTasks retrieves all tasks for the authenticated user (or filtered by contact_id/status).
func (c *CRMHandlers) ListTasks(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(models.UserIDContextKey).(int)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	db := c.DB
	query := `SELECT id, user_id, contact_id, title, description, due_date, status, priority, created_at, updated_at FROM tasks WHERE user_id = ?`
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

	// Optional filtering by status
	status := r.URL.Query().Get("status")
	if status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Error querying tasks: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		var task models.Task
		if err := rows.Scan(
			&task.ID, &task.UserID, &task.ContactID, &task.Title, &task.Description,
			&task.DueDate, &task.Status, &task.Priority, &task.CreatedAt, &task.UpdatedAt,
		); err != nil {
			log.Printf("Error scanning task row: %v", err)
			continue
		}
		tasks = append(tasks, task)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error iterating task rows: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	utils.RespondJSON(w, http.StatusOK, tasks)
}

// UpdateTask updates an existing task.
func (c *CRMHandlers) UpdateTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(models.UserIDContextKey).(int)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	vars := mux.Vars(r)
	taskID, err := strconv.Atoi(vars["id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid task ID")
		return
	}

	var task models.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	task.ID = taskID // Ensure the ID from the URL is used

	db := c.DB

	// Validate contact_id belongs to the user if provided in payload
	if task.ContactID != nil && *task.ContactID != 0 {
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM contacts WHERE id = ? AND user_id = ?)", *task.ContactID, userID).Scan(&exists)
		if err != nil || !exists {
			utils.RespondError(w, http.StatusForbidden, "Contact not found or does not belong to the user")
			return
		}
	}

	stmt, err := db.Prepare(`UPDATE tasks SET contact_id = ?, title = ?, description = ?, due_date = ?, status = ?, priority = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND user_id = ?`)
	if err != nil {
		log.Printf("Error preparing statement: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer stmt.Close()

	result, err := stmt.Exec(
		task.ContactID,
		task.Title,
		task.Description,
		task.DueDate,
		task.Status,
		task.Priority,
		task.ID,
		userID,
	)
	if err != nil {
		log.Printf("Error updating task: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Failed to update task")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		utils.RespondError(w, http.StatusNotFound, "Task not found or unauthorized to update")
		return
	}

	task.UpdatedAt = time.Now().Format(time.RFC3339)
	utils.RespondJSON(w, http.StatusOK, task)
}

// DeleteTask deletes a task.
func (c *CRMHandlers) DeleteTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(models.UserIDContextKey).(int)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	vars := mux.Vars(r)
	taskID, err := strconv.Atoi(vars["id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid task ID")
		return
	}

	db := c.DB
	result, err := db.Exec("DELETE FROM tasks WHERE id = ? AND user_id = ?", taskID, userID)
	if err != nil {
		log.Printf("Error deleting task: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Failed to delete task")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		utils.RespondError(w, http.StatusNotFound, "Task not found or unauthorized to delete")
		return
	}

	utils.RespondJSON(w, http.StatusNoContent, nil)
}
