package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	pgvector "github.com/pgvector/pgvector-go"
	"github.com/vsayfb/gig-platform-categorization-service/internal/config"
)

type HuggingFaceClient struct {
	client *http.Client
	cfg    *config.Config
}

func NewHuggingFaceClient(cfg *config.Config) *HuggingFaceClient {
	return &HuggingFaceClient{
		client: &http.Client{Timeout: 15 * time.Second},
		cfg:    cfg,
	}
}

func (c *HuggingFaceClient) Embed(ctx context.Context, text string) (pgvector.Vector, error) {
	body, _ := json.Marshal(map[string]any{
		"inputs": text,
	})

	url := c.cfg.AI_API_ENDPOINT + c.cfg.HUGGINGFACE_AI_MODEL

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))

	if err != nil {
		return pgvector.Vector{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.HuggingFaceAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)

	if err != nil {
		return pgvector.Vector{}, err
	}

	defer resp.Body.Close()

	// HF returns [][]float32 for sentence-transformers (mean pooling)
	var raw [][]float32

	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return pgvector.Vector{}, fmt.Errorf("decode embedding response: %w", err)
	}

	if len(raw) == 0 {
		return pgvector.Vector{}, fmt.Errorf("empty embedding response")
	}

	return pgvector.NewVector(raw[0]), nil
}
