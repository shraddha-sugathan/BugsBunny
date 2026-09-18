package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/crypto/bcrypt"
)

// pluginManager stays here as it's part of core server state
var pluginManager *PluginManager

func main() {
    // 1. Initialize DB (Now calls the initDB() function you placed in database.go!)
    initDB() 
    defer dbpool.Close()

	// Initialize the Plugin System
	var err error
	pluginManager, err = NewPluginManager(context.Background(), "./plugins")
	if err != nil {
		log.Fatalf("Failed to initialize plugin manager: %v", err)
	}
	
	// Generate a secure hash for "admin123" and update our dummy admin
	hash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	dbpool.Exec(context.Background(), "UPDATE users SET password_hash = $1 WHERE email = 'admin@bugsbunny.local'", string(hash))

	// 2. Setup the Go 1.22+ Router
	mux := http.NewServeMux()
	
	// Ensure the directories exist
	os.MkdirAll("./uploads", os.ModePerm)
	os.MkdirAll("./plugins", os.ModePerm)

	// Static files
	mux.Handle("GET /uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads"))))

	// Unprotected
	mux.HandleFunc("POST /login", loginHandler)

	// Issues & Transitions
    mux.HandleFunc("GET /analytics/dashboard", authMiddleware(handleGetDashboardStats))
	mux.HandleFunc("GET /issues", authMiddleware(getAllIssuesHandler))
	mux.HandleFunc("GET /issues/{id}", authMiddleware(getIssueHandler))
	mux.HandleFunc("POST /issues", authMiddleware(createIssueHandler))
	mux.HandleFunc("PUT /issues/{id}", authMiddleware(updateIssueHandler))
	mux.HandleFunc("DELETE /issues/{id}", authMiddleware(deleteIssueHandler))
	mux.HandleFunc("GET /issues/{id}/transitions", authMiddleware(handleGetIssueTransitions))
	mux.HandleFunc("POST /issues/{id}/transition", authMiddleware(handleTransitionIssue))
	mux.HandleFunc("GET /issues/{id}/comments", authMiddleware(getCommentsHandler))
	mux.HandleFunc("POST /issues/{id}/comments", authMiddleware(addCommentHandler))
	mux.HandleFunc("GET /issues/{id}/attachments", authMiddleware(getAttachmentsHandler))
	mux.HandleFunc("POST /issues/{id}/attachments", authMiddleware(uploadAttachmentHandler))

    // Projects & Versions
	mux.HandleFunc("GET /projects", authMiddleware(getProjectsHandler))
	mux.HandleFunc("POST /projects", authMiddleware(createProjectHandler))
	mux.HandleFunc("GET /projects/{id}/versions", authMiddleware(handleGetProjectVersions))
	mux.HandleFunc("POST /projects/{id}/versions", authMiddleware(handleCreateProjectVersion))
	mux.HandleFunc("DELETE /versions/{id}", authMiddleware(handleDeleteProjectVersion))
	mux.HandleFunc("GET /projects/{id}/stages", authMiddleware(handleGetStages))
	mux.HandleFunc("POST /projects/{id}/stages", authMiddleware(handleCreateStage))
	mux.HandleFunc("POST /projects/{id}/transitions", authMiddleware(handleCreateTransition))
	mux.HandleFunc("GET /projects/{id}/workflow-rules", authMiddleware(handleGetAdminWorkflowRules))

    // Users & Roles
	mux.HandleFunc("GET /profile", authMiddleware(userProfileHandler))
	mux.HandleFunc("GET /users", authMiddleware(getUsersHandler))
	mux.HandleFunc("POST /users", authMiddleware(createUserHandler))
	mux.HandleFunc("GET /roles", authMiddleware(handleGetRoles))
	mux.HandleFunc("POST /roles", authMiddleware(handleCreateRole))
	mux.HandleFunc("DELETE /roles/{id}", authMiddleware(handleDeleteRole))

    // Admin Only
	mux.HandleFunc("GET /admin/users", adminMiddleware(getAdminUsersHandler))
	mux.HandleFunc("PUT /admin/users/{id}/toggle", adminMiddleware(toggleUserStatusHandler))
	mux.HandleFunc("PUT /admin/users/{id}/password", adminMiddleware(resetPasswordHandler))
	mux.HandleFunc("GET /admin/projects", adminMiddleware(getAdminProjectsHandler))
	mux.HandleFunc("PUT /admin/projects/{id}/archive", adminMiddleware(archiveProjectHandler))

	// 3. Global CORS Wrapper
	corsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	fmt.Println("🚀 Bugsbunny Engine starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", corsMiddleware(mux)))
}