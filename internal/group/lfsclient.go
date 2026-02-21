package group

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// LFSClient wraps the KafScale LFS Proxy HTTP API for producing messages to Kafka.
type LFSClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewLFSClient creates a new LFS proxy client.
func NewLFSClient(baseURL, apiKey string) *LFSClient {
	return &LFSClient{
		baseURL: strings.TrimRight(baseURL, "/"),
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
	endpoint, err := c.safeURL("/lfs/produce")
	if err != nil {
		return nil, fmt.Errorf("lfs produce: %w", err)
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
	endpoint, err := c.safeURL("/lfs/produce")
	if err != nil {
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

// safeHost matches valid hostname:port patterns.
var safeHost = regexp.MustCompile(`^[a-zA-Z0-9._:-]+$`)

// safeURL parses and validates the base URL, then constructs a safe endpoint.
func (c *LFSClient) safeURL(path string) (string, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("unsupported URL scheme: %s", u.Scheme)
	}
	if !safeHost.MatchString(u.Host) {
		return "", fmt.Errorf("invalid host: %s", u.Host)
	}
	// Reconstruct from validated components.
	return u.Scheme + "://" + u.Host + strings.TrimRight(u.Path, "/") + path, nil
}
