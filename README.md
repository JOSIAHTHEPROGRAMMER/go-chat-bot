# AI Portfolio Chatbot (Go)

A backend service written in Go that powers an AI driven chatbot capable of answering questions about my projects using large language models and retrieval-augmented generation (RAG).

The system embeds curated project data, performs semantic search, and uses modern LLMs to generate accurate, context-aware responses.

## Overview

This project serves as the backend for a portfolio chatbot. Instead of relying on static responses, it dynamically retrieves relevant project information and feeds it into an LLM to produce answers grounded in real data.

The goal is to provide an interactive way for users to explore my work through natural language.

## Key Features

- Retrieval augmented generation using embedded project data
- Semantic search with cosine similarity
- Support for multiple LLM providers (Groq and Gemini)
- Modular Go architecture for clarity and extensibility
- Simple HTTP API for frontend integration

## Project Structure
```
.
├── config/    # Environment and application configuration
├── data/      # Embedded project data (processed READMEs, metadata)
├── fetcher/   # Utilities for collecting or updating source data
├── llm/       # LLM clients (Groq, Gemini)
├── rag/       # Embedding and semantic search logic
├── routes/    # HTTP route handlers
└── main.go    # Application entry point
```

## How It Works

1. Project documentation is collected and stored in the `data` directory
2. The data is embedded into vector representations
3. When a user asks a question:
   - The query is embedded
   - Relevant documents are retrieved via semantic search
   - The retrieved context is passed to an LLM
4. The model generates a response grounded in the retrieved data

## API

### POST `/chat`

Send a question to the chatbot and receive a generated response.

**Request:**
```json
{
  "message": "What projects use Go"
}
```

**Response:**
```json
{
  "reply": "The projects that used Go are..."
}
```

## Tech Stack

- Go
- Groq API
- Google Gemini API
- Retrieval-Augmented Generation (RAG)
- Cosine similarity for semantic search

