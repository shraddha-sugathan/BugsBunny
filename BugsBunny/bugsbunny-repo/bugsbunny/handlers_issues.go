package main

import (
	"context"	
    "encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"io"
	"time"
	"strings"
	"path/filepath"
)

func createIssueHandler(w http.ResponseWriter, r *http.Request) {
	var issue Issue
	if err := json.NewDecoder(r.Body).Decode(&issue); err != nil {
        log.Printf("[ISSUE ERROR] JSON decode failed: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
    log.Printf("[ISSUE DEBUG] Received payload: ProjectID=%d, Title=%s, Key=%s", 
        issue.ProjectID, issue.Title, issue.IssueKey)

	// Extract the dynamic User ID from the context injected by the middleware
	// If middleware stores *Claims:
    claims, ok := r.Context().Value(claimsKey).(*Claims)
    if !ok || claims == nil {
        http.Error(w, `{"error": "Unauthorized: user identity missing"}`, http.StatusUnauthorized)
        return
    }
    userID := claims.UserID

	// Create the custom data (Priority)
	customData := fmt.Sprintf(`{"priority": "%s", "component": "%s", "module": "%s"}`, 
		issue.Priority, issue.Component, issue.Module)

	// 1. Dynamically find the initial stage ID for this project
	var initialStageID int
	err := dbpool.QueryRow(context.Background(), `
		SELECT id FROM workflow_stages 
		WHERE project_id = $1 AND (status_name = 'Open' OR status_name = 'open') 
		LIMIT 1
	`, issue.ProjectID).Scan(&initialStageID)

	if err != nil {
		err = dbpool.QueryRow(context.Background(), `
			SELECT id FROM workflow_stages WHERE project_id = $1 ORDER BY step_order ASC LIMIT 1
		`, issue.ProjectID).Scan(&initialStageID)
		if err != nil {
            log.Printf("[ISSUE ERROR] Workflow stage query failed for ProjectID %d: %v", issue.ProjectID, err)
			http.Error(w, "Project workflow not configured", http.StatusInternalServerError)
			return
		}
	}

	// 2. Run the INSERT using the dynamic initialStageID!
	_, err = dbpool.Exec(context.Background(), `
		INSERT INTO issues (
			project_id, reporter_id, current_stage_id, issue_key, 
			title, body, custom_data, target_release_id, fixed_in_build_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		issue.ProjectID, userID, initialStageID, issue.IssueKey, 
		issue.Title, issue.Body, customData, 
		issue.TargetReleaseID, issue.FixedInBuildID)

	if err != nil {
		fmt.Println("❌ DATABASE INSERT ERROR:", err)         
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "issue_key": issue.IssueKey})
}

func getAllIssuesHandler(w http.ResponseWriter, r *http.Request) {
    search := r.URL.Query().Get("search")
    priorityFilter := r.URL.Query().Get("priority")
    statusFilter := r.URL.Query().Get("status")
    assigneeFilter := r.URL.Query().Get("assignee")

    query := `
        SELECT i.issue_key, i.title, COALESCE(ws.status_name, 'Open') as status, 
               COALESCE(i.custom_data->>'priority', 'none') as priority, 
               COALESCE(u.email, 'Unassigned') as assignee_email
        FROM issues i
        LEFT JOIN users u ON i.assignee_id = u.id
        LEFT JOIN workflow_stages ws ON i.current_stage_id = ws.id
        WHERE 1=1
    `
    var args []interface{}
    var argIndex = 1

    if search != "" {
        query += fmt.Sprintf(" AND (i.title ILIKE $%d OR i.body ILIKE $%d)", argIndex, argIndex)
        args = append(args, "%"+search+"%")
        argIndex++
    }

    if priorityFilter != "" && priorityFilter != "all" {
        query += fmt.Sprintf(" AND i.custom_data->>'priority' = $%d", argIndex)
        args = append(args, priorityFilter)
        argIndex++
    }

    if statusFilter != "" && statusFilter != "all" {
        query += fmt.Sprintf(" AND i.status = $%d", argIndex)
        args = append(args, statusFilter)
        argIndex++
    }

    if assigneeFilter != "" && assigneeFilter != "all" {
        if assigneeFilter == "unassigned" {
            query += " AND i.assignee_id IS NULL"
        } else {
            query += fmt.Sprintf(" AND i.assignee_id = $%d", argIndex)
            args = append(args, assigneeFilter)
            argIndex++
        }
    }
    
    query += " ORDER BY i.issue_key DESC"

    rows, err := dbpool.Query(context.Background(), query, args...)
    if err != nil {
        fmt.Println("❌ DATABASE QUERY ERROR:", err)
        http.Error(w, "Failed to fetch issues", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var items []QueueItem
    for rows.Next() {
        var item QueueItem
        if err := rows.Scan(&item.IssueKey, &item.Title, &item.Status, &item.Priority, &item.AssigneeEmail); err != nil {
            continue
        }
        items = append(items, item)
    }

    if items == nil {
        items = []QueueItem{}
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(items)
}

// The new handler to update an existing issue
func updateIssueHandler(w http.ResponseWriter, r *http.Request) {
    issueKey := r.PathValue("id")

    var req UpdateIssueRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // 1. Convert AssigneeID 0 to a SQL NULL
    var assigneeIDPtr *int
    if req.AssigneeID > 0 {
        assigneeIDPtr = &req.AssigneeID
    } 

    // 2. Convert TargetReleaseID 0 to a SQL NULL
    var targetReleaseIDPtr *int
    if req.TargetReleaseID > 0 {
        targetReleaseIDPtr = &req.TargetReleaseID
    }

    // 3. Convert FixedInBuildID 0 to a SQL NULL
    var fixedInBuildIDPtr *int
    if req.FixedInBuildID > 0 {
        fixedInBuildIDPtr = &req.FixedInBuildID
    }

    // Pack the dynamic fields into a map so we can insert it into the JSONB column
    customDataMap := map[string]string{
        "priority":  req.Priority,
        "component": req.Component,
        "module":    req.Module,
    }
    customDataBytes, err := json.Marshal(customDataMap)
    if err != nil {
        http.Error(w, "Failed to encode custom data", http.StatusInternalServerError)
        return
    }

    // Execute the update with the new ID columns
    _, err = dbpool.Exec(context.Background(), `
        UPDATE issues 
        SET title = $1, 
            body = $2, 
            assignee_id = $3, 
            target_release_id = $4, 
            fixed_in_build_id = $5,
            custom_data = $6, 
            updated_at = CURRENT_TIMESTAMP
        WHERE issue_key = $7
    `, req.Title, req.Body, assigneeIDPtr, targetReleaseIDPtr, fixedInBuildIDPtr, customDataBytes, issueKey)

    if err != nil {
        log.Printf("Error updating issue: %v\n", err)
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
}

func deleteIssueHandler(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    
    // 1. Fetch attachments to clean up the physical hard drive
    rows, _ := dbpool.Query(context.Background(), "SELECT file_url FROM issue_attachments WHERE issue_key = $1", id)
    for rows.Next() {
        var fileURL string
        if rows.Scan(&fileURL) == nil {
            // Extract "123_image.png" from "http://localhost:8080/uploads/123_image.png"
            parts := strings.Split(fileURL, "/")
            diskFilename := parts[len(parts)-1]
            
            // Delete the physical file from the OS!
            os.Remove(filepath.Join("uploads", diskFilename)) 
        }
    }
    rows.Close() // Close the rows before executing the next query

    // 2. Delete the issue (PostgreSQL ON DELETE CASCADE will handle the DB rows)
    _, err := dbpool.Exec(context.Background(), "DELETE FROM issues WHERE issue_key = $1", id)
    if err != nil {
        http.Error(w, "Failed to delete issue", http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusNoContent)
}

func getIssueHandler(w http.ResponseWriter, r *http.Request) {
    // 1. Rename 'id' to 'issueKey' so it matches the query below
    issueKey := r.PathValue("id")
    var issue Issue
    var customDataBytes []byte
    var projectKey string 
    var currentStageID *int 

    query := `
        SELECT i.issue_key, i.project_id, COALESCE(i.title, ''), COALESCE(i.body, ''), 
               COALESCE(ws.status_name, 'Open'), i.custom_data, p.project_key, 
               i.assignee_id, COALESCE(u.email, 'Unassigned'), 
               i.target_release_id, COALESCE(tr.name, 'N/A'),
               i.fixed_in_build_id, COALESCE(fb.name, 'N/A'),
               i.current_stage_id
        FROM issues i 
        JOIN projects p ON i.project_id = p.id
        LEFT JOIN users u ON i.assignee_id = u.id
        LEFT JOIN workflow_stages ws ON i.current_stage_id = ws.id
        LEFT JOIN project_versions tr ON i.target_release_id = tr.id
        LEFT JOIN project_versions fb ON i.fixed_in_build_id = fb.id
        WHERE i.issue_key = $1`

    err := dbpool.QueryRow(context.Background(), query, issueKey).Scan(
        &issue.IssueKey, &issue.ProjectID, &issue.Title, &issue.Body, 
        &issue.Status, &customDataBytes, &projectKey, // 2. Changed &issue.CustomData to &customDataBytes
        &issue.AssigneeID, &issue.AssigneeEmail, 
        &issue.TargetReleaseID, &issue.TargetRelease,
        &issue.FixedInBuildID, &issue.FixedInBuild,
        &currentStageID,
    )
    if err != nil {
        fmt.Println("❌ DB Scan Error in getIssueHandler:", err) 
        http.Error(w, "Issue not found", http.StatusNotFound)
        return
    }
    
    // 3. Safely parse custom data if it exists
    if len(customDataBytes) > 0 {
        json.Unmarshal(customDataBytes, &issue.CustomData)
    }

    // 4. STRICT WORKFLOW ENGINE: Query only the transitions explicitly allowed by the database
    var allowed []string
    if currentStageID != nil {
        rows, err := dbpool.Query(context.Background(), `
            SELECT ts.status_name 
            FROM workflow_transitions t
            JOIN workflow_stages ts ON t.to_stage_id = ts.id
            WHERE t.from_stage_id = $1
        `, *currentStageID)

        if err == nil {
            defer rows.Close()
            for rows.Next() {
                var targetStageName string
                if err := rows.Scan(&targetStageName); err == nil {
                    allowed = append(allowed, targetStageName)
                }
            }
        }
    }
    issue.AllowedTransitions = allowed 

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(issue)
}

// --- HANDLER 1: Get Available Transitions ---
func handleGetIssueTransitions(w http.ResponseWriter, r *http.Request) {
    issueKey := r.PathValue("id")
    
    // 1. Find the issue's current stage
    var currentStageID *int
    err := dbpool.QueryRow(context.Background(), "SELECT current_stage_id FROM issues WHERE issue_key = $1", issueKey).Scan(&currentStageID)
    if err != nil || currentStageID == nil {
        json.NewEncoder(w).Encode([]TransitionOption{}) // Return empty if no stage set yet
        return
    }

    // 2. Query valid transitions from this stage (ADDED t.required_role_id)
    rows, err := dbpool.Query(context.Background(), `
        SELECT t.id, t.to_stage_id, s.status_name, t.requires_comment, t.requires_build_version, t.required_role_id
        FROM workflow_transitions t
        JOIN workflow_stages s ON t.to_stage_id = s.id
        WHERE t.from_stage_id = $1
    `, currentStageID)
    
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var options []TransitionOption
    for rows.Next() {
        var opt TransitionOption
        // ADDED &opt.RequiredRoleID to the Scan
        if err := rows.Scan(&opt.ID, &opt.ToStageID, &opt.ToStageName, &opt.RequiresComment, &opt.RequiresBuildVersion, &opt.RequiredRoleID); err == nil {
            options = append(options, opt)
        }
    }
    json.NewEncoder(w).Encode(options)
}

// --- HANDLER 2: Execute Transition & Save History ---
func handleTransitionIssue(w http.ResponseWriter, r *http.Request) {
    issueKey := r.PathValue("id")
    
    userID, ok := r.Context().Value("userID").(float64) 
    if !ok {
        if intID, okInt := r.Context().Value("userID").(int); okInt {
            userID = float64(intID)
        } else {
            http.Error(w, "Unauthorized: Invalid token data", http.StatusUnauthorized)
            return
        }
    }
    realUserID := int(userID)

    var req TransitionRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    var previousStageID *int
    err := dbpool.QueryRow(context.Background(), 
        "SELECT current_stage_id FROM issues WHERE issue_key = $1", issueKey).Scan(&previousStageID)
    
    if err != nil || previousStageID == nil {
        http.Error(w, "Issue not found or has no current stage", http.StatusBadRequest)
        return
    }

    var requiredRoleID *int
    err = dbpool.QueryRow(context.Background(), `
        SELECT required_role_id 
        FROM workflow_transitions 
        WHERE from_stage_id = $1 AND to_stage_id = $2
    `, previousStageID, req.NewStageID).Scan(&requiredRoleID)

    if err != nil {
        http.Error(w, "Invalid transition path requested", http.StatusBadRequest)
        return
    }

    if requiredRoleID != nil && *requiredRoleID != 0 {
        var userRoleID *int
        err = dbpool.QueryRow(context.Background(), 
            "SELECT role_id FROM users WHERE id = $1", realUserID).Scan(&userRoleID)
        
        if err != nil || userRoleID == nil || *userRoleID != *requiredRoleID {
            http.Error(w, "Forbidden: You do not have the required role to perform this transition.", http.StatusForbidden)
            return
        }
    }

    var fixedInBuildIDPtr *int
    if req.FixedInBuildID > 0 {
        fixedInBuildIDPtr = &req.FixedInBuildID
    }
    
    // FIX: Changed fixed_in_build to fixed_in_build_id
    _, err = dbpool.Exec(context.Background(), `
        UPDATE issues 
        SET current_stage_id = $1, fixed_in_build_id = $2, updated_at = CURRENT_TIMESTAMP
        WHERE issue_key = $3
    `, req.NewStageID, fixedInBuildIDPtr, issueKey)

    if err != nil {
        log.Printf("Error updating issue: %v\n", err)
        http.Error(w, "Failed to update issue stage", http.StatusInternalServerError)
        return
    }

    // FIX: Changed fixed_in_build to fixed_in_build_id
    _, err = dbpool.Exec(context.Background(), `
        INSERT INTO issue_history (issue_key, user_id, previous_stage_id, new_stage_id, comment, fixed_in_build_id)
        VALUES ($1, $2, $3, $4, $5, $6)
    `, issueKey, realUserID, previousStageID, req.NewStageID, req.Comment, fixedInBuildIDPtr)

    if err != nil {
        log.Printf("Error writing history: %v\n", err)
    }

    w.WriteHeader(http.StatusOK)
}


func getCommentsHandler(w http.ResponseWriter, r *http.Request) {
    issueKey := r.PathValue("id")
    
    // We JOIN with the users table to get the email address of the commenter
    query := `
        SELECT c.id, c.issue_key, c.user_id, u.email, c.body, c.created_at 
        FROM issue_comments c
        JOIN users u ON c.user_id = u.id
        WHERE c.issue_key = $1
        ORDER BY c.created_at ASC
    `
    
    rows, err := dbpool.Query(context.Background(), query, issueKey)
    if err != nil {
        http.Error(w, "Failed to fetch comments", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var comments []Comment
    for rows.Next() {
        var c Comment
        if err := rows.Scan(&c.ID, &c.IssueKey, &c.UserID, &c.UserEmail, &c.Body, &c.CreatedAt); err != nil {
            continue
        }
        comments = append(comments, c)
    }
    
    if comments == nil { comments = []Comment{} }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(comments)
}

func addCommentHandler(w http.ResponseWriter, r *http.Request) {
    issueKey := r.PathValue("id")
    userID := r.Context().Value("userID").(int) // Extracted from JWT!

    var c Comment
    if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
        http.Error(w, "Invalid data", http.StatusBadRequest)
        return
    }

    err := dbpool.QueryRow(context.Background(),
        "INSERT INTO issue_comments (issue_key, user_id, body) VALUES ($1, $2, $3) RETURNING id, created_at",
        issueKey, userID, c.Body).Scan(&c.ID, &c.CreatedAt)
        
    if err != nil {
        http.Error(w, "Failed to add comment", http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusCreated)
}

func getAttachmentsHandler(w http.ResponseWriter, r *http.Request) {
    issueKey := r.PathValue("id")
    
    rows, err := dbpool.Query(context.Background(), "SELECT id, issue_key, filename, file_url, created_at FROM issue_attachments WHERE issue_key = $1 ORDER BY created_at DESC", issueKey)
    if err != nil {
        http.Error(w, "Failed to fetch attachments", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var attachments []Attachment
    for rows.Next() {
        var a Attachment
        if err := rows.Scan(&a.ID, &a.IssueKey, &a.Filename, &a.FileURL, &a.CreatedAt); err != nil {
            continue
        }
        attachments = append(attachments, a)
    }
    
    if attachments == nil { attachments = []Attachment{} }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(attachments)
}

func uploadAttachmentHandler(w http.ResponseWriter, r *http.Request) {
    issueKey := r.PathValue("id")

    // Parse the incoming multipart form (max 10 MB)
    r.ParseMultipartForm(10 << 20)

    // Retrieve the file from form data
    file, handler, err := r.FormFile("file")
    if err != nil {
        http.Error(w, "Error retrieving file", http.StatusBadRequest)
        return
    }
    defer file.Close()

    // Create a unique filename and path
    safeFilename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), handler.Filename)
    filePath := filepath.Join("uploads", safeFilename)
    fileURL := "http://localhost:8080/uploads/" + safeFilename

    // Save the file physically to the disk
    dst, err := os.Create(filePath)
    if err != nil {
        http.Error(w, "Error saving file", http.StatusInternalServerError)
        return
    }
    defer dst.Close()
    io.Copy(dst, file)

    // Save the metadata to PostgreSQL
    _, err = dbpool.Exec(context.Background(), 
        "INSERT INTO issue_attachments (issue_key, filename, file_url) VALUES ($1, $2, $3)", 
        issueKey, handler.Filename, fileURL)

    if err != nil {
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusCreated)
}
