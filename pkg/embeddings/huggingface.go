package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/vsayfb/gig-platform-categorization-service/internal/config"
)

type HuggingFaceClient struct {
	apiKey string
	client *http.Client
	cfg    *config.Config
}

func NewHuggingFaceClient(cfg *config.Config) *HuggingFaceClient {
	return &HuggingFaceClient{
		apiKey: cfg.HuggingFaceAPIKey,
		client: &http.Client{Timeout: 15 * time.Second},
		cfg:    cfg,
	}
}

func (c *HuggingFaceClient) Embed(ctx context.Context, text string) ([]float32, error) {
	body, _ := json.Marshal(map[string]any{
		"inputs": text,
	})

	url := c.cfg.HUGGINGFACE_ENDPOINT + "/" + c.cfg.HUGGINGFACE_AI_MODEL

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))

	if err != nil {
		return nil, err

	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var raw [][]float32

	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode embedding response: %w", err)
	}

	if len(raw) == 0 {
		return nil, fmt.Errorf("empty embedding response")
	}

	return raw[0], nil
}
