package routes

import (
	"encoding/json"
	"net/http"
	"time"
)

// healthResponse is what the /health endpoint returns.
type healthResponse struct {
	Status string `json:"status"`
	Time   string `json:"time"`
}

// HealthHandler returns a 200 with a simple status payload.
// Used by Render and uptime monitors to confirm the server is alive.
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(healthResponse{
		Status: "ok",
		Time:   time.Now().UTC().Format(time.RFC3339),
	})
}
