package debug

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/cloudwego/eino-ext/devops"
	"github.com/dyike/CortexGo/internal/config"
)

// SimpleDebugServer provides a standalone debug server without graph dependencies
type SimpleDebugServer struct {
	config *config.Config
	ctx    context.Context
}

func NewSimpleDebugServer(cfg *config.Config) *SimpleDebugServer {
	return &SimpleDebugServer{
		config: cfg,
		ctx:    context.Background(),
	}
}

func (s *SimpleDebugServer) Start() error {
	if s.config.Debug {
		log.Printf("[SimpleDebug] Initializing Eino visual debug plugin on port %d", s.config.EinoDebugPort)
	}

	// Initialize eino devops
	err := devops.Init(s.ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize Eino debug plugin: %w", err)
	}

	if s.config.Debug {
		log.Printf("[SimpleDebug] Successfully initialized debug server at http://localhost:%d", s.config.EinoDebugPort)
		log.Printf("[SimpleDebug] You can now debug Eino orchestration artifacts through the web interface")
	}

	// Start a simple health check endpoint
	go s.startHealthServer()

	return nil
}

func (s *SimpleDebugServer) startHealthServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("CortexGo Debug Server is running"))
	})
	
	healthPort := s.config.EinoDebugPort + 1
	log.Printf("[SimpleDebug] Health check available at http://localhost:%d/health", healthPort)
	
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", healthPort),
		Handler: mux,
		ReadTimeout: 5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	
	if err := server.ListenAndServe(); err != nil {
		log.Printf("[SimpleDebug] Health server error: %v", err)
	}
}

func (s *SimpleDebugServer) GetDebugURL() string {
	return fmt.Sprintf("http://localhost:%d", s.config.EinoDebugPort)
}