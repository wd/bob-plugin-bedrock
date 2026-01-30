package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/dong/bob-plugin-bedrock/server/bedrock"
	"github.com/dong/bob-plugin-bedrock/server/config"
)

var bedrockClient *bedrock.Client

func main() {
	cfg := config.Load()

	// Initialize Bedrock client
	ctx := context.Background()
	client, err := bedrock.NewClient(ctx, cfg.AWSRegion, cfg.AWSProfile)
	if err != nil {
		log.Fatalf("Failed to create Bedrock client: %v", err)
	}
	bedrockClient = client

	// Setup HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/translate", corsMiddleware(handleTranslate))
	mux.HandleFunc("/health", corsMiddleware(handleHealth))

	addr := "localhost:" + cfg.ServerPort
	log.Printf("Starting server on %s", addr)
	log.Printf("AWS Region: %s, Profile: %s", cfg.AWSRegion, cfg.AWSProfile)

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// corsMiddleware adds CORS headers for local development
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleTranslate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req bedrock.TranslateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.Text == "" {
		http.Error(w, "Missing required field: text", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	if req.Stream {
		handleStreamingTranslate(ctx, w, &req)
	} else {
		handleNonStreamingTranslate(ctx, w, &req)
	}
}

func handleNonStreamingTranslate(ctx context.Context, w http.ResponseWriter, req *bedrock.TranslateRequest) {
	result, err := bedrockClient.Translate(ctx, req)
	if err != nil {
		log.Printf("Translation error: %v", err)
		http.Error(w, fmt.Sprintf("Translation failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"result": result})
}

func handleStreamingTranslate(ctx context.Context, w http.ResponseWriter, req *bedrock.TranslateRequest) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	err := bedrockClient.TranslateStream(ctx, req, func(chunk bedrock.StreamChunk) {
		if chunk.Done {
			fmt.Fprintf(w, "data: [DONE]\n\n")
		} else if chunk.Content != "" {
			data := bedrock.MarshalChunk(chunk)
			fmt.Fprintf(w, "data: %s\n\n", data)
		}
		flusher.Flush()
	})

	if err != nil {
		log.Printf("Streaming translation error: %v", err)
		errChunk := bedrock.StreamChunk{Error: err.Error()}
		data := bedrock.MarshalChunk(errChunk)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}
}
