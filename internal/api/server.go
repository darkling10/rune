package api

import (
	"encoding/json"
	"net/http"

	"github.com/deployerai/deployer/internal/models"
	"github.com/deployerai/deployer/internal/repository"
	"github.com/deployerai/deployer/internal/worker"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/go-github/v60/github"
	"github.com/google/uuid"
	"log"
	"fmt"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

// Server represents the HTTP API server for Deployer AI.
type Server struct {
	router          *chi.Mux
	projectRepo     repository.ProjectRepository
	taskDistributor worker.TaskDistributor
	rdb             *redis.Client
}

// NewServer initializes a new API server with standard middleware and routes.
func NewServer(repo repository.ProjectRepository, taskDistributor worker.TaskDistributor, rdb *redis.Client) *Server {
	s := &Server{
		router:          chi.NewRouter(),
		projectRepo:     repo,
		taskDistributor: taskDistributor,
		rdb:             rdb,
	}
	s.setupMiddleware()
	s.setupRoutes()
	return s
}

func (s *Server) setupMiddleware() {
	// A good practice: inject common middleware for security and observability
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.SetHeader("Content-Type", "application/json"))

	// Basic CORS for UI development
	s.router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*") // Allowing all for dev, restrict in prod
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	})
}

func (s *Server) setupRoutes() {
	s.router.Get("/health", s.handleHealthCheck)

	s.router.Route("/api/v1", func(r chi.Router) {
		r.Get("/projects", s.handleListProjects)
		r.Post("/projects", s.handleCreateProject)
		r.Get("/projects/{projectID}", s.handleGetProject)

		// Credentials endpoints
		r.Post("/projects/{projectID}/credentials", s.handleAddCredential)

		// Logs
		r.Get("/projects/{projectID}/logs/stream", s.handleLogStream)

		// Webhooks
		r.Post("/projects/{projectID}/webhooks/github", s.handleGitHubWebhook)
	})
}

// ServeHTTP implements the http.Handler interface.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := s.projectRepo.ListProjects(r.Context())
	if err != nil {
		http.Error(w, "Failed to list projects: "+err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(projects)
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	project := models.NewProject(req.Name)
	if err := s.projectRepo.CreateProject(r.Context(), project); err != nil {
		http.Error(w, "Failed to create project: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Enqueue a dummy task to demonstrate the worker architecture
	payload := &worker.SnykScanPayload{
		ProjectID: project.ID,
		CommitSHA: "initial-commit",
	}
	if err := s.taskDistributor.DistributeTaskSnykScan(r.Context(), payload); err != nil {
		// Log error but don't fail the request since project was created
		// In production, you might want transactional outbox pattern
		_ = err
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(project)
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	projectIDStr := chi.URLParam(r, "projectID")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	project, err := s.projectRepo.GetProjectByID(r.Context(), projectID)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(project)
}

func (s *Server) handleAddCredential(w http.ResponseWriter, r *http.Request) {
	projectIDStr := chi.URLParam(r, "projectID")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	// Validate project exists before adding credential
	if _, err := s.projectRepo.GetProjectByID(r.Context(), projectID); err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	var req struct {
		Provider models.ProviderType `json:"provider"`
		APIKey   string              `json:"api_key"`
		BaseURL  string              `json:"base_url,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cred := models.NewLLMCredential(projectID, req.Provider, req.APIKey)
	cred.BaseURL = req.BaseURL

	if err := s.projectRepo.CreateCredential(r.Context(), cred); err != nil {
		http.Error(w, "Failed to store credential: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// In a real scenario, we don't return the API key.
	response := map[string]string{
		"id":       cred.ID.String(),
		"message":  "Credential stored securely for project " + projectIDStr,
		"provider": string(req.Provider),
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
	projectIDStr := chi.URLParam(r, "projectID")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	// Validate project exists
	if _, err := s.projectRepo.GetProjectByID(r.Context(), projectID); err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	// For a real production app, you'd fetch the project's specific webhook secret from DB.
	// We'll use a hardcoded/env secret for this demo.
	// secret := []byte(os.Getenv("GITHUB_WEBHOOK_SECRET"))
	// payload, err := github.ValidatePayload(r, secret)

	// Since this is a demo, we'll just read the body directly to avoid needing a valid signature from curl
	payload, err := github.ValidatePayload(r, nil) // nil secret skips signature verification for testing
	if err != nil {
		http.Error(w, "Error validating request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		http.Error(w, "Error parsing webhook: "+err.Error(), http.StatusBadRequest)
		return
	}

	switch e := event.(type) {
	case *github.PushEvent:
		log.Printf("Received PushEvent for repo: %s, commit: %s", e.GetRepo().GetFullName(), e.GetHeadCommit().GetID())

		taskPayload := &worker.RunPipelinePayload{
			ProjectID: projectID,
			CommitSHA: e.GetHeadCommit().GetID(),
			RepoURL:   e.GetRepo().GetCloneURL(),
		}

		if err := s.taskDistributor.DistributeTaskRunPipeline(r.Context(), taskPayload); err != nil {
			log.Printf("Failed to enqueue pipeline task: %v", err)
			http.Error(w, "Failed to enqueue task", http.StatusInternalServerError)
			return
		}

	case *github.PullRequestEvent:
		log.Printf("Received PullRequestEvent for repo: %s, PR: %d", e.GetRepo().GetFullName(), e.GetNumber())
		// You could enqueue a slightly different task for PRs (e.g. just tests, no deployment)

	default:
		log.Printf("Ignored Webhook event type: %s", github.WebHookType(r))
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Webhook processed successfully"))
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for demo
	},
}

func (s *Server) handleLogStream(w http.ResponseWriter, r *http.Request) {
	projectIDStr := chi.URLParam(r, "projectID")
	
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	channel := fmt.Sprintf("logs:%s", projectIDStr)
	pubsub := s.rdb.Subscribe(r.Context(), channel)
	defer pubsub.Close()

	ch := pubsub.Channel()
	
	log.Printf("Client connected to log stream for project %s", projectIDStr)
	
	// Optional: send an initial connection success message
	_ = conn.WriteMessage(websocket.TextMessage, []byte("\033[1;36mConnected to Rune Log Stream...\033[0m\n"))

	// Stream logs from Redis to the WebSocket
	for msg := range ch {
		if err := conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload)); err != nil {
			log.Printf("Client disconnected from log stream: %v", err)
			break
		}
	}
}
