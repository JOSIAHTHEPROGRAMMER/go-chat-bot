# AI Portfolio Chatbot — Go Backend

![Go](https://img.shields.io/badge/Go-1.25.4-00ADD8?style=flat&logo=go&logoColor=white)
![Render](https://img.shields.io/badge/Deployed%20on-Render-46E3B7?style=flat&logo=render&logoColor=white)
![Qdrant](https://img.shields.io/badge/Vector%20DB-Qdrant-FF4081?style=flat)
![MongoDB](https://img.shields.io/badge/Database-MongoDB-47A248?style=flat&logo=mongodb&logoColor=white)
![Groq](https://img.shields.io/badge/LLM-Groq-F55036?style=flat)
![Gemini](https://img.shields.io/badge/LLM-Gemini-4285F4?style=flat&logo=google&logoColor=white)
![License](https://img.shields.io/badge/License-MIT-blue?style=flat)

A production-grade AI backend written in Go that powers a portfolio chatbot. The system fetches GitHub README data and language statistics, embeds them into a vector database, and uses a multi-step planning layer to route user questions to the most appropriate handler — direct answer, semantic search, or a purpose-built tool — before calling an LLM to generate a grounded response.

---

## Architecture

```
Request
  └── Middleware (CORS, Auth, Rate Limit, Logging)
        └── Planner
              ├── Direct       -- greetings and small talk
              ├── RAG          -- semantic search via Qdrant
              └── Tool
                    ├── get_project       -- fetch a named project by path
                    └── filter_by_tech    -- match projects by GitHub language stats
                          └── LLM (Groq / Gemini)
                                └── Response + Session saved to MongoDB
```

---

## Project Structure

```
├── config/
│   ├── model.go          # Active LLM model state
│   └── prompt.go         # System prompt (single source of truth)
├── fetcher/
│   └── github.go         # Fetches READMEs + language stats, handles pagination and blocklist
├── llm/
│   ├── provider.go       # Provider interface (Complete, Chat, Stream)
│   ├── embedder.go       # Embedder interface
│   ├── registry.go       # Provider registry
│   ├── groq.go           # Groq implementation
│   ├── gemini.go         # Gemini implementation
│   └── gemini_embed.go   # Gemini embedding implementation
├── logger/
│   └── logger.go         # Request-scoped observability via context
├── middleware/
│   └── middleware.go     # CORS, Auth, Rate limiting, Logging, Recovery
├── planner/
│   └── planner.go        # Decision layer — routes questions to the right handler
├── rag/
│   ├── embed.go          # Embeds READMEs and upserts into Qdrant
│   ├── qdrant.go         # Qdrant REST client
│   ├── search.go         # Vector similarity search
│   └── store.go          # Store interface (init, set, scroll)
├── routes/
│   ├── chat.go           # POST /chat
│   ├── stream.go         # POST /stream (Server-Sent Events)
│   └── health.go         # GET /health
├── session/
│   └── session.go        # MongoDB conversation history
├── tools/
│   ├── tool.go           # Tool interface and registry
│   ├── get_project.go    # Lookup a project by name
│   └── filter_by_tech.go # Filter projects by GitHub language stats with README fallback
├── go.mod
├── go.sum
└── main.go
```

---

## How It Works

1. On startup the server paginates through all public GitHub repos, skips forks and a configured blocklist, fetches each README, and retrieves the GitHub language breakdown per repo (e.g. `{"JavaScript": 45123, "CSS": 12400}`).
2. Each README is embedded using Gemini `text-embedding-004` (3072 dimensions) with a rate-limit-safe delay between calls, then upserted into Qdrant with language data stored in the payload.
3. Every 6 hours the system refreshes embeddings and alternates between Groq and Gemini as the active LLM provider.
4. When a question arrives the **Planner** classifies intent and routes accordingly:
   - **Direct** — greetings and small talk, no context needed
   - **RAG** — query is embedded, top-k similar documents retrieved from Qdrant and injected into the prompt
   - **Tool: get_project** — a specific project name is detected, the doc is fetched directly from Qdrant by path
   - **Tool: filter_by_tech** — a language or technology is detected, all docs are scanned using GitHub language stats as the primary source and README keyword matching as fallback
5. The prompt is assembled with the system persona, conversation history, retrieved context, and the question.
6. The response is streamed token by token via SSE or returned in full, and the turn is persisted to MongoDB under the session ID.

---

## API

### `POST /chat`

Returns the full response once the LLM finishes.

**Request:**

```json
{
  "question": "What projects use TypeScript?",
  "session_id": ""
}
```

**Response:**

```json
{
  "answer": "The following projects use TypeScript...",
  "plan_type": "tool",
  "session_id": "64f3a2c1e4b0a1d2f3e4b5c6"
}
```

Send `session_id` back on every subsequent request to maintain conversation history.

---

### `POST /stream`

Same as `/chat` but returns tokens as Server-Sent Events.

**Events:**

- `event: session` — fired first, contains the session ID
- `data: <token>` — one event per token as they arrive
- `event: error` — fired if the LLM stream fails
- `event: done` — fired when the stream is complete

**Frontend example:**

```javascript
const res = await fetch("/stream", {
  method: "POST",
  headers: { "Content-Type": "application/json", "X-API-Key": API_KEY },
  body: JSON.stringify({ question, session_id: sessionId }),
});
const reader = res.body.getReader();
// read tokens and append to UI
```

---

### `GET /health`

Unauthenticated. Used by Render for uptime monitoring.

**Response:**

```json
{ "status": "ok", "time": "2025-01-01T00:00:00Z" }
```

---

## Environment Variables

| Variable            | Required   | Description                                              |
| ------------------- | ---------- | -------------------------------------------------------- |
| `GROQ_API_KEY`      | Yes        | Groq API key                                             |
| `GROQ_MODEL`        | Yes        | Groq model name e.g. `llama-3.1-8b-instant`              |
| `GEMINI_API_KEY`    | Yes        | Google Gemini API key                                    |
| `GEMINI_MODEL`      | Yes        | Gemini model name e.g. `gemini-1.5-flash`                |
| `GITHUB_USERNAME`   | Yes        | GitHub username to fetch READMEs from                    |
| `GITHUB_TOKEN`      | Yes        | GitHub personal access token                             |
| `QDRANT_URL`        | Yes        | Qdrant cluster URL                                       |
| `QDRANT_COLLECTION` | Yes        | Qdrant collection name e.g. `Go-bot`                     |
| `QDRANT_API_KEY`    | Cloud only | Qdrant Cloud API key                                     |
| `MONGODB_URI`       | Yes        | MongoDB Atlas connection string                          |
| `MONGODB_DB`        | Yes        | MongoDB database name e.g. `portfolio`                   |
| `ALLOWED_ORIGIN`    | Yes        | Frontend URL for CORS e.g. `https://your-app.vercel.app` |
| `API_KEY`           | Yes        | Secret key sent by the frontend in `X-API-Key` header    |
| `RATE_LIMIT`        | No         | Max requests per minute per IP (default: `10`)           |
| `PORT`              | No         | Server port — injected automatically by Render           |

---

## Running Locally

```bash
# Clone the repo
git clone https://github.com/JOSIAHTHEPROGRAMMER/go-chat-bot
cd go-chat-bot

# Copy and fill in your environment variables
cp .env.example .env

# Install dependencies
go mod tidy

# Run
go run main.go
```

> **Note:** On first run the server embeds all repos into Qdrant. With 40+ repos and a 4-second rate-limit delay between Gemini calls, this takes approximately 3 minutes. Subsequent restarts re-embed idempotently.

---

## Deployment

Deployed on **Render** as a Web Service.

| Setting           | Value                  |
| ----------------- | ---------------------- |
| Environment       | Go                     |
| Build command     | `go build -o server .` |
| Start command     | `./server`             |
| Health check path | `/health`              |

Set all environment variables in the Render dashboard under **Environment**.

---

## Tech Stack

| Layer                | Technology                              |
| -------------------- | --------------------------------------- |
| Language             | Go 1.25.4                               |
| LLM providers        | Groq, Google Gemini                     |
| Embeddings           | Gemini `text-embedding-004` (3072 dims) |
| Vector database      | Qdrant Cloud                            |
| Conversation history | MongoDB Atlas                           |
| Hosting              | Render                                  |
