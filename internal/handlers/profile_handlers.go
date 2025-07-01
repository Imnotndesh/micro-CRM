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
	_, ok := r.Context().Value(models.UserIDContextKey).(int)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}
	var (
		response  models.GetUserResponse
		reqUserID models.GetUserPayload
	)
	if err := json.NewDecoder(r.Body).Decode(&reqUserID); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	c.Log.Info("Getting user info from : ", r.RemoteAddr)
	err := c.DB.QueryRow("SELECT id,username,email,first_name,last_name,created_at,updated_at  FROM users WHERE id=?").Scan(&response.ID, &response.Username, &response.Email, &response.FirstName, &response.LastName, &response.CreatedAt, &response.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		utils.RespondError(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		c.Log.Error("Error querying user : ", err.Error())
		utils.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondJSON(w, http.StatusOK, response)
}

// EditUserInfo Handler for user editing
func (c *CRMHandlers) EditUserInfo(w http.ResponseWriter, r *http.Request) {
	_, ok := r.Context().Value(models.UserIDContextKey).(int)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}
	var (
		currentUser models.User
		updateTime  string
		response    models.EditUserResponse
	)
	if err := json.NewDecoder(r.Body).Decode(&currentUser); err != nil {
		c.Log.Error("Error decoding user : ", err.Error())
		utils.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	// TODO: Workin on Dis
	c.Log.Info("Updating user info from : ", r.RemoteAddr)
	stmt, err := c.DB.Prepare("UPDATE users SET username = ?, email = ?, first_name = ?, last_name = ?, updated_at = ?  WHERE id = ?")
	if err != nil {
		c.Log.Error("Error preparing statement : ", err.Error())
		utils.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer stmt.Close()
	updateTime = time.Now().Format(time.RFC3339)
	res, err := stmt.Exec(
		currentUser.Username,
		currentUser.Email,
		currentUser.FirstName,
		currentUser.LastName,
		updateTime,
		currentUser.ID)
	if err != nil {
		c.Log.Error("Error updating user : ", err.Error())
		utils.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		utils.RespondError(w, http.StatusNotFound, "User Not found or unauthorized to update")
		return
	}
	response.UpdatedAt = updateTime
	response.Message = "Success editing user"
	utils.RespondJSON(w, http.StatusOK, response)
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
