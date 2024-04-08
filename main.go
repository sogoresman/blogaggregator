package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	DB *sql.DB // Change the type to *sql.DB
}

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}

	// Get the database URL from environment variable
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fmt.Println("DATABASE_URL not found in environment variables")
		return
	}

	// Open a connection to the database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Printf("Error connecting to the database: %s\n", err)
		return
	}
	defer db.Close()

	// Create a database queries instance
	dbQueries := database.New(db)

	// Create an instance of apiConfig and store the database connection
	apiCfg := &apiConfig{
		DB: db,
	}

	// Get the port from environment variable or default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Create a ServeMux
	mux := http.NewServeMux()

	// Add CORS middleware
	mux.HandleFunc("/", middlewareCors(rootHandler))

	// Add a handler to create a user
	mux.HandleFunc("/v1/users", createUserHandler(apiCfg))

	// Add a readiness handler
	mux.HandleFunc("/v1/readiness", readinessHandler)

	// Add an error handler
	mux.HandleFunc("/v1/err", errorHandler)

	// Create an HTTP server
	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	// Start the server
	fmt.Printf("Server listening on port %s\n", port)
	err = server.ListenAndServe()
	if err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}
}

func middlewareCors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	// You can add your CRUD operations here
}

func createUserHandler(apiCfg *apiConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user struct {
			Name string `json:"name"`
		}
		err := json.NewDecoder(r.Body).Decode(&user)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}

		// Generate UUID for the user
		userID := uuid.New()

		// Get current time
		currentTime := time.Now().UTC()

		// Insert the user into the database
		_, err = apiCfg.DB.Exec("INSERT INTO users (id, created_at, updated_at, name) VALUES ($1, $2, $3, $4)",
			userID, currentTime, currentTime, user.Name)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to create user")
			return
		}

		// Respond with the created user
		respondWithJSON(w, http.StatusCreated, map[string]interface{}{
			"id":         userID,
			"created_at": currentTime,
			"updated_at": currentTime,
			"name":       user.Name,
		})
	}
}

func respondWithJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	respondWithJSON(w, code, map[string]string{"error": msg})
}

func readinessHandler(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func errorHandler(w http.ResponseWriter, r *http.Request) {
	respondWithError(w, http.StatusInternalServerError, "Internal Server Error")
}
