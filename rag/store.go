package rag

// client is the package-level Qdrant client used across the rag package.
var client = newQdrantClient()

// Set upserts a single doc into Qdrant.
func Set(doc Doc) error {
	return client.Upsert(doc)
}

// StoreAll returns every doc in the collection.
// Used by the tools package for keyword scanning.
func StoreAll() []Doc {
	docs, err := client.Scroll()
	if err != nil {
		return nil
	}
	return docs
}

// InitStore creates the Qdrant collection if it does not already exist.
// Call this once at startup before any reads or writes.

func InitStore() error {
	client = newQdrantClient()
	return client.EnsureCollection()
}
