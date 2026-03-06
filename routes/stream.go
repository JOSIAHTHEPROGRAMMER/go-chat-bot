package routes

import (
	"fmt"
	"net/http"
	"encoding/json"

	"github.com/JOSIAHTHEPROGRAMMER/portfolio-backend/config"
	"github.com/JOSIAHTHEPROGRAMMER/portfolio-backend/llm"
	"github.com/JOSIAHTHEPROGRAMMER/portfolio-backend/logger"
	"github.com/JOSIAHTHEPROGRAMMER/portfolio-backend/planner"
)

// StreamHandler is the streaming version of ChatHandler.
// It returns tokens as Server-Sent Events instead of waiting for the full response.
// The client reads the stream and appends tokens to the UI as they arrive.
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

	// Reuse the same planner - streaming is only a transport difference
	plan, err := planner.Build(r.Context(), req.Question, req.History, provider)
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

	// Provider streams tokens into this channel
	tokenCh := make(chan string)

	// Run the stream in a goroutine so we can flush tokens as they arrive
	errCh := make(chan error, 1)
	go func() {
		errCh <- provider.Stream(plan.Messages, tokenCh)
	}()

	// Forward each token to the client as an SSE event
	for token := range tokenCh {
		fmt.Fprintf(w, "data: %s\n\n", token)
		flusher.Flush()
	}

	// Check if the stream ended with an error
	if err := <-errCh; err != nil {
		fmt.Fprintf(w, "event: error\ndata: %v\n\n", err)
		flusher.Flush()
	}

	// Signal the client that the stream is complete
	fmt.Fprintf(w, "event: done\ndata: \n\n")
	flusher.Flush()
}
