package main

import (
	"context"	
    "encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
    "strconv"
)


func handleGetRoles(w http.ResponseWriter, r *http.Request) {
	rows, err := dbpool.Query(context.Background(), "SELECT id, name FROM roles")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	roles := make([]Role, 0)
	for rows.Next() {
		var role Role
		if err := rows.Scan(&role.ID, &role.Name); err != nil {
			continue
		}
		roles = append(roles, role)
	}
	json.NewEncoder(w).Encode(roles)
}

func handleCreateRole(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	roleName := strings.TrimSpace(input.Name)
	if roleName == "" {
		http.Error(w, "Role name cannot be empty", http.StatusBadRequest)
		return
	}

	// 1. Explicit duplicate check (case-insensitive)
	var exists bool
	err := dbpool.QueryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM roles WHERE LOWER(name) = LOWER($1))",
		roleName).Scan(&exists)
	if err != nil {
		http.Error(w, "Database check failed", http.StatusInternalServerError)
		return
	}
	if exists {
		http.Error(w, fmt.Sprintf("Role '%s' already exists", roleName), http.StatusConflict)
		return
	}

	// 2. Insert role
	var newID int
	err = dbpool.QueryRow(context.Background(),
		"INSERT INTO roles (name, description) VALUES ($1, $2) RETURNING id",
		roleName, input.Description).Scan(&newID)
	if err != nil {
		log.Printf("Failed to insert role: %v\n", err)
		http.Error(w, "Could not create role", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"id":   newID,
		"name": roleName,
	})
}
// DELETE /roles/{id}
func handleDeleteRole(w http.ResponseWriter, r *http.Request) {
	roleID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid role ID", http.StatusBadRequest)
		return
	}

	// 1. Prevent deleting the primary admin role
	var roleName string
	err = dbpool.QueryRow(context.Background(), "SELECT name FROM roles WHERE id = $1", roleID).Scan(&roleName)
	if err != nil {
		http.Error(w, "Role not found", http.StatusNotFound)
		return
	}
	if strings.ToLower(roleName) == "admin" || strings.ToLower(roleName) == "administrator" {
		http.Error(w, "Cannot delete system admin role", http.StatusBadRequest)
		return
	}

	// 2. Check if users are assigned to this role via role_id or role name
	var assignedCount int
	err = dbpool.QueryRow(context.Background(), 
		"SELECT COUNT(*) FROM users WHERE role_id = $1 OR role = $2", 
		roleID, roleName).Scan(&assignedCount)
	if err == nil && assignedCount > 0 {
		http.Error(w, "Cannot delete role: active users are assigned to it", http.StatusConflict)
		return
	}

	// 3. Execute delete
	_, err = dbpool.Exec(context.Background(), "DELETE FROM roles WHERE id = $1", roleID)
	if err != nil {
		log.Printf("Failed to delete role: %v\n", err)
		http.Error(w, "Cannot delete role (foreign key constraint)", http.StatusConflict)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
