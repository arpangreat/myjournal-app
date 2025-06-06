// main.go
package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/cors"
	"golang.org/x/crypto/bcrypt"
)

// Database instance
var db *sql.DB

// JWT secret key (in production, use environment variable)
var jwtSecret = []byte("your-secret-key-change-this-in-production")

// Hugging Face API configuration
// var huggingFaceAPIKey = os.Getenv("HUGGINGFACE_API_KEY") // Optional: Set for higher rate limits
var huggingFaceAPIKey string // Declare the variable

func init() {
	// Load .env file first
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Then initialize the variable
	huggingFaceAPIKey = os.Getenv("HUGGINGFACE_API_KEY") // Optional: Set for higher rate limits
}

const huggingFaceAPIURL = "https://router.huggingface.co/hf-inference/models/"

// Migration struct
type Migration struct {
	Version int
	Name    string
	SQL     string
}

// User struct
type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password,omitempty"`
}

// Entry struct
type Entry struct {
	ID           int         `json:"id"`
	UserID       int         `json:"user_id"`
	Title        string      `json:"title"`
	Text         string      `json:"text"`
	Date         string      `json:"date"`
	MoodAnalysis *MoodResult `json:"mood_analysis,omitempty"`
}

// Mood analysis structs
type MoodResult struct {
	OverallSentiment string          `json:"overall_sentiment"`
	SentimentScore   float64         `json:"sentiment_score"`
	Emotions         []EmotionResult `json:"emotions"`
	Summary          string          `json:"summary"`
	Suggestions      string          `json:"suggestions"`
	AnalyzedAt       time.Time       `json:"analyzed_at"`
}

type EmotionResult struct {
	Label string  `json:"label"`
	Score float64 `json:"score"`
}

// Hugging Face API response structures
type HuggingFaceSentimentResponse []struct {
	Label string  `json:"label"`
	Score float64 `json:"score"`
}

type HuggingFaceEmotionResponse []struct {
	Label string  `json:"label"`
	Score float64 `json:"score"`
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

// Database migrations
var migrations = []Migration{
	{
		Version: 1,
		Name:    "create_users_table",
		SQL: `
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			email TEXT UNIQUE NOT NULL,
			password TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
	},
	{
		Version: 2,
		Name:    "create_entries_table",
		SQL: `
		CREATE TABLE IF NOT EXISTS entries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			text TEXT NOT NULL,
			date TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users (id)
		);`,
	},
	{
		Version: 3,
		Name:    "create_mood_analysis_table",
		SQL: `
		CREATE TABLE IF NOT EXISTS mood_analysis (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			entry_id INTEGER NOT NULL,
			overall_sentiment TEXT NOT NULL,
			sentiment_score REAL NOT NULL,
			emotions TEXT NOT NULL, -- JSON string
			summary TEXT,
			suggestions TEXT,
			analyzed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (entry_id) REFERENCES entries (id) ON DELETE CASCADE
		);`,
	},
}

// Hugging Face API functions
func callHuggingFaceAPI(modelName, text string) ([]byte, error) {
	url := huggingFaceAPIURL + modelName

	payload := map[string]interface{}{
		"inputs": text,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if huggingFaceAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+huggingFaceAPIKey)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

func analyzeSentiment(text string) (string, float64, error) {
	response, err := callHuggingFaceAPI("tabularisai/multilingual-sentiment-analysis", text)
	if err != nil {
		return "", 0, err
	}

	// Log the raw response for debugging
	log.Printf("Raw sentiment API response: %s", string(response))

	// Try to unmarshal as nested array first (Hugging Face returns [[{...}]])
	var nestedResponse [][]struct {
		Label string  `json:"label"`
		Score float64 `json:"score"`
	}

	if err := json.Unmarshal(response, &nestedResponse); err == nil && len(nestedResponse) > 0 {
		// Use the first inner array
		sentimentResponse := nestedResponse[0]
		if len(sentimentResponse) == 0 {
			return "neutral", 0, nil
		}

		// Find the sentiment with highest score
		var bestSentiment string
		var bestScore float64
		for _, result := range sentimentResponse {
			if result.Score > bestScore {
				bestScore = result.Score
				bestSentiment = result.Label
			}
		}

		// Convert to readable format and score
		switch strings.ToLower(bestSentiment) {
		case "negative", "very negative":
			return "negative", -bestScore, nil
		case "neutral":
			return "neutral", 0, nil
		case "positive", "very positive":
			return "positive", bestScore, nil
		default:
			return bestSentiment, bestScore, nil
		}
	}

	// Fallback: Try to unmarshal as single array
	var sentimentResponse []struct {
		Label string  `json:"label"`
		Score float64 `json:"score"`
	}

	if err := json.Unmarshal(response, &sentimentResponse); err == nil {
		if len(sentimentResponse) == 0 {
			return "neutral", 0, nil
		}

		// Find the sentiment with highest score
		var bestSentiment string
		var bestScore float64
		for _, result := range sentimentResponse {
			if result.Score > bestScore {
				bestScore = result.Score
				bestSentiment = result.Label
			}
		}

		// Convert to readable format and score
		switch strings.ToLower(bestSentiment) {
		case "negative", "very negative":
			return "negative", -bestScore, nil
		case "neutral":
			return "neutral", 0, nil
		case "positive", "very positive":
			return "positive", bestScore, nil
		default:
			return bestSentiment, bestScore, nil
		}
	}

	// If both formats fail, try single object format
	var singleResponse struct {
		Label string  `json:"label"`
		Score float64 `json:"score"`
	}
	if err := json.Unmarshal(response, &singleResponse); err == nil {
		switch strings.ToLower(singleResponse.Label) {
		case "negative", "very negative":
			return "negative", -singleResponse.Score, nil
		case "neutral":
			return "neutral", 0, nil
		case "positive", "very positive":
			return "positive", singleResponse.Score, nil
		default:
			return singleResponse.Label, singleResponse.Score, nil
		}
	}

	log.Printf("Failed to parse sentiment response in any expected format")
	log.Printf("Raw response: %s", string(response))
	return "neutral", 0, fmt.Errorf("failed to parse sentiment response")
}

func analyzeEmotions(text string) ([]EmotionResult, error) {
	// Use j-hartmann/emotion-english-distilroberta-base model
	response, err := callHuggingFaceAPI("j-hartmann/emotion-english-distilroberta-base", text)
	if err != nil {
		return nil, err
	}

	// Log the raw response for debugging
	log.Printf("Raw emotion API response: %s", string(response))

	// Try to unmarshal as nested array first (Hugging Face returns [[{...}]])
	var nestedResponse [][]struct {
		Label string  `json:"label"`
		Score float64 `json:"score"`
	}

	if err := json.Unmarshal(response, &nestedResponse); err == nil && len(nestedResponse) > 0 {
		// Use the first inner array
		emotionResponse := nestedResponse[0]
		var emotions []EmotionResult
		for _, emotion := range emotionResponse {
			emotions = append(emotions, EmotionResult{
				Label: emotion.Label,
				Score: emotion.Score,
			})
		}
		return emotions, nil
	}

	// Fallback: Try to unmarshal as single array
	var emotionResponse []struct {
		Label string  `json:"label"`
		Score float64 `json:"score"`
	}

	if err := json.Unmarshal(response, &emotionResponse); err == nil {
		var emotions []EmotionResult
		for _, emotion := range emotionResponse {
			emotions = append(emotions, EmotionResult{
				Label: emotion.Label,
				Score: emotion.Score,
			})
		}
		return emotions, nil
	}

	// If array format fails, try single object format
	var singleResponse struct {
		Label string  `json:"label"`
		Score float64 `json:"score"`
	}
	if err := json.Unmarshal(response, &singleResponse); err == nil {
		return []EmotionResult{{
			Label: singleResponse.Label,
			Score: singleResponse.Score,
		}}, nil
	}

	log.Printf("Failed to parse emotion response in any expected format")
	log.Printf("Raw response: %s", string(response))
	return nil, fmt.Errorf("failed to parse emotion response")
}

func performMoodAnalysis(text string) (*MoodResult, error) {
	// Analyze sentiment
	sentiment, score, err := analyzeSentiment(text)
	if err != nil {
		log.Printf("Sentiment analysis failed: %v", err)
		sentiment = "neutral"
		score = 0
	}

	// Analyze emotions
	emotions, err := analyzeEmotions(text)
	if err != nil {
		log.Printf("Emotion analysis failed: %v", err)
		emotions = []EmotionResult{}
	}

	// Generate summary
	summary := generateMoodSummary(sentiment, emotions)

	// Generate AI suggestions
	suggestions, err := generateAISuggestions(text)
	if err != nil {
		log.Printf("AI suggestion generation failed: %v", err)
		suggestions = "Try reflecting more on your feelings or write another journal entry."
	}

	return &MoodResult{
		OverallSentiment: sentiment,
		SentimentScore:   score,
		Emotions:         emotions,
		Summary:          summary,
		Suggestions:      suggestions,
		AnalyzedAt:       time.Now(),
	}, nil
}

func generateMoodSummary(sentiment string, emotions []EmotionResult) string {
	var summary strings.Builder

	summary.WriteString(fmt.Sprintf("Overall sentiment: %s. ", strings.Title(sentiment)))

	if len(emotions) > 0 {
		// Find top emotion
		var topEmotion EmotionResult
		for _, emotion := range emotions {
			if emotion.Score > topEmotion.Score {
				topEmotion = emotion
			}
		}

		if topEmotion.Score > 0.3 {
			summary.WriteString(fmt.Sprintf("Primary emotion detected: %s (%.1f%% confidence).",
				strings.Title(topEmotion.Label), topEmotion.Score*100))
		}
	}

	return summary.String()
}

// mistralai/Mixtral-8x7B-Instruct-v0.1
func generateAISuggestions(text string) (string, error) {
	// Use a more reliable model for text generation
	model := "mistralai/Mixtral-8x7B-Instruct-v0.1" // Alternative: "gpt2" or "facebook/blenderbot-400M-distill"

	// Create a more focused prompt
	prompt := fmt.Sprintf("Based on this journal entry, suggest one helpful wellness activity:\n\nJournal: \"%s\"\n\nSuggestion:", text)

	payload := map[string]interface{}{
		"inputs": prompt,
		"parameters": map[string]interface{}{
			"max_length":   150,
			"temperature":  0.7,
			"do_sample":    true,
			"pad_token_id": 50256,
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", huggingFaceAPIURL+model, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	if huggingFaceAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+huggingFaceAPIKey)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("AI suggestion API error: %s", string(body))
		return generateFallbackSuggestion(text), nil // Return fallback instead of error
	}

	var result []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("Failed to decode AI response: %v", err)
		return generateFallbackSuggestion(text), nil
	}

	if len(result) > 0 {
		if generated, ok := result[0]["generated_text"].(string); ok {
			// Clean up the response
			cleaned := cleanAISuggestion(generated, prompt)
			if cleaned != "" {
				return cleaned, nil
			}
		}
	}

	return generateFallbackSuggestion(text), nil
}

// Clean up AI-generated suggestions
func cleanAISuggestion(generated, originalPrompt string) string {
	// Remove the original prompt from the response
	suggestion := strings.Replace(generated, originalPrompt, "", 1)

	// Clean up common issues
	suggestion = strings.TrimSpace(suggestion)

	// Remove any remaining prompt artifacts
	lines := strings.Split(suggestion, "\n")
	var cleanLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip lines that look like prompts or metadata
		if strings.Contains(strings.ToLower(line), "journal entry") ||
			strings.Contains(strings.ToLower(line), "suggestion:") ||
			strings.Contains(strings.ToLower(line), "based on") ||
			len(line) < 10 { // Skip very short lines
			continue
		}

		// Clean up formatting
		line = strings.Trim(line, "\"'*-")
		if line != "" {
			cleanLines = append(cleanLines, line)
		}
	}

	if len(cleanLines) > 0 {
		// Take the first meaningful line and ensure it's well-formatted
		suggestion = cleanLines[0]

		// Ensure it ends with proper punctuation
		if !strings.HasSuffix(suggestion, ".") && !strings.HasSuffix(suggestion, "!") && !strings.HasSuffix(suggestion, "?") {
			suggestion += "."
		}

		// Capitalize first letter
		if len(suggestion) > 0 {
			suggestion = strings.ToUpper(string(suggestion[0])) + suggestion[1:]
		}

		return suggestion
	}

	return ""
}

// Generate fallback suggestions based on sentiment
func generateFallbackSuggestion(text string) string {
	// Simple keyword-based suggestions as fallback
	lowerText := strings.ToLower(text)

	// Negative sentiment indicators
	if strings.Contains(lowerText, "stress") || strings.Contains(lowerText, "anxious") || strings.Contains(lowerText, "worry") {
		return "Try a 5-minute breathing exercise: breathe in for 4 counts, hold for 4, breathe out for 6. This can help calm your nervous system."
	}

	if strings.Contains(lowerText, "sad") || strings.Contains(lowerText, "down") || strings.Contains(lowerText, "depressed") {
		return "Consider taking a short walk outside or doing something creative like drawing or listening to your favorite music."
	}

	if strings.Contains(lowerText, "tired") || strings.Contains(lowerText, "exhausted") || strings.Contains(lowerText, "sleep") {
		return "Focus on getting quality rest tonight. Try creating a calming bedtime routine without screens for the last hour before sleep."
	}

	if strings.Contains(lowerText, "angry") || strings.Contains(lowerText, "frustrated") || strings.Contains(lowerText, "mad") {
		return "Try some physical activity to release tension, like stretching, going for a walk, or doing jumping jacks for 2 minutes."
	}

	if strings.Contains(lowerText, "lonely") || strings.Contains(lowerText, "alone") {
		return "Reach out to a friend or family member, even if just to say hello. Consider joining a community activity or volunteering."
	}

	// Positive sentiment
	if strings.Contains(lowerText, "happy") || strings.Contains(lowerText, "good") || strings.Contains(lowerText, "great") {
		return "Celebrate this positive moment! Consider writing down three things you're grateful for today."
	}

	// Default suggestions
	suggestions := []string{
		"Take a few minutes to practice mindfulness by focusing on your breathing and being present in the moment.",
		"Try journaling about three things you're grateful for today, no matter how small they might seem.",
		"Consider doing some light physical activity like stretching or taking a short walk to boost your mood.",
		"Reach out to someone you care about and let them know you're thinking of them.",
		"Practice self-compassion by treating yourself with the same kindness you'd show a good friend.",
	}

	// Return a random suggestion
	return suggestions[len(text)%len(suggestions)]
}

func saveMoodAnalysis(entryID int, moodResult *MoodResult) error {
	emotionsJSON, err := json.Marshal(moodResult.Emotions)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
	INSERT INTO mood_analysis (entry_id, overall_sentiment, sentiment_score, emotions, summary, suggestions)
	VALUES (?, ?, ?, ?, ?, ?)`,
		entryID, moodResult.OverallSentiment, moodResult.SentimentScore,
		string(emotionsJSON), moodResult.Summary, moodResult.Suggestions)

	return err
}

func getMoodAnalysis(entryID int) (*MoodResult, error) {
	var moodResult MoodResult
	var emotionsJSON string

	err := db.QueryRow(`
		SELECT overall_sentiment, sentiment_score, emotions, summary, suggestions, analyzed_at
		FROM mood_analysis WHERE entry_id = ?`, entryID).Scan(
		&moodResult.OverallSentiment, &moodResult.SentimentScore,
		&emotionsJSON, &moodResult.Summary, &moodResult.Suggestions, &moodResult.AnalyzedAt)

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(emotionsJSON), &moodResult.Emotions); err != nil {
		return nil, err
	}

	return &moodResult, nil
}

// Create migrations table
func createMigrationsTable() error {
	createMigrationsSQL := `
	CREATE TABLE IF NOT EXISTS migrations (
		version INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	_, err := db.Exec(createMigrationsSQL)
	return err
}

// Get current migration version
func getCurrentMigrationVersion() (int, error) {
	var version int
	err := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM migrations").Scan(&version)
	if err != nil {
		return 0, err
	}
	return version, nil
}

// Run migrations
func runMigrations() error {
	// Create migrations table first
	if err := createMigrationsTable(); err != nil {
		return fmt.Errorf("failed to create migrations table: %v", err)
	}

	// Get current version
	currentVersion, err := getCurrentMigrationVersion()
	if err != nil {
		return fmt.Errorf("failed to get current migration version: %v", err)
	}

	// Run pending migrations
	for _, migration := range migrations {
		if migration.Version > currentVersion {
			fmt.Printf("Running migration %d: %s\n", migration.Version, migration.Name)

			// Execute migration
			if _, err := db.Exec(migration.SQL); err != nil {
				return fmt.Errorf("failed to run migration %d (%s): %v", migration.Version, migration.Name, err)
			}

			// Record migration
			if _, err := db.Exec("INSERT INTO migrations (version, name) VALUES (?, ?)", migration.Version, migration.Name); err != nil {
				return fmt.Errorf("failed to record migration %d: %v", migration.Version, err)
			}

			fmt.Printf("Migration %d completed successfully\n", migration.Version)
		}
	}

	return nil
}

// Create database backup
func createBackup() error {
	// Create backups directory if it doesn't exist
	backupDir := "./backups"
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %v", err)
	}

	// Generate backup filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	backupPath := filepath.Join(backupDir, fmt.Sprintf("journal_backup_%s.db", timestamp))

	// Open source database file
	sourceFile, err := os.Open("./journal.db")
	if err != nil {
		return fmt.Errorf("failed to open source database: %v", err)
	}
	defer sourceFile.Close()

	// Create backup file
	backupFile, err := os.Create(backupPath)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %v", err)
	}
	defer backupFile.Close()

	// Copy database file
	if _, err := io.Copy(backupFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy database: %v", err)
	}

	fmt.Printf("Database backup created: %s\n", backupPath)
	return nil
}

// Cleanup old backups (keep last 10)
func cleanupOldBackups() error {
	backupDir := "./backups"

	// Read backup directory
	files, err := os.ReadDir(backupDir)
	if err != nil {
		// If directory doesn't exist, that's fine
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read backup directory: %v", err)
	}

	// Filter backup files and sort by modification time
	var backupFiles []os.FileInfo
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "journal_backup_") && strings.HasSuffix(file.Name(), ".db") {
			info, err := file.Info()
			if err != nil {
				continue
			}
			backupFiles = append(backupFiles, info)
		}
	}

	// If we have more than 10 backups, delete the oldest ones
	if len(backupFiles) > 10 {
		// Sort by modification time (oldest first)
		for i := 0; i < len(backupFiles)-1; i++ {
			for j := i + 1; j < len(backupFiles); j++ {
				if backupFiles[i].ModTime().After(backupFiles[j].ModTime()) {
					backupFiles[i], backupFiles[j] = backupFiles[j], backupFiles[i]
				}
			}
		}

		// Delete oldest backups
		filesToDelete := len(backupFiles) - 10
		for i := 0; i < filesToDelete; i++ {
			oldBackupPath := filepath.Join(backupDir, backupFiles[i].Name())
			if err := os.Remove(oldBackupPath); err != nil {
				log.Printf("Warning: failed to delete old backup %s: %v", oldBackupPath, err)
			} else {
				fmt.Printf("Deleted old backup: %s\n", backupFiles[i].Name())
			}
		}
	}

	return nil
}

// Schedule automatic backups
func scheduleBackups() {
	ticker := time.NewTicker(24 * time.Hour) // Daily backups
	go func() {
		for range ticker.C {
			if err := createBackup(); err != nil {
				log.Printf("Automatic backup failed: %v", err)
			} else {
				// Cleanup old backups after successful backup
				if err := cleanupOldBackups(); err != nil {
					log.Printf("Failed to cleanup old backups: %v", err)
				}
			}
		}
	}()
}

// Initialize database
func initDB() {
	var err error
	db, err = sql.Open("sqlite3", "./journal.db")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}

	// Run migrations
	if err := runMigrations(); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	// Create initial backup
	if err := createBackup(); err != nil {
		log.Printf("Warning: Failed to create initial backup: %v", err)
	}

	// Schedule automatic backups
	scheduleBackups()

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

		// Try to get mood analysis for this entry
		if moodAnalysis, err := getMoodAnalysis(entry.ID); err == nil {
			entry.MoodAnalysis = moodAnalysis
		}

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

	// Perform mood analysis in background
	go func() {
		combinedText := entry.Title + " " + entry.Text
		if moodResult, err := performMoodAnalysis(combinedText); err == nil {
			if err := saveMoodAnalysis(int(entryID), moodResult); err != nil {
				log.Printf("Failed to save mood analysis for entry %d: %v", entryID, err)
			} else {
				log.Printf("Mood analysis completed for entry %d", entryID)
			}
		} else {
			log.Printf("Failed to perform mood analysis for entry %d: %v", entryID, err)
		}
	}()

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

	// Re-analyze mood in background
	go func() {
		combinedText := entry.Title + " " + entry.Text
		if moodResult, err := performMoodAnalysis(combinedText); err == nil {
			// Delete old analysis and save new one
			db.Exec("DELETE FROM mood_analysis WHERE entry_id = ?", entryID)
			if err := saveMoodAnalysis(entryID, moodResult); err != nil {
				log.Printf("Failed to save updated mood analysis for entry %d: %v", entryID, err)
			} else {
				log.Printf("Mood analysis updated for entry %d", entryID)
			}
		} else {
			log.Printf("Failed to perform mood analysis for updated entry %d: %v", entryID, err)
		}
	}()

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

	// Delete entry (mood analysis will be deleted due to CASCADE)
	_, err = db.Exec("DELETE FROM entries WHERE id = ?", entryID)
	if err != nil {
		http.Error(w, "Failed to delete entry", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// New endpoint to get mood analysis for a specific entry
func getMoodAnalysisHandler(w http.ResponseWriter, r *http.Request) {
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

	// Get mood analysis
	moodAnalysis, err := getMoodAnalysis(entryID)
	if err != nil {
		http.Error(w, "Mood analysis not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(moodAnalysis)
}

func main() {
	// Initialize database
	initDB()
	defer db.Close()

	// Check if Hugging Face API key is provided
	r := mux.NewRouter()

	// Auth routes
	r.HandleFunc("/api/signup", signupHandler).Methods("POST")
	r.HandleFunc("/api/login", loginHandler).Methods("POST")

	// Protected entry routes
	r.HandleFunc("/api/entries", authenticateToken(getEntriesHandler)).Methods("GET")
	r.HandleFunc("/api/entries", authenticateToken(createEntryHandler)).Methods("POST")
	r.HandleFunc("/api/entries/{id}", authenticateToken(updateEntryHandler)).Methods("PUT")
	r.HandleFunc("/api/entries/{id}", authenticateToken(deleteEntryHandler)).Methods("DELETE")
	r.HandleFunc("/api/entries/{id}/mood", authenticateToken(getMoodAnalysisHandler)).Methods("GET")

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
