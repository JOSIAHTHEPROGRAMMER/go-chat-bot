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
	"github.com/JOSIAHTHEPROGRAMMER/go-chat-bot/middleware"
	"github.com/JOSIAHTHEPROGRAMMER/go-chat-bot/rag"
	"github.com/JOSIAHTHEPROGRAMMER/go-chat-bot/routes"
	"github.com/JOSIAHTHEPROGRAMMER/go-chat-bot/session"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	config.SetCurrentModel("groq")
	fmt.Println("Server starting...")
	fmt.Println("Initial model:", config.GetCurrentModel())

	// Connect to MongoDB
	if err := session.Connect(); err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}
	fmt.Println("MongoDB connected")

	// Initialize the Qdrant collection - creates it if it does not exist yet
	if err := rag.InitStore(); err != nil {
		log.Fatal("Failed to initialize Qdrant store:", err)
	}
	fmt.Println("Qdrant store ready")

	// Embed all READMEs and upsert into Qdrant on every startup.
	// Upserts are idempotent so re-running is safe and keeps data fresh.
	docs, err := rag.EmbedAllReadmes()
	if err != nil {
		log.Fatal("Failed to generate initial embeddings:", err)
	}
	fmt.Printf("Embedded %d docs into Qdrant\n", len(docs))

	go autoUpdateRoutine()

	// /health is unauthenticated and not rate limited so Render can reach it freely
	http.HandleFunc("/health", middleware.CORS(routes.HealthHandler))
	http.HandleFunc("/chat", middleware.Chain(routes.ChatHandler))
	http.HandleFunc("/stream", middleware.Chain(routes.StreamHandler))

	// Render injects PORT - fall back to 8080 for local development
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{Addr: ":" + port}

	// Listen for SIGTERM and SIGINT so we can drain in-flight requests before exit.
	// Render sends SIGTERM when it stops or redeploys the service.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		fmt.Printf("Server running on port %s\n", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server error:", err)
		}
	}()

	// Block until a signal is received
	<-quit
	fmt.Println("Shutting down - draining in-flight requests...")

	// Give in-flight requests up to 15 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Forced shutdown:", err)
	}

	fmt.Println("Server stopped cleanly")
}

func autoUpdateRoutine() {
	ticker := time.NewTicker(6 * time.Hour)
	for range ticker.C {
		switchModel()
		fmt.Println("Switched model to:", config.GetCurrentModel())

		docs, err := rag.EmbedAllReadmes()
		if err != nil {
			fmt.Println("Error updating embeddings:", err)
			continue
		}
		fmt.Printf("Refreshed %d docs in Qdrant\n", len(docs))
	}
}

// switchModel alternates between Groq and Gemini on each tick.
func switchModel() {
	if config.GetCurrentModel() == "groq" {
		config.SetCurrentModel("gemini")
	} else {
		config.SetCurrentModel("groq")
	}
}
