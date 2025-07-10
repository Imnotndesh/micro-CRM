package handlers

import (
	"encoding/json"
	"micro-CRM/internal/models"
	"micro-CRM/internal/utils"
	"net/http"
)

func (c *CRMHandlers) GetDashboardStats(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(models.UserIDContextKey).(int)

	// Query your database for stats
	var stats models.DashboardStats

	// Example queries (adjust based on your database schema)
	c.DB.QueryRow("SELECT COUNT(*) FROM contacts WHERE user_id = ?", userID).Scan(&stats.TotalContacts)
	c.DB.QueryRow("SELECT COUNT(*) FROM companies WHERE user_id = ?", userID).Scan(&stats.TotalCompanies)
	c.DB.QueryRow("SELECT COUNT(*) FROM tasks WHERE user_id = ?", userID).Scan(&stats.TotalTasks)
	c.DB.QueryRow("SELECT COUNT(*) FROM tasks WHERE user_id = ? AND status = 'Done'", userID).Scan(&stats.CompletedTasks)
	c.DB.QueryRow("SELECT COUNT(*) FROM interactions WHERE user_id = ? AND interaction_at > datetime('now')", userID).Scan(&stats.UpcomingInteractions)
	c.DB.QueryRow("SELECT COUNT(*) FROM files WHERE user_id = ?", userID).Scan(&stats.FilesUploaded)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
func (c *CRMHandlers) GetPipelineData(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(models.UserIDContextKey).(int)

	rows, err := c.DB.Query(`
		SELECT pipeline_stage, COUNT(*) as count 
		FROM companies 
		WHERE user_id = ? 
		GROUP BY pipeline_stage
	`, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var pipelineData []models.PipelineStage
	colors := map[string]string{
		"Lead":        "#8884d8",
		"Qualified":   "#82ca9d",
		"Proposal":    "#ffc658",
		"Prospect":    "#aabbcc",
		"Negotiation": "#ff7300",
		"Closed Won":  "#00ff00",
		"Closed Lost": "#ff0000",
	}

	for rows.Next() {
		var stage models.PipelineStage
		rows.Scan(&stage.Stage, &stage.Count)
		stage.Color = colors[stage.Stage]
		pipelineData = append(pipelineData, stage)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"data": pipelineData})
}

// GetInteractionTrends returns interaction trends over time
func (c *CRMHandlers) GetInteractionTrends(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(models.UserIDContextKey).(int)

	rows, err := c.DB.Query(`
		SELECT
			DATE(interaction_at) as date,
			SUM(CASE WHEN type = 'Call' THEN 1 ELSE 0 END) as calls,
			SUM(CASE WHEN type = 'Email' THEN 1 ELSE 0 END) as emails,
			SUM(CASE WHEN type = 'Meeting' THEN 1 ELSE 0 END) as meetings
		FROM interactions
		WHERE user_id = ?
			AND DATE(interaction_at) >= DATE('now', '-30 days')
		GROUP BY DATE(interaction_at)
		ORDER BY DATE(interaction_at)
	`, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var trends []map[string]interface{}
	for rows.Next() {
		var date string
		var calls, emails, meetings int
		if err := rows.Scan(&date, &calls, &emails, &meetings); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		trends = append(trends, map[string]interface{}{
			"date":     date,
			"calls":    calls,
			"emails":   emails,
			"meetings": meetings,
		})
	}
	utils.RespondJSON(w, http.StatusOK, trends)
}
func (c *CRMHandlers) GetRecentInteractions(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(models.UserIDContextKey).(int)
	rows, err := c.DB.Query(`
		SELECT i.contact_id, c.first_name, c.last_name, i.type, i.description, i.duration, i.interaction_at
		FROM interactions i
		JOIN contacts c ON i.contact_id = c.id
		WHERE i.user_id = ?
		ORDER BY i.interaction_at DESC
		LIMIT 5
	`, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var recentInteractions []models.RecentInteraction
	for rows.Next() {
		var ri models.RecentInteraction
		err := rows.Scan(
			&ri.ContactID,
			&ri.FirstName,
			&ri.LastName,
			&ri.Type,
			&ri.Description,
			&ri.Duration,
			&ri.InteractionAt,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		recentInteractions = append(recentInteractions, ri)
	}

	w.Header().Set("Content-Type", "application/json")
	utils.RespondJSON(w, http.StatusOK, recentInteractions)
}
func (c *CRMHandlers) GetSuggestedContacts(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(models.UserIDContextKey).(int)

	query := `
        SELECT 
            c.id,
            c.first_name,
            c.last_name,
            c.email,
            co.name AS company,
            MIN(due_info.due) AS next_due
        FROM contacts c
        LEFT JOIN companies co ON c.company_id = co.id
        LEFT JOIN (
            SELECT contact_id, MIN(due_date) AS due FROM tasks 
            WHERE user_id = ? AND status != 'Done' AND due_date IS NOT NULL 
            GROUP BY contact_id
            UNION
            SELECT contact_id, MIN(follow_up_date) AS due FROM interactions 
            WHERE user_id = ? AND follow_up_date IS NOT NULL 
            GROUP BY contact_id
        ) AS due_info ON c.id = due_info.contact_id
        WHERE c.user_id = ?
        GROUP BY c.id
        HAVING next_due IS NOT NULL
        ORDER BY next_due ASC
        LIMIT 5;
    `

	rows, err := c.DB.Query(query, userID, userID, userID)
	if err != nil {
		http.Error(w, "Failed to query suggested contacts", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var suggestions []models.SuggestedContact
	for rows.Next() {
		var s models.SuggestedContact
		if err := rows.Scan(&s.ID, &s.FirstName, &s.LastName, &s.Email, &s.Company, &s.NextDue); err != nil {
			http.Error(w, "Failed to scan row", http.StatusInternalServerError)
			return
		}
		suggestions = append(suggestions, s)
	}

	w.Header().Set("Content-Type", "application/json")
	utils.RespondJSON(w, http.StatusOK, suggestions)
}
