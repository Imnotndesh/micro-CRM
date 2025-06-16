package handlers

import (
	"micro-CRM/internal/utils"
	"net/http"
)

func (c *CRMHandlers) Hello(w http.ResponseWriter, r *http.Request) {
	utils.RespondJSON(w, http.StatusOK, "OK")
}
func (c *CRMHandlers) DBPing(w http.ResponseWriter, r *http.Request) {
	err := c.DB.Ping()
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondJSON(w, http.StatusOK, "OK")
}
