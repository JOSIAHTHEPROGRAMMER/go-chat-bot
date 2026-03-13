package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/JOSIAHTHEPROGRAMMER/go-chat-bot/config"
	"github.com/JOSIAHTHEPROGRAMMER/go-chat-bot/fetcher"
	"github.com/JOSIAHTHEPROGRAMMER/go-chat-bot/llm"
	"github.com/JOSIAHTHEPROGRAMMER/go-chat-bot/middleware"
	"github.com/JOSIAHTHEPROGRAMMER/go-chat-bot/rag"
	"github.com/JOSIAHTHEPROGRAMMER/go-chat-bot/routes"
	"github.com/JOSIAHTHEPROGRAMMER/go-chat-bot/session"
	"github.com/JOSIAHTHEPROGRAMMER/go-chat-bot/tools"

	"github.com/joho/godotenv"
)

func main() {
	fmt.Println("Server starting...")

	_ = godotenv.Load()

	// Embedder — Gemini

	llm.RegisterEmbedder(&llm.GeminiEmbedder{})
	fmt.Println("Gemini embedder registered")

	// LLM — start on Groq, rotates to Gemini every 6 hours

	config.SetCurrentModel("groq")
	fmt.Println("Active provider: groq")

	// MongoDB

	if err := session.Connect(); err != nil {
		log.Fatalf("MongoDB connect failed: %v", err)
	}
	fmt.Println("MongoDB connected")

	// Qdrant

	if err := rag.InitStore(); err != nil {
		log.Fatalf("Qdrant init failed: %v", err)
	}
	fmt.Println("Qdrant store ready")

	// GitHub ingestion + embedding

	docs, err := fetcher.FetchREADMEs()
	if err != nil {
		log.Fatalf("GitHub fetch failed: %v", err)
	}
	fmt.Printf("Fetched %d READMEs from GitHub\n", len(docs))

	embedded, err := rag.EmbedAndStore(docs)
	if err != nil {
		log.Fatalf("Embedding failed: %v", err)
	}
	fmt.Printf("Embedded %d docs into Qdrant\n", embedded)

	// Tools

	tools.Register(&tools.GetProjectTool{})
	tools.Register(&tools.FilterByTechTool{})

	// Refresh embeddings + rotate provider every 6 hours

	go func() {
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			if config.GetCurrentModel() == "groq" {
				config.SetCurrentModel("gemini")
			} else {
				config.SetCurrentModel("groq")
			}
			fmt.Println("Switched provider to:", config.GetCurrentModel())

			fmt.Println("Refreshing embeddings...")
			refreshDocs, err := fetcher.FetchREADMEs()
			if err != nil {
				fmt.Printf("refresh fetch failed: %v\n", err)
				continue
			}
			count, err := rag.EmbedAndStore(refreshDocs)
			if err != nil {
				fmt.Printf("refresh embed failed: %v\n", err)
				continue
			}
			fmt.Printf("Refreshed %d docs\n", count)
		}
	}()

	// HTTP routes

	mux := http.NewServeMux()
	mux.HandleFunc("/health", middleware.CORS(routes.HealthHandler))
	mux.HandleFunc("/chat", middleware.Chain(routes.ChatHandler))
	mux.HandleFunc("/stream", middleware.Chain(routes.StreamHandler))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		fmt.Printf("Server running on port %s\n", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-quit
	fmt.Println("Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}
	fmt.Println("Server stopped")
}
