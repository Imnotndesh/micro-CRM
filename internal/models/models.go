package models

import (
	"database/sql"
)

// User represents a user in the system.
type User struct {
	ID           int    `json:"id"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	PasswordHash string `json:"-,omitempty"`
	FirstName    string `json:"first_name,omitempty"`
	LastName     string `json:"last_name,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
	UpdatedAt    string `json:"updated_at,omitempty"`
}

// GetUserPayload payload for GetUserinfo handler
type GetUserPayload struct {
	ID int `json:"id"`
}

// GetUserResponse response struct for GetUserInfo Function in profile handler
type GetUserResponse struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// EditUserPayload expected payload for edit profile handler
type EditUserPayload struct {
	Username    string `json:"username"`
	Email       string `json:"email"`
	FirstName   string `json:"first_name,omitempty"`
	LastName    string `json:"last_name,omitempty"`
	NewPassword string `json:"new_password,omitempty"`
}

// UpdateUserResponse response for UpdateUserInfo handler
type UpdateUserResponse struct {
	Message   string `json:"message"`
	UpdatedAt string `json:"updated_at"`
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

// UserDeleteResponse profile delete response
type UserDeleteResponse struct {
	Message string `json:"message"`
}

// Company represents a company record.
type Company struct {
	ID            int     `json:"id" db:"id"`
	UserID        int     `json:"user_id" db:"user_id"`
	Name          string  `json:"name" db:"name"`
	Website       *string `json:"website,omitempty" db:"website"`
	Industry      *string `json:"industry,omitempty" db:"industry"`
	Notes         *string `json:"notes,omitempty" db:"notes"`
	CompanySize   *int    `json:"company_size,omitempty" db:"company_size"`
	Address       *string `json:"address,omitempty" db:"address"`
	PhoneNumber   *string `json:"phone_number,omitempty" db:"phone_number"`
	CreatedAt     string  `json:"created_at" db:"created_at"`
	UpdatedAt     string  `json:"updated_at" db:"updated_at"`
	PipelineStage string  `json:"pipeline_stage" db:"pipeline_stage"`
}

// Contact represents a contact person.
type Contact struct {
	ID                    int     `json:"id"`
	UserID                int     `json:"user_id"`
	CompanyID             *int    `json:"company_id,omitempty"` // Nullable FK
	FirstName             string  `json:"first_name"`
	LastName              string  `json:"last_name"`
	Email                 *string `json:"email,omitempty"`
	PhoneNumber           *string `json:"phone_number,omitempty"`
	JobTitle              *string `json:"job_title,omitempty"`
	Notes                 *string `json:"notes,omitempty"`
	CreatedAt             string  `json:"created_at"`
	UpdatedAt             string  `json:"updated_at"`
	LastInteractionAt     *string `json:"last_interaction_at,omitempty"`
	NextActionAt          *string `json:"next_action_at,omitempty"`
	NextActionDescription *string `json:"next_action_description,omitempty"`
	PipelineStage         *string `json:"pipeline_stage,omitempty"`
}

// Interaction represents a recorded interaction with a contact.
type Interaction struct {
	ID            int     `json:"id"`
	UserID        int     `json:"user_id"`
	ContactID     int     `json:"contact_id"`
	Type          string  `json:"type"`                     // e.g., "call", "email", etc.
	Subject       string  `json:"subject"`                  // Required
	Duration      int     `json:"duration"`                 // Default: 0
	Outcome       string  `json:"outcome"`                  // Default: "none"
	FollowUp      int     `json:"follow_up"`                // 0 or 1 (boolean-ish)
	Description   *string `json:"description,omitempty"`    // Optional
	InteractionAt *string `json:"interaction_at"`           // Default: CURRENT_TIMESTAMP
	FollowUpDate  *string `json:"follow_up_date,omitempty"` // Optional
	CreatedAt     string  `json:"created_at"`               // Default: CURRENT_TIMESTAMP
}

// RecentInteraction Dashboard recent interactions
type RecentInteraction struct {
	ContactID     int     `json:"contact_id"`
	FirstName     string  `json:"first_name"`
	LastName      string  `json:"last_name"`
	Type          string  `json:"type"`
	Description   *string `json:"description,omitempty"`
	Duration      *int    `json:"duration,omitempty"`
	InteractionAt *string `json:"interaction_at,omitempty"`
}

type SuggestedContact struct {
	ID        int     `json:"id"`
	FirstName string  `json:"first_name"`
	LastName  string  `json:"last_name"`
	Email     *string `json:"email,omitempty"`
	Company   *string `json:"company,omitempty"`
	NextDue   *string `json:"next_due,omitempty"`
}

// Task represents a task related to a contact or general.
type Task struct {
	ID          int     `json:"id"`
	UserID      int     `json:"user_id"`
	ContactID   *int    `json:"contact_id,omitempty"` // nullable foreign key
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
	DueDate     *string `json:"due_date,omitempty"`
	Status      string  `json:"status"`
	Priority    string  `json:"priority"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

// File represents metadata for an uploaded file.
type File struct {
	ID            int     `json:"id"`
	UserID        int     `json:"user_id"`
	ContactID     *int    `json:"contact_id,omitempty"`
	CompanyID     *int    `json:"company_id,omitempty"`
	FileName      string  `json:"file_name"`
	StoragePath   string  `json:"storage_path"` // Path on the server's filesystem
	FileType      *string `json:"file_type,omitempty"`
	FileSize      *int    `json:"file_size,omitempty"` // In bytes
	UploadedAt    string  `json:"uploaded_at,omitempty"`
	InteractionID *int    `json:"interaction_id,omitempty"`
}
type EnvParams struct {
	DbPath       string
	JWTToken     string
	ApiPort      string
	KeyFilePath  string
	CertFilePath string
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
	PendingTasks         int `json:"pendingTasks"`
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

const (
	DefaultDBPath   = "/app/database/micro-crm.db"
	DefaultKeyPath  = "./certs/micro-crm-key.pem"
	DefaultCertPath = "./certs/micro-crm-cert.pem"
	DefaultApiPort  = "9080"
)
