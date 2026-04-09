package web

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/aweist/schedule-watcher/league"
	"github.com/aweist/schedule-watcher/models"
	"github.com/aweist/schedule-watcher/notifier"
	"github.com/aweist/schedule-watcher/storage"
)

//go:embed templates/*
var templates embed.FS

//go:embed static/*
var staticFiles embed.FS

type Server struct {
	storage   *storage.BoltStorage
	notifier  notifier.Notifier
	port      string
	leagues   []league.League
}

type LeagueTeam struct {
	League   string
	TeamKey  string
	TeamName string
}

type PageData struct {
	Games         []models.Game
	NotifiedGames []models.NotifiedGame
	CurrentTime   string
	CurrentDate   string
	Now           time.Time
	Leagues       []league.League
}

type AdminPageData struct {
	Recipients  []models.EmailRecipient
	LeagueTeams []LeagueTeam
}

type SnapshotsPageData struct {
	Snapshots []models.Snapshot
}

func NewServer(storage *storage.BoltStorage, port string, leagues []league.League) *Server {
	return &Server{
		storage: storage,
		port:    port,
		leagues: leagues,
	}
}

func (s *Server) SetNotifier(n notifier.Notifier) {
	s.notifier = n
}

func (s *Server) Start() {
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Printf("Failed to create static file system: %v", err)
	}
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	http.HandleFunc("/", s.handleDebugPage)
	http.HandleFunc("/admin", s.handleAdminPage)
	http.HandleFunc("/snapshots", s.handleSnapshotsPage)
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

func (s *Server) handleDebugPage(w http.ResponseWriter, r *http.Request) {
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

	sort.Slice(games, func(i, j int) bool {
		return games[i].Date.Before(games[j].Date)
	})

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
		Leagues:       s.leagues,
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, fmt.Sprintf("Template execution error: %v", err), http.StatusInternalServerError)
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

	leagueName := r.FormValue("league")
	teamKey := r.FormValue("team_key")
	gameID := r.FormValue("id")
	if gameID == "" {
		http.Error(w, "Game ID is required", http.StatusBadRequest)
		return
	}

	if err := s.storage.DeleteGame(leagueName, teamKey, gameID); err != nil {
		http.Error(w, fmt.Sprintf("Error deleting game: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Deleted game: %s/%s/%s", leagueName, teamKey, gameID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (s *Server) handleDeleteNotifiedGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	leagueName := r.FormValue("league")
	teamKey := r.FormValue("team_key")
	gameID := r.FormValue("id")
	if gameID == "" {
		http.Error(w, "Game ID is required", http.StatusBadRequest)
		return
	}

	if err := s.storage.DeleteNotifiedGame(leagueName, teamKey, gameID); err != nil {
		http.Error(w, fmt.Sprintf("Error deleting notified game: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Deleted notified game: %s/%s/%s", leagueName, teamKey, gameID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (s *Server) handleTestEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.notifier == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"message": "No email notifier configured",
		})
		return
	}

	// Get all active recipients across all leagues/teams
	allRecipients, err := s.storage.GetAllEmailRecipients()
	if err != nil || len(allRecipients) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"message": "No email recipients configured",
		})
		return
	}

	var emails []string
	for _, r := range allRecipients {
		if r.IsActive {
			emails = append(emails, r.Email)
		}
	}

	testGame := models.Game{
		ID:          "test-email-" + fmt.Sprintf("%d", time.Now().Unix()),
		League:      "test",
		TeamKey:     "test",
		TeamCaptain: "Test Team",
		TeamNumber:  99,
		Division:    "Test Division",
		Date:        time.Now().Add(24 * time.Hour),
		Time:        "7:00 PM",
		Court:       "Test Court",
	}

	if err := s.notifier.SendNotification(testGame, emails); err != nil {
		log.Printf("Test email failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"message": fmt.Sprintf("Email test failed: %v", err),
		})
		return
	}

	log.Printf("Test email sent successfully")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
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

	sort.Slice(recipients, func(i, j int) bool {
		if recipients[i].League != recipients[j].League {
			return recipients[i].League < recipients[j].League
		}
		if recipients[i].TeamKey != recipients[j].TeamKey {
			return recipients[i].TeamKey < recipients[j].TeamKey
		}
		return recipients[i].Name < recipients[j].Name
	})

	// Build league/team list for the add form dropdown
	var leagueTeams []LeagueTeam
	for _, lg := range s.leagues {
		for _, t := range lg.Teams() {
			leagueTeams = append(leagueTeams, LeagueTeam{
				League:   lg.Name(),
				TeamKey:  t.Key,
				TeamName: fmt.Sprintf("%s - %s", lg.DisplayName(), t.Name),
			})
		}
	}

	data := AdminPageData{
		Recipients:  recipients,
		LeagueTeams: leagueTeams,
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, fmt.Sprintf("Template execution error: %v", err), http.StatusInternalServerError)
	}
}

func (s *Server) handleSnapshotsPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFS(templates, "templates/snapshots.html")
	if err != nil {
		http.Error(w, fmt.Sprintf("Template error: %v", err), http.StatusInternalServerError)
		return
	}

	snapshots, err := s.storage.GetAllSnapshots()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching snapshots: %v", err), http.StatusInternalServerError)
		return
	}

	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].FetchedAt.After(snapshots[j].FetchedAt)
	})

	data := SnapshotsPageData{
		Snapshots: snapshots,
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
	leagueName := r.FormValue("league")
	teamKey := r.FormValue("team_key")

	if name == "" || email == "" || leagueName == "" || teamKey == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"message": "Name, email, league, and team are required",
		})
		return
	}

	recipient := models.EmailRecipient{
		ID:       generateID(),
		League:   leagueName,
		TeamKey:  teamKey,
		Name:     name,
		Email:    email,
		AddedAt:  time.Now(),
		IsActive: true,
	}

	if err := s.storage.AddRecipientForTeam(leagueName, teamKey, recipient); err != nil {
		log.Printf("Error adding recipient: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"message": "Error adding recipient",
		})
		return
	}

	log.Printf("Added email recipient: %s (%s) for %s/%s", name, email, leagueName, teamKey)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (s *Server) handleDeleteRecipient(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	recipientID := r.FormValue("id")
	leagueName := r.FormValue("league")
	teamKey := r.FormValue("team_key")

	if recipientID == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"message": "Recipient ID is required",
		})
		return
	}

	if err := s.storage.DeleteEmailRecipient(leagueName, teamKey, recipientID); err != nil {
		log.Printf("Error deleting recipient: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
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
			"status":  "error",
			"message": "Recipient ID and active status are required",
		})
		return
	}

	active, err := strconv.ParseBool(activeStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"message": "Invalid active status",
		})
		return
	}

	recipient, err := s.storage.GetEmailRecipient(recipientID)
	if err != nil || recipient == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"message": "Recipient not found",
		})
		return
	}

	recipient.IsActive = active
	if err := s.storage.UpdateEmailRecipient(*recipient); err != nil {
		log.Printf("Error updating recipient: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"message": "Error updating recipient",
		})
		return
	}

	log.Printf("Updated recipient %s (%s) - Active: %v", recipient.Name, recipient.Email, active)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
