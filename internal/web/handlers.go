package web

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// handleRecent handles /api/recent requests
func (s *Server) handleRecent(w http.ResponseWriter, r *http.Request) {
	hours := 24
	if h := r.URL.Query().Get("hours"); h != "" {
		if parsed, err := strconv.Atoi(h); err == nil {
			hours = parsed
		}
	}

	results, err := s.db.GetRecent(hours)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// handleStats handles /api/stats requests
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.db.GetStats(24)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handleOutages handles /api/outages requests
func (s *Server) handleOutages(w http.ResponseWriter, r *http.Request) {
	outages, err := s.db.GetOutages(7)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(outages)
}

// handleHeatmap handles /api/heatmap requests
func (s *Server) handleHeatmap(w http.ResponseWriter, r *http.Request) {
	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil {
			days = parsed
		}
	}

	heatmapData, err := s.db.GetHeatmapData(days)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(heatmapData)
}

// handlePatterns handles /api/patterns requests
func (s *Server) handlePatterns(w http.ResponseWriter, r *http.Request) {
	// Get daily patterns for specific hour
	hour := r.URL.Query().Get("hour")
	if hour == "" {
		http.Error(w, "hour parameter required", http.StatusBadRequest)
		return
	}

	patterns, err := s.db.GetPatterns(hour)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(patterns)
}
