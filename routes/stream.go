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

// StreamHandler is the streaming version of ChatHandler.
// Returns tokens as Server-Sent Events instead of waiting for the full response.
func StreamHandler(w http.ResponseWriter, r *http.Request) {
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

	// Reuse the same planner - streaming is only a transport difference
	plan, err := planner.Build(ctx, req.Question, history, provider)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Planner error: %v", err)
		return
	}

	// SSE headers - tells the browser to keep the connection open and read events
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send the session ID first so the client can store it before tokens arrive
	fmt.Fprintf(w, "event: session\ndata: %s\n\n", sessionID)
	flusher.Flush()

	tokenCh := make(chan string)
	errCh := make(chan error, 1)

	// Collect the full answer while streaming so we can persist it
	var fullAnswer string

	go func() {
		errCh <- provider.Stream(plan.Messages, tokenCh)
	}()

	for token := range tokenCh {
		fullAnswer += token
		fmt.Fprintf(w, "data: %s\n\n", token)
		flusher.Flush()
	}

	if err := <-errCh; err != nil {
		fmt.Fprintf(w, "event: error\ndata: %v\n\n", err)
		flusher.Flush()
	}

	// Persist the completed turn to MongoDB
	if err := session.Append(ctx, sessionID, req.Question, fullAnswer); err != nil {
		fmt.Printf("session append error: %v\n", err)
	}

	// Signal the client that the stream is complete
	fmt.Fprintf(w, "event: done\ndata: \n\n")
	flusher.Flush()
}
