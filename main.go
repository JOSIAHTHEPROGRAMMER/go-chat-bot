package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/JOSIAHTHEPROGRAMMER/portfolio-backend/config"
	"github.com/JOSIAHTHEPROGRAMMER/portfolio-backend/rag"
	"github.com/JOSIAHTHEPROGRAMMER/portfolio-backend/routes"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	config.SetCurrentModel(os.Getenv("GROQ_MODEL"))
	fmt.Println("Server starting...")
	fmt.Println("Initial model:", config.GetCurrentModel())

	// Try to load existing embeddings from disk first.
	// This avoids re-embedding everything on every restart.
	if err := rag.LoadFromDisk(); err != nil {
		fmt.Println("No existing embeddings found, generating fresh ones...")

		docs, err := rag.EmbedAllReadmes()
		if err != nil {
			log.Fatal("Failed to generate initial embeddings:", err)
		}
		fmt.Printf("Generated embeddings for %d docs\n", len(docs))
	} else {
		fmt.Println("Loaded embeddings from disk into memory store")
	}

	// Periodically switch models and refresh embeddings
	go autoUpdateRoutine()

	http.HandleFunc("/chat", routes.ChatHandler)

	fmt.Println("Server running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
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
		fmt.Printf("Refreshed embeddings for %d docs\n", len(docs))
	}
}

// switchModel alternates between Groq and Gemini on each tick.
func switchModel() {
	if config.GetCurrentModel() == os.Getenv("GROQ_MODEL") {
		config.SetCurrentModel(os.Getenv("GEMINI_MODEL"))
	} else {
		config.SetCurrentModel(os.Getenv("GROQ_MODEL"))
	}
}
