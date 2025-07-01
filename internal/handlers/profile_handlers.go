package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"micro-CRM/internal/models"
	"micro-CRM/internal/utils"
	"net/http"
	"time"
)

// GetUserInfo Handler for fetching user information
func (c *CRMHandlers) GetUserInfo(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(models.UserIDContextKey).(int)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}
	var response models.User
	err := c.DB.QueryRow("SELECT username,email,first_name,last_name,created_at,updated_at FROM users WHERE id = ?", userID).Scan(&response.Username, &response.Email, &response.FirstName, &response.LastName, &response.CreatedAt, &response.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		utils.RespondError(w, http.StatusNotFound, "User not found")
		return
	}
	response.ID = userID
	if err != nil {
		c.Log.Fatal("Error getting user", err.Error())
		utils.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondJSON(w, http.StatusOK, response)
}

// UpdateUserInfo Handler for user editing
func (c *CRMHandlers) UpdateUserInfo(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(models.UserIDContextKey).(int)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}
	var (
		updateRequest  models.EditUserPayload
		updateResponse models.UpdateUserResponse
	)
	err := json.NewDecoder(r.Body).Decode(&updateRequest)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	stmt, err := c.DB.Prepare("UPDATE users SET username = ?,email = ?,first_name = ?,last_name = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?")
	if err != nil {
		c.Log.Fatal("Error preparing statement", err.Error())
		utils.RespondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer stmt.Close()
	res, err := stmt.Exec(
		updateRequest.Username,
		updateRequest.Email,
		updateRequest.FirstName,
		updateRequest.LastName,
		userID,
	)
	if err != nil {
		c.Log.Fatal("Error updating user: ", err.Error())
		utils.RespondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		utils.RespondError(w, http.StatusNotFound, "User not found")
		return
	}
	updateResponse = models.UpdateUserResponse{
		Message:   "User updated",
		UpdatedAt: time.Now().Format(time.RFC3339),
	}
	utils.RespondJSON(w, http.StatusOK, updateResponse)
}
func (c *CRMHandlers) DeleteUser(w http.ResponseWriter, r *http.Request) {
	UserID, ok := r.Context().Value(models.UserIDContextKey).(int)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}
	var (
		deleteResponse models.UserDeleteResponse
	)
	res, err := c.DB.Exec("DELETE FROM users WHERE id=?", UserID)
	if err != nil {
		c.Log.Error("Error deleting user : ", err.Error())
		utils.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.Log.Error("Error deleting user : ", UserID)
		utils.RespondError(w, http.StatusNotFound, "User Not found or unauthorized to delete")
		return
	}
	deleteResponse.Message = "Success deleting user"
	utils.RespondJSON(w, http.StatusOK, deleteResponse)
}
