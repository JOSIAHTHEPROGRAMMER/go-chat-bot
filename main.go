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

	// Set initial model
	config.SetCurrentModel(os.Getenv("GROQ_MODEL"))
	fmt.Println("Server starting...")
	fmt.Println("Initial model:", config.GetCurrentModel())

	// Initial embeddings
	docs, err := rag.EmbedAllReadmes()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Initial embeddings generated for %d docs\n", len(docs))

	// Start auto update
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

func switchModel() {
	if config.GetCurrentModel() == os.Getenv("GROQ_MODEL") {
		config.SetCurrentModel(os.Getenv("GEMINI_MODEL"))
	} else {
		config.SetCurrentModel(os.Getenv("GROQ_MODEL"))
	}
}
