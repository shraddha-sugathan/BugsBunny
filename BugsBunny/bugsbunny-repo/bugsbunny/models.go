package main

import (
    "encoding/json"
	"time"
)


type LoginRequest struct {
    Email    string `json:"email"`
    Password string `json:"password"`
}

type User struct {
    ID           int    `json:"id"`
    Email        string `json:"email"`
    PasswordHash string `json:"-"` // The hyphen ensures passwords NEVER leak to the Vue frontend!
    Role         string `json:"role"`
    IsActive     bool   `json:"is_active"`
}

type Project struct {
    ID          int    `json:"id"`
    ProjectKey  string `json:"project_key"`
    Name        string `json:"name"`
    Description string `json:"description"`
    IsArchived  bool   `json:"is_archived"`
}

type Comment struct {
    ID        int       `json:"id"`
    IssueKey  string    `json:"issue_key"`
    UserID    int       `json:"user_id"`
    UserEmail string    `json:"user_email"` 
    Body      string    `json:"body"`
    CreatedAt time.Time `json:"created_at"`
}

type Attachment struct {
    ID        int       `json:"id"`
    IssueKey  string    `json:"issue_key"`
    Filename  string    `json:"filename"`
    FileURL   string    `json:"file_url"`
    CreatedAt time.Time `json:"created_at"`
}

// This handles both reading from the DB and receiving from Vue
type Issue struct {
    IssueKey           string          `json:"issue_key"`
    ProjectID          int             `json:"project_id"`
    Title              string          `json:"title"`
    Body               string          `json:"body"`
    Status             string          `json:"status"`
    AllowedTransitions []string        `json:"allowed_transitions"`
    Resolution         *string         `json:"resolution"` 
    ReporterID         int             `json:"reporter_id"`
    AssigneeID         *int            `json:"assignee_id"` 
    CustomData         json.RawMessage `json:"custom_data"` 
    AssigneeEmail      string          `json:"assignee_email"`
    
    // --- The Real Database Links ---
    TargetReleaseID    *int            `json:"target_release_id,omitempty"` 
    FixedInBuildID     *int            `json:"fixed_in_build_id,omitempty"` 

    // --- Display Strings (Sent to Vue so it shows names, not numbers) ---
    TargetRelease      string          `json:"target_release"`
    FixedInBuild       string          `json:"fixed_in_build,omitempty"` 
    
    // --- Virtual Fields (Catches the form data from Vue) ---
    Priority           string          `json:"priority,omitempty"`
    Component          string          `json:"component,omitempty"`
    Module             string          `json:"module,omitempty"`
    Release            string          `json:"release,omitempty"`
    AttachmentCount    int             `json:"attachment_count"`
}

type QueueItem struct {
    IssueKey      string `json:"issue_key"`
    Title         string `json:"title"`
    Status        string `json:"status"`
    Priority      string `json:"priority"`
    AssigneeEmail string `json:"assignee_email"`
}

type UserProfile struct {
    Email         string      `json:"email"`
    TotalIssues   int         `json:"total_issues"`
    ResolvedCount int         `json:"resolved_count"`
    Issues        []IssueStub `json:"issues"`
}

type IssueStub struct {
    IssueKey string `json:"issue_key"`
    Title    string `json:"title"`
    Status   string `json:"status"`
}

type UserStub struct {
    ID    int    `json:"id"`
    Email string `json:"email"`
}

// --- WORKFLOW STRUCTS ---

type Role struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type WorkflowStage struct {
	ID         int    `json:"id"`
	ProjectID  int    `json:"project_id"`
	StatusName string `json:"status_name"`
	StepOrder  int    `json:"step_order"`
	SLAHours   int    `json:"sla_hours"`
}

type WorkflowTransition struct {
	ID                   int  `json:"id"`
	ProjectID            int  `json:"project_id"`
	FromStageID          int  `json:"from_stage_id"`
	ToStageID            int  `json:"to_stage_id"`
	RequiredRoleID       *int `json:"required_role_id"` // Pointer because it can be null
	RequiresComment      bool `json:"requires_comment"`
	RequiresBuildVersion bool `json:"requires_build_version"`
}

// --- NEW STRUCTS ---
type TransitionOption struct {
	ID                   int    `json:"id"`
	ToStageID            int    `json:"to_stage_id"`
	ToStageName          string `json:"to_stage_name"`
	RequiresComment      bool   `json:"requires_comment"`
	RequiresBuildVersion bool   `json:"requires_build_version"`
	RequiredRoleID       *int   `json:"required_role_id"`
}

type TransitionRequest struct {
	NewStageID     int    `json:"new_stage_id"`
	Comment        string `json:"comment"`
	FixedInBuildID int    `json:"fixed_in_build_id"`
}

type ProjectVersion struct {
    ID              int    `json:"id"`
    Name            string `json:"name"`
    VersionType     string `json:"version_type"`
    ParentReleaseID *int   `json:"parent_release_id,omitempty"` // New: Links build to release
}

// 1. Create a struct to match the Vue edit form EXACTLY
type UpdateIssueRequest struct {
	Title         string `json:"title"`
	Body          string `json:"body"`
	Priority      string `json:"priority"`
	AssigneeID    int    `json:"assignee_id"`
	Component     string `json:"component"`
	Module        string `json:"module"`
	//TargetRelease string `json:"target_release"`
	TargetReleaseID int    `json:"target_release_id"`
    FixedInBuildID  int    `json:"fixed_in_build_id"`
}
