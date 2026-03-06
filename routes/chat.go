package routes

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/JOSIAHTHEPROGRAMMER/portfolio-backend/config"
	"github.com/JOSIAHTHEPROGRAMMER/portfolio-backend/llm"
	"github.com/JOSIAHTHEPROGRAMMER/portfolio-backend/logger"
	"github.com/JOSIAHTHEPROGRAMMER/portfolio-backend/planner"
)

type ChatRequest struct {
	Question string        `json:"question"`
	History  []llm.Message `json:"history"` // prior turns, empty on first message
}

type ChatResponse struct {
	Answer   string `json:"answer"`
	PlanType string `json:"plan_type"`
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

	// Resolve the active LLM provider
	provider, err := llm.Get(config.GetCurrentModel())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Provider error: %v", err)
		return
	}

	// Write provider name into the request log for observability
	if rl := logger.FromContext(r.Context()); rl != nil {
		rl.Provider = provider.Name()
	}

	// Planner classifies the question and builds the full message slice
	plan, err := planner.Build(r.Context(), req.Question, req.History, provider)
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

	json.NewEncoder(w).Encode(ChatResponse{
		Answer:   answer,
		PlanType: string(plan.Type),
	})
}
