package rag

import "fmt"

// client is the package-level Qdrant client.
// Initialized in InitStore() so godotenv.Load() runs first.
var client *qdrantClient

// Set upserts a single doc into Qdrant.
func Set(doc Doc) error {
	return client.Upsert(doc)
}

// StoreAll returns every doc in the collection.
// Used by the tools package for keyword scanning.
func StoreAll() []Doc {
	docs, err := client.Scroll()
	if err != nil {
		fmt.Printf("StoreAll error: %v\n", err)
		return nil
	}
	return docs
}

// InitStore initializes the Qdrant client and creates the collection if needed.
// Must be called after godotenv.Load().
func InitStore() error {
	client = newQdrantClient()
	return client.EnsureCollection()
}
