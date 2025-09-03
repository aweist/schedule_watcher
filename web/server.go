package web

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/aweist/schedule-watcher/models"
	"github.com/aweist/schedule-watcher/notifier"
	"github.com/aweist/schedule-watcher/storage"
)

//go:embed templates/*
var templates embed.FS

type Server struct {
	storage   *storage.BoltStorage
	notifiers []notifier.Notifier
	port      string
}

type PageData struct {
	Games         []models.Game
	NotifiedGames []models.NotifiedGame
	CurrentTime   string
	CurrentDate   string
	Now           time.Time
	TeamName      string
}

type AdminPageData struct {
	Recipients []models.EmailRecipient
}

func NewServer(storage *storage.BoltStorage, port string) *Server {
	return &Server{
		storage:   storage,
		notifiers: nil, // Will be set later
		port:      port,
	}
}

func (s *Server) SetNotifiers(notifiers []notifier.Notifier) {
	s.notifiers = notifiers
}

func (s *Server) Start(teamName string) {
	http.HandleFunc("/", s.handleDebugPage(teamName))
	http.HandleFunc("/admin", s.handleAdminPage)
	http.HandleFunc("/api/games", s.handleAPIGames)
	http.HandleFunc("/api/notified", s.handleAPINotified)
	http.HandleFunc("/api/game/delete", s.handleDeleteGame)
	http.HandleFunc("/api/notified/delete", s.handleDeleteNotifiedGame)
	http.HandleFunc("/api/test-email", s.handleTestEmail)
	http.HandleFunc("/api/recipients/add", s.handleAddRecipient)
	http.HandleFunc("/api/recipients/delete", s.handleDeleteRecipient)
	http.HandleFunc("/api/recipients/toggle", s.handleToggleRecipient)
	
	log.Printf("Starting debug web server on http://localhost:%s", s.port)
	if err := http.ListenAndServe(":"+s.port, nil); err != nil {
		log.Printf("Web server error: %v", err)
	}
}

func (s *Server) handleDebugPage(teamName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFS(templates, "templates/debug.html")
		if err != nil {
			http.Error(w, fmt.Sprintf("Template error: %v", err), http.StatusInternalServerError)
			return
		}

		games, err := s.storage.GetAllGames()
		if err != nil {
			http.Error(w, fmt.Sprintf("Error fetching games: %v", err), http.StatusInternalServerError)
			return
		}

		notifiedGames, err := s.storage.GetAllNotifiedGames()
		if err != nil {
			http.Error(w, fmt.Sprintf("Error fetching notified games: %v", err), http.StatusInternalServerError)
			return
		}

		// Sort games by date
		sort.Slice(games, func(i, j int) bool {
			return games[i].Date.Before(games[j].Date)
		})

		// Sort notified games by notification time (most recent first)
		sort.Slice(notifiedGames, func(i, j int) bool {
			return notifiedGames[i].NotifiedAt.After(notifiedGames[j].NotifiedAt)
		})

		now := time.Now()
		data := PageData{
			Games:         games,
			NotifiedGames: notifiedGames,
			CurrentTime:   now.Format("2006-01-02 15:04:05 MST"),
			CurrentDate:   now.Format("2006-01-02"),
			Now:           now,
			TeamName:      teamName,
		}

		if err := tmpl.Execute(w, data); err != nil {
			http.Error(w, fmt.Sprintf("Template execution error: %v", err), http.StatusInternalServerError)
		}
	}
}

func (s *Server) handleAPIGames(w http.ResponseWriter, r *http.Request) {
	games, err := s.storage.GetAllGames()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching games: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(games)
}

func (s *Server) handleAPINotified(w http.ResponseWriter, r *http.Request) {
	notifiedGames, err := s.storage.GetAllNotifiedGames()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching notified games: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notifiedGames)
}

func (s *Server) handleDeleteGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	gameID := r.FormValue("id")
	if gameID == "" {
		http.Error(w, "Game ID is required", http.StatusBadRequest)
		return
	}

	if err := s.storage.DeleteGame(gameID); err != nil {
		http.Error(w, fmt.Sprintf("Error deleting game: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Deleted game: %s", gameID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (s *Server) handleDeleteNotifiedGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	gameID := r.FormValue("id")
	if gameID == "" {
		http.Error(w, "Game ID is required", http.StatusBadRequest)
		return
	}

	if err := s.storage.DeleteNotifiedGame(gameID); err != nil {
		http.Error(w, fmt.Sprintf("Error deleting notified game: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Deleted notified game: %s", gameID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (s *Server) handleTestEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if len(s.notifiers) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "error", 
			"message": "No email notifiers configured",
		})
		return
	}

	// Create a test game
	testGame := models.Game{
		ID:          "test-email-" + fmt.Sprintf("%d", time.Now().Unix()),
		TeamCaptain: "Test Team",
		TeamNumber:  99,
		Division:    "Test Division",
		Date:        time.Now().Add(24 * time.Hour),
		Time:        "7:00 PM",
		Court:       "Test Court",
	}

	// Send test notification
	for _, n := range s.notifiers {
		if n.GetType() == "email" {
			if err := n.SendNotification(testGame); err != nil {
				log.Printf("Test email failed: %v", err)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{
					"status": "error", 
					"message": fmt.Sprintf("Email test failed: %v", err),
				})
				return
			}
		}
	}

	log.Printf("Test email sent successfully")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"message": "Test email sent successfully!",
	})
}

func (s *Server) handleAdminPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFS(templates, "templates/admin.html")
	if err != nil {
		http.Error(w, fmt.Sprintf("Template error: %v", err), http.StatusInternalServerError)
		return
	}

	recipients, err := s.storage.GetAllEmailRecipients()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching recipients: %v", err), http.StatusInternalServerError)
		return
	}

	// Sort recipients by name
	sort.Slice(recipients, func(i, j int) bool {
		return recipients[i].Name < recipients[j].Name
	})

	data := AdminPageData{
		Recipients: recipients,
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, fmt.Sprintf("Template execution error: %v", err), http.StatusInternalServerError)
	}
}

func generateID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (s *Server) handleAddRecipient(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := r.FormValue("name")
	email := r.FormValue("email")

	if name == "" || email == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "error",
			"message": "Name and email are required",
		})
		return
	}

	recipient := models.EmailRecipient{
		ID:       generateID(),
		Name:     name,
		Email:    email,
		AddedAt:  time.Now(),
		IsActive: true,
	}

	if err := s.storage.AddEmailRecipient(recipient); err != nil {
		log.Printf("Error adding recipient: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "error",
			"message": "Error adding recipient",
		})
		return
	}

	log.Printf("Added email recipient: %s (%s)", name, email)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (s *Server) handleDeleteRecipient(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	recipientID := r.FormValue("id")
	if recipientID == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "error",
			"message": "Recipient ID is required",
		})
		return
	}

	if err := s.storage.DeleteEmailRecipient(recipientID); err != nil {
		log.Printf("Error deleting recipient: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "error",
			"message": "Error deleting recipient",
		})
		return
	}

	log.Printf("Deleted email recipient: %s", recipientID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (s *Server) handleToggleRecipient(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	recipientID := r.FormValue("id")
	activeStr := r.FormValue("active")

	if recipientID == "" || activeStr == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "error",
			"message": "Recipient ID and active status are required",
		})
		return
	}

	active, err := strconv.ParseBool(activeStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "error",
			"message": "Invalid active status",
		})
		return
	}

	// Get existing recipient
	recipient, err := s.storage.GetEmailRecipient(recipientID)
	if err != nil || recipient == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "error",
			"message": "Recipient not found",
		})
		return
	}

	// Update status
	recipient.IsActive = active
	if err := s.storage.UpdateEmailRecipient(*recipient); err != nil {
		log.Printf("Error updating recipient: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "error",
			"message": "Error updating recipient",
		})
		return
	}

	log.Printf("Updated recipient %s (%s) - Active: %v", recipient.Name, recipient.Email, active)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}