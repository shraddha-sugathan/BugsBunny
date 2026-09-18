package main

import (
	"context"	
    "encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func loginHandler(w http.ResponseWriter, r *http.Request) {
    var req LoginRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }

    var user User
	// Fetch the user from the database
    err := dbpool.QueryRow(context.Background(), 
    "SELECT id, password_hash, role, is_active FROM users WHERE email = $1", req.Email).
    Scan(&user.ID, &user.PasswordHash, &user.Role, &user.IsActive)

    if err != nil {
        log.Printf("[DB ERROR] Failed to query/scan user: %v\n", err)
        http.Error(w, "Invalid credentials", http.StatusUnauthorized)
        return
    }

    log.Printf("[LOGIN DEBUG] Found user: ID=%d, Role=%s, HashLen=%d\n", user.ID, user.Role, len(user.PasswordHash))

	if err == nil && !user.IsActive {
        log.Printf("[LOGIN DEBUG] User inactive\n")
	    http.Error(w, "Account deactivated", http.StatusForbidden)
	    return
	}

    // Compare the submitted password with the database hash
    if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
        log.Printf("[LOGIN DEBUG] DB error or user not found: %v\n", err)
        http.Error(w, "Invalid credentials", http.StatusUnauthorized)
        return
    }


    if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
    log.Printf("[LOGIN DEBUG] Bcrypt match failed: %v\n", err)
    http.Error(w, "Invalid credentials", http.StatusUnauthorized)
    return
    }
    // Create the JWT Token valid for 24 hours
    expirationTime := time.Now().Add(24 * time.Hour)
    claims := &Claims{
        UserID: user.ID,
        Role:   user.Role,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(expirationTime),
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    tokenString, err := token.SignedString(jwtSecret)
    if err != nil {
        http.Error(w, "Error generating token", http.StatusInternalServerError)
        return
    }

    // Send the token back to Vue!
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{
        "token": tokenString,
        "role":  user.Role,
        "email": req.Email,
    })
}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
    // 1. Create a temporary struct to catch the exact JSON sent from Vue
    var req struct {
        Email    string `json:"email"`
        Password string `json:"password"`
        Role     string `json:"role"`
    }

    // 2. Decode the stream exactly once
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid data", http.StatusBadRequest)
        return
    }

    // 3. Hash the raw password
    hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)

    // 4. Insert into the database
    var newUserID int
    err := dbpool.QueryRow(context.Background(),
        "INSERT INTO users (email, password_hash, role) VALUES ($1, $2, $3) RETURNING id",
        req.Email, string(hash), req.Role).Scan(&newUserID)
        
    if err != nil {
        fmt.Println("❌ Error creating user:", err)
        http.Error(w, "Failed to create user", http.StatusInternalServerError)
        return
    }
    
    w.WriteHeader(http.StatusCreated)
}

func getUsersHandler(w http.ResponseWriter, r *http.Request) {
    rows, err := dbpool.Query(context.Background(), "SELECT id, email FROM users ORDER BY email")
    if err != nil {
        http.Error(w, "Failed to fetch users", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var users []UserStub
    for rows.Next() {
        var u UserStub
        rows.Scan(&u.ID, &u.Email)
        users = append(users, u)
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(users)
}

func userProfileHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(int)
	var profile UserProfile

	// 1. Get user email
	dbpool.QueryRow(context.Background(), "SELECT email FROM users WHERE id = $1", userID).Scan(&profile.Email)

	// 2. Calculate Metrics (Now using assignee_id instead of reporter_id)
	dbpool.QueryRow(context.Background(), "SELECT COUNT(*) FROM issues WHERE assignee_id = $1", userID).Scan(&profile.TotalIssues)
	dbpool.QueryRow(context.Background(), "SELECT COUNT(*) FROM issues WHERE assignee_id = $1 AND status = 'resolved'", userID).Scan(&profile.ResolvedCount)

	// 3. Fetch the user's assigned issues
	rows, _ := dbpool.Query(context.Background(), "SELECT issue_key, title, status FROM issues WHERE assignee_id = $1 ORDER BY issue_key DESC", userID)
	defer rows.Close()
	
	for rows.Next() {
		var i IssueStub
		rows.Scan(&i.IssueKey, &i.Title, &i.Status)
		profile.Issues = append(profile.Issues, i)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

func toggleUserStatusHandler(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    _, err := dbpool.Exec(context.Background(), "UPDATE users SET is_active = NOT is_active WHERE id = $1", id)
    if err != nil {
        http.Error(w, "Failed to update user", http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusOK)
}

func resetPasswordHandler(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    var req struct { Password string `json:"password"` }
    json.NewDecoder(r.Body).Decode(&req)

    hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
    _, err := dbpool.Exec(context.Background(), "UPDATE users SET password_hash = $1 WHERE id = $2", string(hash), id)
    
    if err != nil {
        http.Error(w, "Failed to reset password", http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusOK)
}


func getAdminUsersHandler(w http.ResponseWriter, r *http.Request) {
    rows, err := dbpool.Query(context.Background(), "SELECT id, email, role, is_active FROM users ORDER BY id ASC")
    if err != nil {
        http.Error(w, "Failed to fetch users", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var users []User
    for rows.Next() {
        var u User
        if err := rows.Scan(&u.ID, &u.Email, &u.Role, &u.IsActive); err == nil {
            users = append(users, u)
        }
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(users)
}

