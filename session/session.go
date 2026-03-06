package session

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/JOSIAHTHEPROGRAMMER/portfolio-backend/llm"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var col *mongo.Collection

// Session represents a single conversation stored in MongoDB.
type Session struct {
	ID        primitive.ObjectID `bson:"_id"`
	History   []llm.Message      `bson:"history"`
	UpdatedAt time.Time          `bson:"updated_at"`
}

// Connect initializes the MongoDB connection. Call once at startup.
func Connect() error {
	uri := os.Getenv("MONGODB_URI")
	db := os.Getenv("MONGODB_DB")

	if uri == "" || db == "" {
		return fmt.Errorf("missing MONGODB_URI or MONGODB_DB in env")
	}

	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		return err
	}

	// Verify the connection is live
	if err := client.Ping(context.Background(), nil); err != nil {
		return fmt.Errorf("MongoDB ping failed: %w", err)
	}

	col = client.Database(db).Collection("sessions")
	return nil
}

// New creates a new session with an empty history and returns its ID.
func New(ctx context.Context) (string, error) {
	doc := Session{
		ID:        primitive.NewObjectID(),
		History:   []llm.Message{},
		UpdatedAt: time.Now(),
	}

	_, err := col.InsertOne(ctx, doc)
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	return doc.ID.Hex(), nil
}

// Load retrieves the conversation history for a given session ID.
// Returns an empty slice if the session does not exist yet.
func Load(ctx context.Context, sessionID string) ([]llm.Message, error) {
	id, err := primitive.ObjectIDFromHex(sessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID: %w", err)
	}

	var s Session
	err = col.FindOne(ctx, bson.M{"_id": id}).Decode(&s)
	if err == mongo.ErrNoDocuments {
		return []llm.Message{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load session: %w", err)
	}

	return s.History, nil
}

// Append adds a user message and assistant reply to the session history.
func Append(ctx context.Context, sessionID, question, answer string) error {
	id, err := primitive.ObjectIDFromHex(sessionID)
	if err != nil {
		return fmt.Errorf("invalid session ID: %w", err)
	}

	newMessages := []llm.Message{
		{Role: "user", Content: question},
		{Role: "assistant", Content: answer},
	}

	update := bson.M{
		"$push": bson.M{"history": bson.M{"$each": newMessages}},
		"$set":  bson.M{"updated_at": time.Now()},
	}

	// Upsert so we do not need a separate New() call if the session was lost
	opts := options.Update().SetUpsert(true)
	_, err = col.UpdateOne(ctx, bson.M{"_id": id}, update, opts)
	if err != nil {
		return fmt.Errorf("failed to append to session: %w", err)
	}

	return nil
}
