// main.go
package main

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
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
// Updated structs to include CreatedAt fields
type User struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Password  string    `json:"password,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Entry struct {
	ID           int         `json:"id"`
	UserID       int         `json:"user_id"`
	Title        string      `json:"title"`
	Text         string      `json:"text"`
	Date         string      `json:"date"`
	CreatedAt    time.Time   `json:"created_at"`
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

// Vector embedding for entries
type EntryEmbedding struct {
	ID        int       `json:"id"`
	EntryID   int       `json:"entry_id"`
	UserID    int       `json:"user_id"`
	Embedding []float64 `json:"embedding"`
	TextHash  string    `json:"text_hash"`
	CreatedAt time.Time `json:"created_at"`
}

// Similar entry for RAG context
type SimilarEntry struct {
	Entry      Entry       `json:"entry"`
	Similarity float64     `json:"similarity"`
	MoodResult *MoodResult `json:"mood_result,omitempty"`
}

// RAG Context for analysis
type RAGContext struct {
	SimilarEntries []SimilarEntry `json:"similar_entries"`
	UserPatterns   UserPatterns   `json:"user_patterns"`
}

// User mood patterns
type UserPatterns struct {
	CommonEmotions   []EmotionResult  `json:"common_emotions"`
	SentimentTrends  []SentimentTrend `json:"sentiment_trends"`
	TriggerKeywords  []string         `json:"trigger_keywords"`
	CopingStrategies []string         `json:"coping_strategies"`
}

type SentimentTrend struct {
	Period    string  `json:"period"`
	Sentiment string  `json:"sentiment"`
	Score     float64 `json:"score"`
	Count     int     `json:"count"`
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
	Migration{
		Version: 4,
		Name:    "create_embeddings_table",
		SQL: `
	CREATE TABLE IF NOT EXISTS entry_embeddings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		entry_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		embedding TEXT NOT NULL, -- JSON string of float64 array
		text_hash TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (entry_id) REFERENCES entries (id) ON DELETE CASCADE,
		FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_embeddings_user_id ON entry_embeddings(user_id);
	CREATE INDEX IF NOT EXISTS idx_embeddings_entry_id ON entry_embeddings(entry_id);`,
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

	// Get the created user with created_at timestamp
	var user User
	err = db.QueryRow("SELECT id, name, email, created_at FROM users WHERE id = ?", userID).
		Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt)
	if err != nil {
		http.Error(w, "Failed to retrieve created user", http.StatusInternalServerError)
		return
	}

	// Generate token
	token, err := generateToken(int(userID), req.Email)
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

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get user from database - now including created_at
	var user User
	var hashedPassword string
	err := db.QueryRow("SELECT id, name, email, password, created_at FROM users WHERE email = ?", req.Email).
		Scan(&user.ID, &user.Name, &user.Email, &hashedPassword, &user.CreatedAt)
	if err != nil {
		http.Error(w, "No such user found, Please sign up!", http.StatusUnauthorized)
		return
	}

	// Check password
	if !checkPassword(req.Password, hashedPassword) {
		http.Error(w, "Invalid password", http.StatusUnauthorized)
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

	// Updated query to include created_at
	rows, err := db.Query("SELECT id, title, text, date, created_at FROM entries WHERE user_id = ? ORDER BY created_at DESC", userID)
	if err != nil {
		http.Error(w, "Failed to fetch entries", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var entry Entry
		// Updated scan to include created_at
		err := rows.Scan(&entry.ID, &entry.Title, &entry.Text, &entry.Date, &entry.CreatedAt)
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

	// Get the created entry with created_at timestamp
	err = db.QueryRow("SELECT id, title, text, date, created_at FROM entries WHERE id = ?", entryID).
		Scan(&entry.ID, &entry.Title, &entry.Text, &entry.Date, &entry.CreatedAt)
	if err != nil {
		http.Error(w, "Failed to retrieve created entry", http.StatusInternalServerError)
		return
	}
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

// Text preprocessing for better embeddings
func preprocessText(text string) string {
	// Convert to lowercase
	text = strings.ToLower(text)

	// Remove extra whitespace
	text = strings.TrimSpace(text)

	// Simple tokenization and cleaning
	words := strings.Fields(text)
	var cleanWords []string

	for _, word := range words {
		// Remove punctuation
		word = strings.Trim(word, ".,!?;:\"'()[]{}...")
		if len(word) > 2 { // Keep words longer than 2 characters
			cleanWords = append(cleanWords, word)
		}
	}

	return strings.Join(cleanWords, " ")
}

// Generate text hash for duplicate detection
func generateTextHash(text string) string {
	hash := sha256.Sum256([]byte(preprocessText(text)))
	return hex.EncodeToString(hash[:])
}

// Simple TF-IDF based embedding (fallback if Hugging Face fails)
func generateSimpleEmbedding(text string) []float64 {
	text = preprocessText(text)
	words := strings.Fields(text)

	// Create a simple word frequency vector
	wordFreq := make(map[string]int)
	for _, word := range words {
		wordFreq[word]++
	}

	// Convert to fixed-size vector (384 dimensions to match sentence-transformers)
	embedding := make([]float64, 384)

	// Simple hash-based positioning
	for word, freq := range wordFreq {
		hash := sha256.Sum256([]byte(word))
		for i := 0; i < 384; i += 32 {
			idx := int(hash[i%32]) % 384
			embedding[idx] += float64(freq) * 0.1
		}
	}

	// Normalize the vector
	var norm float64
	for _, val := range embedding {
		norm += val * val
	}
	norm = math.Sqrt(norm)

	if norm > 0 {
		for i := range embedding {
			embedding[i] /= norm
		}
	}

	return embedding
}

// Generate embedding using Hugging Face sentence-transformers
func generateEmbedding(text string) ([]float64, error) {
	model := "BAAI/bge-small-en-v1.5"

	response, err := callHuggingFaceAPI(model, text)
	if err != nil {
		// Fallback to simple embedding
		log.Printf("Hugging Face embedding failed, using fallback: %v", err)
		return generateSimpleEmbedding(text), nil
	}

	// Parse embedding response
	var embedding []float64
	if err := json.Unmarshal(response, &embedding); err != nil {
		// Try nested array format
		var nestedEmbedding [][]float64
		if err := json.Unmarshal(response, &nestedEmbedding); err == nil && len(nestedEmbedding) > 0 {
			embedding = nestedEmbedding[0]
		} else {
			log.Printf("Failed to parse embedding response, using fallback: %v", err)
			return generateSimpleEmbedding(text), nil
		}
	}

	return embedding, nil
}

// Calculate cosine similarity between two vectors
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float64

	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// Save entry embedding to database
func saveEntryEmbedding(entryID, userID int, text string, embedding []float64) error {
	textHash := generateTextHash(text)
	embeddingJSON, err := json.Marshal(embedding)
	if err != nil {
		return err
	}

	// Check if embedding already exists
	var existingID int
	err = db.QueryRow("SELECT id FROM entry_embeddings WHERE entry_id = ?", entryID).Scan(&existingID)

	if err == sql.ErrNoRows {
		// Insert new embedding
		_, err = db.Exec(`
			INSERT INTO entry_embeddings (entry_id, user_id, embedding, text_hash)
			VALUES (?, ?, ?, ?)`,
			entryID, userID, string(embeddingJSON), textHash)
	} else if err == nil {
		// Update existing embedding
		_, err = db.Exec(`
			UPDATE entry_embeddings SET embedding = ?, text_hash = ?
			WHERE id = ?`,
			string(embeddingJSON), textHash, existingID)
	}

	return err
}

// Find similar entries using vector similarity
func findSimilarEntries(userID int, queryEmbedding []float64, limit int) ([]SimilarEntry, error) {
	rows, err := db.Query(`
		SELECT ee.entry_id, ee.embedding, e.title, e.text, e.date, e.created_at
		FROM entry_embeddings ee
		JOIN entries e ON ee.entry_id = e.id
		WHERE ee.user_id = ?
		ORDER BY e.created_at DESC`,
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candidates []SimilarEntry

	for rows.Next() {
		var entryID int
		var embeddingJSON string
		var entry Entry

		err := rows.Scan(&entryID, &embeddingJSON, &entry.Title, &entry.Text, &entry.Date, &entry.CreatedAt)
		if err != nil {
			continue
		}

		// Parse embedding
		var embedding []float64
		if err := json.Unmarshal([]byte(embeddingJSON), &embedding); err != nil {
			continue
		}

		// Calculate similarity
		similarity := cosineSimilarity(queryEmbedding, embedding)

		entry.ID = entryID
		entry.UserID = userID

		// Get mood analysis if available
		moodAnalysis, _ := getMoodAnalysis(entryID)

		candidates = append(candidates, SimilarEntry{
			Entry:      entry,
			Similarity: similarity,
			MoodResult: moodAnalysis,
		})
	}

	// Sort by similarity (descending)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Similarity > candidates[j].Similarity
	})

	// Return top results
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}

	return candidates, nil
}

// Analyze user patterns from historical data
func analyzeUserPatterns(userID int) (*UserPatterns, error) {
	// Get user's mood analysis history
	rows, err := db.Query(`
		SELECT ma.overall_sentiment, ma.sentiment_score, ma.emotions, ma.analyzed_at
		FROM mood_analysis ma
		JOIN entries e ON ma.entry_id = e.id
		WHERE e.user_id = ?
		ORDER BY ma.analyzed_at DESC
		LIMIT 50`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var patterns UserPatterns
	emotionFreq := make(map[string]int)
	sentimentCounts := make(map[string]int)
	var totalScore float64
	var count int

	for rows.Next() {
		var sentiment string
		var score float64
		var emotionsJSON string
		var analyzedAt time.Time

		if err := rows.Scan(&sentiment, &score, &emotionsJSON, &analyzedAt); err != nil {
			continue
		}

		// Count sentiments
		sentimentCounts[sentiment]++
		totalScore += score
		count++

		// Parse emotions
		var emotions []EmotionResult
		if err := json.Unmarshal([]byte(emotionsJSON), &emotions); err == nil {
			for _, emotion := range emotions {
				if emotion.Score > 0.3 { // Only count significant emotions
					emotionFreq[emotion.Label]++
				}
			}
		}
	}

	// Build common emotions
	type emotionCount struct {
		emotion string
		count   int
	}
	var emotionCounts []emotionCount
	for emotion, count := range emotionFreq {
		emotionCounts = append(emotionCounts, emotionCount{emotion, count})
	}
	sort.Slice(emotionCounts, func(i, j int) bool {
		return emotionCounts[i].count > emotionCounts[j].count
	})

	for i, ec := range emotionCounts {
		if i >= 5 { // Top 5 emotions
			break
		}
		patterns.CommonEmotions = append(patterns.CommonEmotions, EmotionResult{
			Label: ec.emotion,
			Score: float64(ec.count) / float64(count),
		})
	}

	// Build coping strategies based on patterns
	patterns.CopingStrategies = generateCopingStrategies(patterns.CommonEmotions)

	return &patterns, nil
}

// Generate personalized coping strategies
func generateCopingStrategies(commonEmotions []EmotionResult) []string {
	strategies := []string{}

	for _, emotion := range commonEmotions {
		switch strings.ToLower(emotion.Label) {
		case "anxiety", "fear":
			strategies = append(strategies, "Practice deep breathing exercises when feeling anxious")
		case "sadness":
			strategies = append(strategies, "Engage in activities that bring you joy, like listening to music")
		case "anger":
			strategies = append(strategies, "Try physical exercise or journaling to release tension")
		case "joy", "happiness":
			strategies = append(strategies, "Continue doing activities that bring you happiness")
		}
	}

	if len(strategies) == 0 {
		strategies = append(strategies, "Practice mindfulness and self-reflection through journaling")
	}

	return strategies
}

// Enhanced mood analysis with RAG context
func performRAGMoodAnalysis(userID int, text string) (*MoodResult, error) {
	// Generate embedding for current text
	embedding, err := generateEmbedding(text)
	if err != nil {
		log.Printf("Failed to generate embedding: %v", err)
		// Fallback to original analysis
		return performMoodAnalysis(text)
	}

	// Find similar entries
	similarEntries, err := findSimilarEntries(userID, embedding, 3)
	if err != nil {
		log.Printf("Failed to find similar entries: %v", err)
		// Fallback to original analysis
		return performMoodAnalysis(text)
	}

	// Analyze user patterns
	patterns, err := analyzeUserPatterns(userID)
	if err != nil {
		log.Printf("Failed to analyze user patterns: %v", err)
		patterns = &UserPatterns{}
	}

	// Perform basic sentiment and emotion analysis
	sentiment, score, err := analyzeSentiment(text)
	if err != nil {
		log.Printf("Sentiment analysis failed: %v", err)
		sentiment = "neutral"
		score = 0
	}

	emotions, err := analyzeEmotions(text)
	if err != nil {
		log.Printf("Emotion analysis failed: %v", err)
		emotions = []EmotionResult{}
	}

	// Generate enhanced summary with RAG context
	summary := generateRAGMoodSummary(sentiment, emotions, similarEntries, patterns)

	// Generate personalized suggestions
	suggestions := generateRAGSuggestions(text, similarEntries, patterns)

	return &MoodResult{
		OverallSentiment: sentiment,
		SentimentScore:   score,
		Emotions:         emotions,
		Summary:          summary,
		Suggestions:      suggestions,
		AnalyzedAt:       time.Now(),
	}, nil
}

// Generate enhanced mood summary with RAG context
func generateRAGMoodSummary(sentiment string, emotions []EmotionResult, similarEntries []SimilarEntry, patterns *UserPatterns) string {
	var summary strings.Builder

	summary.WriteString(fmt.Sprintf("Overall sentiment: %s. ", strings.Title(sentiment)))

	if len(emotions) > 0 {
		var topEmotion EmotionResult
		for _, emotion := range emotions {
			if emotion.Score > topEmotion.Score {
				topEmotion = emotion
			}
		}

		if topEmotion.Score > 0.3 {
			summary.WriteString(fmt.Sprintf("Primary emotion: %s (%.1f%% confidence). ",
				strings.Title(topEmotion.Label), topEmotion.Score*100))
		}
	}

	// Add pattern-based insights
	if len(patterns.CommonEmotions) > 0 {
		for _, emotion := range emotions {
			for _, commonEmotion := range patterns.CommonEmotions {
				if strings.EqualFold(emotion.Label, commonEmotion.Label) {
					summary.WriteString(fmt.Sprintf("This aligns with your typical %s patterns. ",
						emotion.Label))
					break
				}
			}
		}
	}

	// Add context from similar entries
	if len(similarEntries) > 0 && similarEntries[0].Similarity > 0.7 {
		summary.WriteString("This entry is similar to previous experiences you've written about. ")
	}

	return summary.String()
}

// Generate personalized suggestions using RAG
func generateRAGSuggestions(text string, similarEntries []SimilarEntry, patterns *UserPatterns) string {
	// 1. Try generating a fresh AI suggestion
	if suggestion, err := generateAISuggestions(text); err == nil && suggestion != "" {
		log.Printf("Generated fresh RAG suggestion")
		return suggestion
	} else {
		log.Printf("Failed to generate AI suggestion: %v", err)
	}

	// 2. Fallback: Check for any useful past suggestion in similar entries
	for _, similar := range similarEntries {
		if similar.Similarity > 0.6 && similar.MoodResult != nil && similar.MoodResult.Suggestions != "" {
			return fmt.Sprintf("Previously, you found this helpful: %s", similar.MoodResult.Suggestions)
		}
	}

	// 3. Fallback: Use first available coping strategy
	if len(patterns.CopingStrategies) > 0 {
		return patterns.CopingStrategies[0]
	}

	// 4. Final fallback
	return generateContextAwareSuggestion(text, similarEntries)
}

func generateContextAwareSuggestion(text string, similarEntries []SimilarEntry) string {
	lowerText := strings.ToLower(text)

	// Check for recurring themes in similar entries
	if len(similarEntries) > 0 {
		commonThemes := extractCommonThemes(similarEntries)
		if len(commonThemes) > 0 {
			return fmt.Sprintf("I notice this is a recurring theme for you. Consider focusing on %s as a way to address these feelings.", commonThemes[0])
		}
	}

	// Enhanced keyword-based suggestions
	if strings.Contains(lowerText, "overwhelmed") || strings.Contains(lowerText, "too much") {
		return "Break down your tasks into smaller, manageable steps. Focus on completing just one thing at a time."
	}

	if strings.Contains(lowerText, "grateful") || strings.Contains(lowerText, "thankful") {
		return "Continue cultivating gratitude! Consider keeping a daily gratitude practice to maintain this positive mindset."
	}

	// Default fallback
	return generateFallbackSuggestion(text)
}

func extractCommonThemes(entries []SimilarEntry) []string {
	// Simple theme extraction based on common words
	wordCount := make(map[string]int)

	for _, entry := range entries {
		words := strings.Fields(strings.ToLower(entry.Entry.Text))
		for _, word := range words {
			if len(word) > 4 && !isCommonWord(word) { // Skip short and common words
				wordCount[word]++
			}
		}
	}

	// Find most common meaningful words
	var themes []string
	for word, count := range wordCount {
		if count >= 2 { // Appears in at least 2 entries
			themes = append(themes, word)
		}
	}

	return themes
}

func isCommonWord(word string) bool {
	commonWords := map[string]bool{
		"that": true, "this": true, "with": true, "have": true, "will": true,
		"been": true, "were": true, "said": true, "each": true, "which": true,
		"their": true, "time": true, "would": true, "there": true, "could": true,
	}
	return commonWords[word]
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

	// Get the updated entry with created_at timestamp
	err = db.QueryRow("SELECT id, title, text, date, created_at FROM entries WHERE id = ?", entryID).
		Scan(&entry.ID, &entry.Title, &entry.Text, &entry.Date, &entry.CreatedAt)
	if err != nil {
		http.Error(w, "Failed to retrieve updated entry", http.StatusInternalServerError)
		return
	}
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

// Get current user's profile
func getUserProfileHandler(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))

	var user User
	err := db.QueryRow("SELECT id, name, email, created_at FROM users WHERE id = ?", userID).
		Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Do not expose password
	user.Password = ""

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// Update username and/or password
func updateUserProfileHandler(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))

	type UpdateRequest struct {
		Name            string `json:"name"`
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	updated := false

	// Update name
	if strings.TrimSpace(req.Name) != "" {
		_, err := db.Exec("UPDATE users SET name = ? WHERE id = ?", req.Name, userID)
		if err != nil {
			http.Error(w, "Failed to update name", http.StatusInternalServerError)
			return
		}
		updated = true
	}

	// Update password
	if strings.TrimSpace(req.NewPassword) != "" {
		if strings.TrimSpace(req.CurrentPassword) == "" {
			http.Error(w, "Current password is required to change password", http.StatusBadRequest)
			return
		}

		// Fetch hashed password from DB
		var storedHashed string
		err := db.QueryRow("SELECT password FROM users WHERE id = ?", userID).Scan(&storedHashed)
		if err != nil {
			http.Error(w, "Failed to verify current password", http.StatusInternalServerError)
			return
		}

		// Verify current password
		if !checkPassword(req.CurrentPassword, storedHashed) {
			http.Error(w, "Current password is incorrect", http.StatusUnauthorized)
			return
		}

		// Hash new password
		hashedPassword, err := hashPassword(req.NewPassword)
		if err != nil {
			http.Error(w, "Failed to hash new password", http.StatusInternalServerError)
			return
		}

		_, err = db.Exec("UPDATE users SET password = ? WHERE id = ?", hashedPassword, userID)
		if err != nil {
			http.Error(w, "Failed to update password", http.StatusInternalServerError)
			return
		}
		updated = true
	}

	if !updated {
		http.Error(w, "No valid changes provided", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Update createEntryHandler to use RAG analysis
func createEntryHandlerWithRAG(w http.ResponseWriter, r *http.Request) {
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
		entry.Date = time.Now().Format("1/2/2006")
	}

	// Insert entry
	result, err := db.Exec("INSERT INTO entries (user_id, title, text, date) VALUES (?, ?, ?, ?)",
		userID, entry.Title, entry.Text, entry.Date)
	if err != nil {
		http.Error(w, "Failed to create entry", http.StatusInternalServerError)
		return
	}

	entryID, _ := result.LastInsertId()

	// Get the created entry with created_at timestamp
	err = db.QueryRow("SELECT id, title, text, date, created_at FROM entries WHERE id = ?", entryID).
		Scan(&entry.ID, &entry.Title, &entry.Text, &entry.Date, &entry.CreatedAt)
	if err != nil {
		http.Error(w, "Failed to retrieve created entry", http.StatusInternalServerError)
		return
	}
	entry.UserID = userID

	// Perform RAG-enhanced mood analysis in background
	go func() {
		combinedText := entry.Title + " " + entry.Text

		// Generate and save embedding
		if embedding, err := generateEmbedding(combinedText); err == nil {
			if err := saveEntryEmbedding(int(entryID), userID, combinedText, embedding); err != nil {
				log.Printf("Failed to save embedding for entry %d: %v", entryID, err)
			}
		}

		// Perform RAG-enhanced mood analysis
		if moodResult, err := performRAGMoodAnalysis(userID, combinedText); err == nil {
			if err := saveMoodAnalysis(int(entryID), moodResult); err != nil {
				log.Printf("Failed to save mood analysis for entry %d: %v", entryID, err)
			} else {
				log.Printf("RAG mood analysis completed for entry %d", entryID)
			}
		} else {
			log.Printf("Failed to perform RAG mood analysis for entry %d: %v", entryID, err)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(entry)
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
	// r.HandleFunc("/api/entries", authenticateToken(createEntryHandler)).Methods("POST")
	r.HandleFunc("/api/entries", authenticateToken(createEntryHandlerWithRAG)).Methods("POST")
	r.HandleFunc("/api/entries/{id}", authenticateToken(updateEntryHandler)).Methods("PUT")
	r.HandleFunc("/api/entries/{id}", authenticateToken(deleteEntryHandler)).Methods("DELETE")
	r.HandleFunc("/api/entries/{id}/mood", authenticateToken(getMoodAnalysisHandler)).Methods("GET")

	// User profile routes
	r.HandleFunc("/api/user/profile", authenticateToken(getUserProfileHandler)).Methods("GET")
	r.HandleFunc("/api/user/profile", authenticateToken(updateUserProfileHandler)).Methods("PUT")
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
