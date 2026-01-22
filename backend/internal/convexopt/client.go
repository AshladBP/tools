package convexopt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is an HTTP client for the Python Convex Optimizer service.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new Convex Optimizer client.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute, // Long timeout for optimization
		},
	}
}

// Optimize sends an optimization request to the Python service.
func (c *Client) Optimize(req *ConvexOptimizeRequest) (*ConvexOptimizeResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.baseURL+"/api/convex/optimize",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Try to parse error message
		var errResp struct {
			Detail string `json:"detail"`
		}
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Detail != "" {
			return nil, fmt.Errorf("server error (%d): %s", resp.StatusCode, errResp.Detail)
		}
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result ConvexOptimizeResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// Health checks if the Python service is available.
func (c *Client) Health() (*HealthResponse, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/convex/health")
	if err != nil {
		return nil, fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("service unhealthy: %d", resp.StatusCode)
	}

	var result HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode health response: %w", err)
	}

	return &result, nil
}

// Validate validates the configuration without running optimization.
func (c *Client) Validate(req *ConvexOptimizeRequest) (bool, []string, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return false, nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.baseURL+"/api/convex/validate",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return false, nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Valid  bool     `json:"valid"`
		Errors []string `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, nil, fmt.Errorf("decode response: %w", err)
	}

	return result.Valid, result.Errors, nil
}
