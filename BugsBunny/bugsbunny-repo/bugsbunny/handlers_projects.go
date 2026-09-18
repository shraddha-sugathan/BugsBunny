package main

import (
	"context"	
    "encoding/json"
	"fmt"
	"log"
	"net/http"	
    "strconv"
)


func getProjectsHandler(w http.ResponseWriter, r *http.Request) {
    // We use COALESCE and check for NULL so older projects don't disappear
    query := `SELECT id, name, project_key, description, COALESCE(is_archived, false) 
              FROM projects 
              WHERE is_archived IS NULL OR is_archived = FALSE 
              ORDER BY id ASC`
              
    rows, err := dbpool.Query(context.Background(), query)
    if err != nil {
        http.Error(w, "Failed to fetch projects", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var projects []Project
    for rows.Next() {
        var p Project
        // We must scan exactly 5 fields to match the SQL query and the Project struct
        if err := rows.Scan(&p.ID, &p.Name, &p.ProjectKey, &p.Description, &p.IsArchived); err != nil {
            fmt.Println("Project scan error:", err)
            continue
        }
        projects = append(projects, p)
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(projects)
}

func createProjectHandler(w http.ResponseWriter, r *http.Request) {
    var p Project
    if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
        http.Error(w, "Invalid data", http.StatusBadRequest)
        return
    }

    err := dbpool.QueryRow(context.Background(),
        "INSERT INTO projects (project_key, name, description) VALUES ($1, $2, $3) RETURNING id",
        p.ProjectKey, p.Name, p.Description).Scan(&p.ID)
        
    if err != nil {
        http.Error(w, "Failed to create project", http.StatusInternalServerError)
        return
    }
    json.NewEncoder(w).Encode(p)
}

func archiveProjectHandler(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    _, err := dbpool.Exec(context.Background(), "UPDATE projects SET is_archived = TRUE WHERE id = $1", id)
    if err != nil {
        http.Error(w, "Failed to archive project", http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusOK)
}

func getAdminProjectsHandler(w http.ResponseWriter, r *http.Request) {
    rows, err := dbpool.Query(context.Background(), "SELECT id, name, project_key, description, is_archived FROM projects ORDER BY id ASC")
    if err != nil {
        http.Error(w, "Failed to fetch projects", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var projects []Project
    for rows.Next() {
        var p Project
        if err := rows.Scan(&p.ID, &p.Name, &p.ProjectKey, &p.Description, &p.IsArchived); err == nil {
            projects = append(projects, p)
        }
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(projects)
}

func handleGetProjectVersions(w http.ResponseWriter, r *http.Request) {
	projectID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	rows, err := dbpool.Query(context.Background(), 
		"SELECT id, name, version_type FROM project_versions WHERE project_id = $1 ORDER BY name DESC", projectID)
	if err != nil {
		log.Printf("Error fetching versions: %v\n", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	versions := make([]ProjectVersion, 0)
	for rows.Next() {
		var v ProjectVersion
		if err := rows.Scan(&v.ID, &v.Name, &v.VersionType); err == nil {
			versions = append(versions, v)
		}
	}
	json.NewEncoder(w).Encode(versions)
}
// POST /projects/{id}/versions
func handleCreateProjectVersion(w http.ResponseWriter, r *http.Request) {
	projectID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	var v ProjectVersion
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	if v.Name == "" || (v.VersionType != "release" && v.VersionType != "build") {
		http.Error(w, "Valid name and version_type ('release' or 'build') are required", http.StatusBadRequest)
		return
	}

	// Insert and scan the new ID back into the struct so Vue can render it immediately
	err = dbpool.QueryRow(context.Background(), `
		INSERT INTO project_versions (project_id, name, version_type) 
		VALUES ($1, $2, $3) RETURNING id
	`, projectID, v.Name, v.VersionType).Scan(&v.ID)

	if err != nil {
		log.Printf("Error inserting version: %v\n", err)
		http.Error(w, "Failed to create version (might be a duplicate)", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(v)
}

// DELETE /versions/{id}
func handleDeleteProjectVersion(w http.ResponseWriter, r *http.Request) {
	versionID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid version ID", http.StatusBadRequest)
		return
	}

	_, err = dbpool.Exec(context.Background(), "DELETE FROM project_versions WHERE id = $1", versionID)
	if err != nil {
		log.Printf("Error deleting version: %v\n", err)
		http.Error(w, "Failed to delete version", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func handleGetAdminWorkflowRules(w http.ResponseWriter, r *http.Request) {
	projectID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	rows, err := dbpool.Query(context.Background(), `
		SELECT t.id, fs.status_name as from_stage, ts.status_name as to_stage, 
		       COALESCE(r.name, 'Any Role') as role_name, t.requires_comment, t.requires_build_version
		FROM workflow_transitions t
		JOIN workflow_stages fs ON t.from_stage_id = fs.id
		JOIN workflow_stages ts ON t.to_stage_id = ts.id
		LEFT JOIN roles r ON t.required_role_id = r.id
		WHERE fs.project_id = $1
		ORDER BY fs.step_order ASC
	`, projectID)

	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type AdminRuleView struct {
		ID              int    `json:"id"`
		FromStage       string `json:"from_stage"`
		ToStage         string `json:"to_stage"`
		RoleName        string `json:"role_name"`
		RequiresComment bool   `json:"requires_comment"`
		RequiresBuild   bool   `json:"requires_build"`
	}

	var rules []AdminRuleView
	for rows.Next() {
		var rule AdminRuleView
		if err := rows.Scan(&rule.ID, &rule.FromStage, &rule.ToStage, &rule.RoleName, &rule.RequiresComment, &rule.RequiresBuild); err == nil {
			rules = append(rules, rule)
		}
	}
	
	json.NewEncoder(w).Encode(rules)
}

func handleGetStages(w http.ResponseWriter, r *http.Request) {
	// 1. Convert the URL parameter to an integer
	projectID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	rows, err := dbpool.Query(context.Background(), 
		"SELECT id, project_id, status_name, step_order, sla_hours FROM workflow_stages WHERE project_id = $1 ORDER BY step_order ASC", projectID)
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var stages []WorkflowStage
	for rows.Next() {
		var s WorkflowStage
		if err := rows.Scan(&s.ID, &s.ProjectID, &s.StatusName, &s.StepOrder, &s.SLAHours); err != nil {
			continue
		}
		stages = append(stages, s)
	}
	json.NewEncoder(w).Encode(stages)
}

func handleCreateStage(w http.ResponseWriter, r *http.Request) {
	// 1. Convert the URL parameter to an integer
	projectID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	var s WorkflowStage
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 2. Pass the strict integer to pgx
	err = dbpool.QueryRow(context.Background(),
		`INSERT INTO workflow_stages (project_id, status_name, step_order, sla_hours) 
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		projectID, s.StatusName, s.StepOrder, s.SLAHours).Scan(&s.ID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	s.ProjectID = projectID
	json.NewEncoder(w).Encode(s)
}

func handleCreateTransition(w http.ResponseWriter, r *http.Request) {
	// 1. Convert the URL parameter to an integer
	projectID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	var t WorkflowTransition
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 2. Pass the strict integer to pgx
	err = dbpool.QueryRow(context.Background(),
		`INSERT INTO workflow_transitions 
		 (project_id, from_stage_id, to_stage_id, required_role_id, requires_comment, requires_build_version) 
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		projectID, t.FromStageID, t.ToStageID, t.RequiredRoleID, t.RequiresComment, t.RequiresBuildVersion).Scan(&t.ID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	t.ProjectID = projectID
	json.NewEncoder(w).Encode(t)
}

