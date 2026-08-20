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
	"os"
	"path/filepath"
	"strings"

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
	s.setupStaticRoutes()
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

		// Executions
		r.Get("/projects/{projectID}/executions", s.handleListExecutions)
		r.Get("/projects/{projectID}/executions/{execID}", s.handleGetExecution)
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

	// We must have a concurrent reader to process Ping/Pong and Close frames from the client!
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				log.Printf("Client read loop exited: %v", err)
				return
			}
		}
	}()

	// Stream logs from Redis to the WebSocket
	for {
		select {
		case msg := <-ch:
			if err := conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload)); err != nil {
				log.Printf("Failed to write to client: %v", err)
				return
			}
		case <-done:
			log.Printf("Client disconnected from log stream")
			return
		case <-r.Context().Done():
			return
		}
	}
}

func (s *Server) handleListExecutions(w http.ResponseWriter, r *http.Request) {
	projectIDStr := chi.URLParam(r, "projectID")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	execs, err := s.projectRepo.ListExecutions(r.Context(), projectID)
	if err != nil {
		log.Printf("Error fetching executions: %v", err)
		http.Error(w, "Failed to fetch executions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(execs)
}

func (s *Server) handleGetExecution(w http.ResponseWriter, r *http.Request) {
	execIDStr := chi.URLParam(r, "execID")
	execID, err := uuid.Parse(execIDStr)
	if err != nil {
		http.Error(w, "Invalid execution ID", http.StatusBadRequest)
		return
	}

	exec, err := s.projectRepo.GetExecution(r.Context(), execID)
	if err != nil {
		log.Printf("Error fetching execution %s: %v", execIDStr, err)
		http.Error(w, "Execution not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(exec)
}

// Serve React App
// We will serve static files from the ./web/dist directory (built by Docker).
func (s *Server) setupStaticRoutes() {
	// If the user hits any non-API route, serve the React app (SPA fallback)
	workDir, _ := os.Getwd()
	filesDir := http.Dir(filepath.Join(workDir, "web", "dist"))

	s.router.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		
		// If path doesn't start with /api, we serve the frontend
		if !strings.HasPrefix(path, "/api") {
			// Check if file exists in the dist directory
			fullPath := filepath.Join(workDir, "web", "dist", path)
			if _, err := os.Stat(fullPath); os.IsNotExist(err) || path == "/" {
				// If file doesn't exist, serve index.html (SPA routing)
				http.ServeFile(w, r, filepath.Join(workDir, "web", "dist", "index.html"))
				return
			}
			
			// Otherwise serve the static file
			fs := http.StripPrefix("/", http.FileServer(filesDir))
			fs.ServeHTTP(w, r)
		}
	})
}
