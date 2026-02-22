package group

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// LFSClient wraps the KafScale LFS Proxy HTTP API for producing messages to Kafka.
type LFSClient struct {
	baseURL    string // validated http(s) URL, no trailing slash
	apiKey     string
	httpClient *http.Client
}

// NewLFSClient creates a new LFS proxy client.
// The baseURL is parsed and validated at construction time; only http and https
// schemes are accepted.
func NewLFSClient(baseURL, apiKey string) *LFSClient {
	u, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return &LFSClient{
			baseURL: "http://localhost:0",
			apiKey:  apiKey,
			httpClient: &http.Client{
				Timeout: 30 * time.Second,
			},
		}
	}
	// Reconstruct from parsed parts to normalise.
	safe := u.Scheme + "://" + u.Host
	if u.Path != "" {
		safe += u.Path
	}
	return &LFSClient{
		baseURL: safe,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// LFSEnvelope is the response from the LFS proxy after a successful produce.
type LFSEnvelope struct {
	KfsLFS      int    `json:"kfs_lfs"`
	Bucket      string `json:"bucket"`
	Key         string `json:"key"`
	Size        int64  `json:"size"`
	SHA256      string `json:"sha256"`
	Checksum    string `json:"checksum"`
	ChecksumAlg string `json:"checksum_alg"`
	ContentType string `json:"content_type"`
	CreatedAt   string `json:"created_at"`
	ProxyID     string `json:"proxy_id"`
}

// Produce sends a message to the LFS proxy which produces it to the given Kafka topic.
func (c *LFSClient) Produce(ctx context.Context, topic string, requestID string, payload []byte) (*LFSEnvelope, error) {
	endpoint := c.baseURL + "/lfs/produce"
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		return nil, fmt.Errorf("lfs produce: invalid URL scheme")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("lfs produce: create request: %w", err)
	}

	req.Header.Set("X-Kafka-Topic", topic)
	req.Header.Set("Content-Type", "application/json")
	if requestID != "" {
		req.Header.Set("X-Request-ID", requestID)
	}
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("lfs produce: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("lfs produce: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lfs produce: status %d: %s", resp.StatusCode, string(body))
	}

	var envelope LFSEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("lfs produce: decode response: %w", err)
	}

	return &envelope, nil
}

// ProduceEnvelope marshals a GroupEnvelope and produces it to the given topic.
func (c *LFSClient) ProduceEnvelope(ctx context.Context, topic string, env *GroupEnvelope) error {
	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("lfs produce envelope: marshal: %w", err)
	}
	_, err = c.Produce(ctx, topic, env.CorrelationID, data)
	return err
}

// Healthy checks if the LFS proxy is reachable.
func (c *LFSClient) Healthy(ctx context.Context) bool {
	endpoint := c.baseURL + "/lfs/produce"
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		return false
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	// The proxy returns 400 for GET (method not allowed or missing topic), but that means it's up.
	return resp.StatusCode < 500
}
