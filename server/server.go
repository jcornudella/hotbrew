// Package server provides the hotbrew API server
package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

var emailRegex = regexp.MustCompile(`^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$`)

// Subscriber represents a newsletter subscriber
type Subscriber struct {
	Token     string     `json:"token"`
	Email     string     `json:"email,omitempty"`
	Name      string     `json:"name,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	Config    UserConfig `json:"config"`
}

// UserConfig holds subscriber preferences
type UserConfig struct {
	Theme         string   `json:"theme"`
	HNEnabled     bool     `json:"hn_enabled"`
	HNMax         int      `json:"hn_max"`
	GitHubEnabled bool     `json:"github_enabled"`
	GitHubTopics  []string `json:"github_topics"`
	SearchTerms   []string `json:"search_terms"`
}

// NewsletterContent represents the daily newsletter data
type NewsletterContent struct {
	GeneratedAt time.Time        `json:"generated_at"`
	Sections    []ContentSection `json:"sections"`
}

// ContentSection is a section of the newsletter
type ContentSection struct {
	Name  string        `json:"name"`
	Icon  string        `json:"icon"`
	Items []ContentItem `json:"items"`
}

// ContentItem is a single item in the newsletter
type ContentItem struct {
	ID        string                 `json:"id"`
	Title     string                 `json:"title"`
	Subtitle  string                 `json:"subtitle"`
	URL       string                 `json:"url"`
	Timestamp time.Time              `json:"timestamp"`
	Priority  int                    `json:"priority"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Server is the hotbrew API server
type Server struct {
	subscribers map[string]*Subscriber
	mu          sync.RWMutex
	dataFile    string
}

// New creates a new server
func New(dataDir string) *Server {
	s := &Server{
		subscribers: make(map[string]*Subscriber),
		dataFile:    filepath.Join(dataDir, "subscribers.json"),
	}
	s.load()
	return s
}

// load reads subscribers from disk
func (s *Server) load() {
	data, err := os.ReadFile(s.dataFile)
	if err != nil {
		return
	}
	json.Unmarshal(data, &s.subscribers)
}

// save writes subscribers to disk
func (s *Server) save() {
	data, _ := json.MarshalIndent(s.subscribers, "", "  ")
	if err := os.MkdirAll(filepath.Dir(s.dataFile), 0o700); err != nil {
		fmt.Printf("error creating data dir: %v\n", err)
		return
	}
	if err := os.WriteFile(s.dataFile, data, 0o600); err != nil {
		fmt.Printf("error writing subscribers: %v\n", err)
	}
}

// generateToken creates a unique token
func generateToken() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// Handler returns the HTTP handler
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/subscribe", s.handleSubscribe)
	mux.HandleFunc("/api/config/", s.handleConfig)
	mux.HandleFunc("/api/newsletter/", s.handleNewsletter)
	mux.HandleFunc("/api/health", s.handleHealth)

	// Static files (website)
	mux.Handle("/", http.FileServer(http.Dir("web")))

	// CORS middleware
	return corsMiddleware(mux)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// POST /api/subscribe
func (s *Server) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}

	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if req.Email == "" || len(req.Email) > 320 || !emailRegex.MatchString(req.Email) {
		http.Error(w, "Valid email required", http.StatusBadRequest)
		return
	}
	if len(req.Name) > 256 {
		http.Error(w, "Name too long", http.StatusBadRequest)
		return
	}

	token := generateToken()

	sub := &Subscriber{
		Token:     token,
		Email:     req.Email,
		Name:      req.Name,
		CreatedAt: time.Now(),
		Config: UserConfig{
			Theme:         "synthwave",
			HNEnabled:     true,
			HNMax:         8,
			GitHubEnabled: true,
			GitHubTopics:  []string{"ai", "llm", "machine-learning"},
			SearchTerms:   []string{"Claude Code", "vibe coding", "AI coding"},
		},
	}

	s.mu.Lock()
	s.subscribers[token] = sub
	s.save()
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"token":   token,
		"message": "Welcome to hotbrew! Run: hotbrew login " + token,
	})
}

// GET /api/config/:token
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Path[len("/api/config/"):]
	if token == "" {
		http.Error(w, "Token required", http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	sub, ok := s.subscribers[token]
	s.mu.RUnlock()

	if !ok {
		http.Error(w, "Invalid token", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sub.Config)
}

// GET /api/newsletter/:token
func (s *Server) handleNewsletter(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Path[len("/api/newsletter/"):]
	if token == "" {
		http.Error(w, "Token required", http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	_, ok := s.subscribers[token]
	s.mu.RUnlock()

	if !ok {
		http.Error(w, "Invalid token", http.StatusNotFound)
		return
	}

	// Return a response that tells the CLI to fetch fresh content
	// In a production setup, you'd cache this and update periodically
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"fetch_live": true,
		"message":    "Fetch content directly from sources",
	})
}

// GET /api/health
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"time":   time.Now(),
	})
}

// Run starts the server
func Run(addr string, dataDir string) error {
	s := New(dataDir)
	fmt.Printf("☕ hotbrew server running at http://%s\n", addr)
	return http.ListenAndServe(addr, s.Handler())
}
