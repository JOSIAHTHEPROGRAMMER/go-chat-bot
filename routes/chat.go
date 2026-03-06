package routes

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/JOSIAHTHEPROGRAMMER/portfolio-backend/config"
	"github.com/JOSIAHTHEPROGRAMMER/portfolio-backend/llm"
	"github.com/JOSIAHTHEPROGRAMMER/portfolio-backend/logger"
	"github.com/JOSIAHTHEPROGRAMMER/portfolio-backend/planner"
	"github.com/JOSIAHTHEPROGRAMMER/portfolio-backend/session"
)

type ChatRequest struct {
	Question  string `json:"question"`
	SessionID string `json:"session_id"` // empty on first message, returned after
}

type ChatResponse struct {
	Answer    string `json:"answer"`
	PlanType  string `json:"plan_type"`
	SessionID string `json:"session_id"` // client stores this and sends it back next turn
}

func ChatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprint(w, "Only POST allowed")
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "Invalid JSON")
		return
	}

	ctx := r.Context()

	// Create a new session if this is the first message
	sessionID := req.SessionID
	if sessionID == "" {
		var err error
		sessionID, err = session.New(ctx)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Session error: %v", err)
			return
		}
	}

	// Load conversation history for this session
	history, err := session.Load(ctx, sessionID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Session load error: %v", err)
		return
	}

	// Resolve the active LLM provider
	provider, err := llm.Get(config.GetCurrentModel())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Provider error: %v", err)
		return
	}

	// Write provider name into the request log for observability
	if rl := logger.FromContext(ctx); rl != nil {
		rl.Provider = provider.Name()
	}

	// Planner classifies the question and builds the full message slice
	plan, err := planner.Build(ctx, req.Question, history, provider)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Planner error: %v", err)
		return
	}

	// Send the full conversation to the LLM
	answer, err := provider.Chat(plan.Messages)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "LLM error: %v", err)
		return
	}

	// Persist the new turn to MongoDB
	if err := session.Append(ctx, sessionID, req.Question, answer); err != nil {
		// Non-fatal - log it but still return the answer
		fmt.Printf("session append error: %v\n", err)
	}

	json.NewEncoder(w).Encode(ChatResponse{
		Answer:    answer,
		PlanType:  string(plan.Type),
		SessionID: sessionID,
	})
}
