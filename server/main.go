// main.go
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/cors"
	"golang.org/x/crypto/bcrypt"
)

// Database instance
var db *sql.DB

// JWT secret key (in production, use environment variable)
var jwtSecret = []byte("your-secret-key-change-this-in-production")

// User struct
type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password,omitempty"`
}

// Entry struct
type Entry struct {
	ID     int    `json:"id"`
	UserID int    `json:"user_id"`
	Title  string `json:"title"`
	Text   string `json:"text"`
	Date   string `json:"date"`
}

// Auth request structs
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type SignupRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// JWT Claims
type Claims struct {
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// Initialize database
func initDB() {
	var err error
	db, err = sql.Open("sqlite3", "./journal.db")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}

	// Create users table
	createUsersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	// Create entries table
	createEntriesTable := `
	CREATE TABLE IF NOT EXISTS entries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		title TEXT NOT NULL,
		text TEXT NOT NULL,
		date TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users (id)
	);`

	if _, err := db.Exec(createUsersTable); err != nil {
		log.Fatal("Failed to create users table:", err)
	}

	if _, err := db.Exec(createEntriesTable); err != nil {
		log.Fatal("Failed to create entries table:", err)
	}

	fmt.Println("Database initialized successfully")
}

// Hash password
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

// Check password
func checkPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// Generate JWT token
func generateToken(userID int, email string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// Middleware to authenticate JWT
func authenticateToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		tokenString := strings.Replace(authHeader, "Bearer ", "", 1)
		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Add user ID to request context
		r.Header.Set("X-User-ID", strconv.Itoa(claims.UserID))
		next.ServeHTTP(w, r)
	}
}

// Auth handlers
func signupHandler(w http.ResponseWriter, r *http.Request) {
	var req SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Name == "" || req.Email == "" || req.Password == "" {
		http.Error(w, "All fields are required", http.StatusBadRequest)
		return
	}

	// Check if user already exists
	var existingID int
	err := db.QueryRow("SELECT id FROM users WHERE email = ?", req.Email).Scan(&existingID)
	if err == nil {
		http.Error(w, "User already exists", http.StatusConflict)
		return
	}

	// Hash password
	hashedPassword, err := hashPassword(req.Password)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	// Insert user
	result, err := db.Exec("INSERT INTO users (name, email, password) VALUES (?, ?, ?)",
		req.Name, req.Email, hashedPassword)
	if err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	userID, _ := result.LastInsertId()

	// Generate token
	token, err := generateToken(int(userID), req.Email)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	user := User{
		ID:    int(userID),
		Name:  req.Name,
		Email: req.Email,
	}

	response := AuthResponse{
		Token: token,
		User:  user,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get user from database
	var user User
	var hashedPassword string
	err := db.QueryRow("SELECT id, name, email, password FROM users WHERE email = ?", req.Email).
		Scan(&user.ID, &user.Name, &user.Email, &hashedPassword)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Check password
	if !checkPassword(req.Password, hashedPassword) {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Generate token
	token, err := generateToken(user.ID, user.Email)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	response := AuthResponse{
		Token: token,
		User:  user,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Entry handlers
func getEntriesHandler(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))

	rows, err := db.Query("SELECT id, title, text, date FROM entries WHERE user_id = ? ORDER BY created_at DESC", userID)
	if err != nil {
		http.Error(w, "Failed to fetch entries", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var entry Entry
		err := rows.Scan(&entry.ID, &entry.Title, &entry.Text, &entry.Date)
		if err != nil {
			http.Error(w, "Failed to scan entry", http.StatusInternalServerError)
			return
		}
		entry.UserID = userID
		entries = append(entries, entry)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

func createEntryHandler(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))

	var entry Entry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate input
	if entry.Title == "" || entry.Text == "" {
		http.Error(w, "Title and text are required", http.StatusBadRequest)
		return
	}

	// Set date if not provided
	if entry.Date == "" {
		entry.Date = time.Now().Format("1/2/2006") // MM/DD/YYYY format to match frontend
	}

	// Insert entry
	result, err := db.Exec("INSERT INTO entries (user_id, title, text, date) VALUES (?, ?, ?, ?)",
		userID, entry.Title, entry.Text, entry.Date)
	if err != nil {
		http.Error(w, "Failed to create entry", http.StatusInternalServerError)
		return
	}

	entryID, _ := result.LastInsertId()
	entry.ID = int(entryID)
	entry.UserID = userID

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(entry)
}

func updateEntryHandler(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	vars := mux.Vars(r)
	entryID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid entry ID", http.StatusBadRequest)
		return
	}

	var entry Entry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Check if entry belongs to user
	var ownerID int
	err = db.QueryRow("SELECT user_id FROM entries WHERE id = ?", entryID).Scan(&ownerID)
	if err != nil {
		http.Error(w, "Entry not found", http.StatusNotFound)
		return
	}
	if ownerID != userID {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Update entry
	_, err = db.Exec("UPDATE entries SET title = ?, text = ? WHERE id = ?",
		entry.Title, entry.Text, entryID)
	if err != nil {
		http.Error(w, "Failed to update entry", http.StatusInternalServerError)
		return
	}

	entry.ID = entryID
	entry.UserID = userID

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entry)
}

func deleteEntryHandler(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	vars := mux.Vars(r)
	entryID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid entry ID", http.StatusBadRequest)
		return
	}

	// Check if entry belongs to user
	var ownerID int
	err = db.QueryRow("SELECT user_id FROM entries WHERE id = ?", entryID).Scan(&ownerID)
	if err != nil {
		http.Error(w, "Entry not found", http.StatusNotFound)
		return
	}
	if ownerID != userID {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Delete entry
	_, err = db.Exec("DELETE FROM entries WHERE id = ?", entryID)
	if err != nil {
		http.Error(w, "Failed to delete entry", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func main() {
	// Initialize database
	initDB()
	defer db.Close()

	// Create router
	r := mux.NewRouter()

	// Auth routes
	r.HandleFunc("/api/signup", signupHandler).Methods("POST")
	r.HandleFunc("/api/login", loginHandler).Methods("POST")

	// Protected entry routes
	r.HandleFunc("/api/entries", authenticateToken(getEntriesHandler)).Methods("GET")
	r.HandleFunc("/api/entries", authenticateToken(createEntryHandler)).Methods("POST")
	r.HandleFunc("/api/entries/{id}", authenticateToken(updateEntryHandler)).Methods("PUT")
	r.HandleFunc("/api/entries/{id}", authenticateToken(deleteEntryHandler)).Methods("DELETE")

	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:5173"}, // Add your frontend URLs
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(r)

	fmt.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
