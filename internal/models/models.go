package models

import "database/sql"

// User represents a user in the system.
type User struct {
	ID           int    `json:"id"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	PasswordHash string `json:"-"` // Don't expose password hash in JSON
	FirstName    string `json:"first_name,omitempty"`
	LastName     string `json:"last_name,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
	UpdatedAt    string `json:"updated_at,omitempty"`
}

// UserRegistrationPayload for incoming registration requests.
type UserRegistrationPayload struct {
	Username  string `json:"username"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// UserLoginPayload for incoming login requests.
type UserLoginPayload struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Company represents a company record.
type Company struct {
	ID            int     `json:"id"`
	UserID        int     `json:"user_id"` // Owner of the company record
	Name          string  `json:"name"`
	Website       *string `json:"website,omitempty"`
	Industry      *string `json:"industry,omitempty"`
	Address       *string `json:"address,omitempty"`
	PhoneNumber   *string `json:"phone_number,omitempty"`
	CreatedAt     string  `json:"created_at,omitempty"`
	UpdatedAt     string  `json:"updated_at,omitempty"`
	PipelineStage string  `json:"pipeline_stage,omitempty"`
}

// Contact represents a contact person.
type Contact struct {
	ID                    int     `json:"id"`
	UserID                int     `json:"user_id"` // Owner of the contact record
	CompanyID             *int    `json:"company_id,omitempty"`
	FirstName             string  `json:"first_name"`
	LastName              string  `json:"last_name"`
	Email                 *string `json:"email,omitempty"`
	PhoneNumber           *string `json:"phone_number,omitempty"`
	JobTitle              *string `json:"job_title,omitempty"`
	Notes                 *string `json:"notes,omitempty"`
	CreatedAt             string  `json:"created_at,omitempty"`
	UpdatedAt             string  `json:"updated_at,omitempty"`
	LastInteractionAt     *string `json:"last_interaction_at,omitempty"`
	NextActionAt          *string `json:"next_action_at,omitempty"`
	NextActionDescription *string `json:"next_action_description,omitempty"`
	PipelineStage         string  `json:"pipeline_stage,omitempty"`
}

// Interaction represents a recorded interaction with a contact.
type Interaction struct {
	ID            int     `json:"id"`
	UserID        int     `json:"user_id"`
	ContactID     int     `json:"contact_id"`
	Type          string  `json:"type"` // e.g., 'Call', 'Email', 'Meeting', 'Note'
	Description   *string `json:"description,omitempty"`
	InteractionAt *string `json:"interaction_at,omitempty"`
	CreatedAt     string  `json:"created_at,omitempty"`
}

// Task represents a task related to a contact or general.
type Task struct {
	ID          int     `json:"id"`
	UserID      int     `json:"user_id"`
	ContactID   *int    `json:"contact_id,omitempty"` // Can be NULL if not associated with a specific contact
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
	DueDate     *string `json:"due_date,omitempty"`
	Status      string  `json:"status,omitempty"`   // e.g., 'To Do', 'In Progress', 'Done'
	Priority    string  `json:"priority,omitempty"` // e.g., 'Low', 'Medium', 'High'
	CreatedAt   string  `json:"created_at,omitempty"`
	UpdatedAt   string  `json:"updated_at,omitempty"`
}

// File represents metadata for an uploaded file.
type File struct {
	ID          int     `json:"id"`
	UserID      int     `json:"user_id"`
	ContactID   *int    `json:"contact_id,omitempty"`
	CompanyID   *int    `json:"company_id,omitempty"`
	FileName    string  `json:"file_name"`
	StoragePath string  `json:"storage_path"` // Path on the server's filesystem
	FileType    *string `json:"file_type,omitempty"`
	FileSize    *int    `json:"file_size,omitempty"` // In bytes
	UploadedAt  string  `json:"uploaded_at,omitempty"`
}
type EnvParams struct {
	DbPath   string
	JWTToken string
	ApiPort  string
}
type Handlers struct {
	Db *sql.DB
}

// ContextKey for storing user ID in context.
type ContextKey string

const UserIDContextKey ContextKey = "userID"

// DashboardStats represents dashboard statistics
type DashboardStats struct {
	TotalContacts        int `json:"totalContacts"`
	TotalCompanies       int `json:"totalCompanies"`
	TotalTasks           int `json:"totalTasks"`
	CompletedTasks       int `json:"completedTasks"`
	UpcomingInteractions int `json:"upcomingInteractions"`
	FilesUploaded        int `json:"filesUploaded"`
}

// PipelineStage represents pipeline distribution data
type PipelineStage struct {
	Stage string `json:"stage"`
	Count int    `json:"count"`
	Color string `json:"color"`
}

// InteractionTrend represents daily interaction trends
type InteractionTrend struct {
	Date     string `json:"date"`
	Calls    int    `json:"calls"`
	Emails   int    `json:"emails"`
	Meetings int    `json:"meetings"`
}
