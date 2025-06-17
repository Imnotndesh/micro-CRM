package handlers

import (
	"encoding/json"
	"micro-CRM/internal/models"
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
		FROM contacts 
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
		WHERE user_id = ? AND interaction_at >= date('now', '-7 days')
		GROUP BY DATE(interaction_at)
		ORDER BY date
	`, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var trends []models.InteractionTrend
	for rows.Next() {
		var trend models.InteractionTrend
		rows.Scan(&trend.Date, &trend.Calls, &trend.Emails, &trend.Meetings)
		trends = append(trends, trend)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"data": trends})
}
