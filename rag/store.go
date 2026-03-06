package rag

import "sync"

// store is the global in-memory vector store.
// Keyed by repo path so individual docs can be updated without a full reload.
var store = &vectorStore{
	docs: make(map[string]Doc),
}

type vectorStore struct {
	mu   sync.RWMutex
	docs map[string]Doc
}

// Set adds or replaces a single doc in the store.
func (s *vectorStore) Set(doc Doc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.docs[doc.Path] = doc
}

// All returns a snapshot of every doc currently in the store.
func (s *vectorStore) All() []Doc {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Doc, 0, len(s.docs))
	for _, d := range s.docs {
		out = append(out, d)
	}
	return out
}

// Size returns how many docs are currently loaded.
func (s *vectorStore) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.docs)
}

// StoreAll exposes the store's contents to other packages (e.g. tools).
// Returns a snapshot - safe to iterate without holding the lock.
func StoreAll() []Doc {
	return store.All()
}

// LoadFromDisk populates the store from the persisted embeddings JSON.
// Call this at startup so a restart doesn't require re-embedding everything.
func LoadFromDisk() error {
	docs, err := LoadEmbeddings()
	if err != nil {
		return err
	}
	for _, d := range docs {
		store.Set(d)
	}
	return nil
}
