package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
)

type DashboardStats struct {
	TotalProjects   int `json:"total_projects"`
	MyActiveIssues  int `json:"my_active_issues"`
	UnassignedBugs  int `json:"unassigned_bugs"`
	RecentlyUpdated int `json:"recently_updated"`
}

func handleGetDashboardStats(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(claimsKey).(*Claims)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var stats DashboardStats

	// 1. Total Active Projects
	err := dbpool.QueryRow(context.Background(), 
		"SELECT COUNT(*) FROM projects WHERE is_archived = false").Scan(&stats.TotalProjects)
	if err != nil {
		log.Printf("Error fetching project count: %v", err)
	}

	// 2. My Active Issues (Assigned to the logged-in user, not closed)
	err = dbpool.QueryRow(context.Background(), 
		"SELECT COUNT(*) FROM issues WHERE assignee_id = $1 AND status != 'Closed' AND status != 'Resolved'", 
		claims.UserID).Scan(&stats.MyActiveIssues)
	if err != nil {
		log.Printf("Error fetching user issues: %v", err)
	}

	// 3. Unassigned Open Issues
	err = dbpool.QueryRow(context.Background(), 
		"SELECT COUNT(*) FROM issues WHERE assignee_id IS NULL AND status != 'Closed' AND status != 'Resolved'").Scan(&stats.UnassignedBugs)
	if err != nil {
		log.Printf("Error fetching unassigned issues: %v", err)
	}

	// 4. Recently Updated Issues (Last 7 days)
	err = dbpool.QueryRow(context.Background(), 
		"SELECT COUNT(*) FROM issues WHERE updated_at >= NOW() - INTERVAL '7 days'").Scan(&stats.RecentlyUpdated)
	if err != nil {
		log.Printf("Error fetching recent issues: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}