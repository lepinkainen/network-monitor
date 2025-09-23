package web

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"

	"network-monitor/internal/database"
)

// Server handles web requests
type Server struct {
	db          *database.DB
	port        int
	staticFiles fs.FS
}

// New creates a new web server
func New(db *database.DB, port int, staticFS fs.FS) *Server {
	return &Server{
		db:          db,
		port:        port,
		staticFiles: staticFS,
	}
}

// Start starts the web server
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/recent", s.handleRecent)
	mux.HandleFunc("/api/stats", s.handleStats)
	mux.HandleFunc("/api/outages", s.handleOutages)
	mux.HandleFunc("/api/heatmap", s.handleHeatmap)
	mux.HandleFunc("/api/patterns", s.handlePatterns)

	// Static files - serve embedded static/ directory as webroot
	staticFS, _ := fs.Sub(s.staticFiles, "static")
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	log.Printf("Web server starting on port %d", s.port)
	return http.ListenAndServe(fmt.Sprintf(":%d", s.port), mux)
}
