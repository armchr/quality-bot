package codeapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"quality-bot/src/config"
	"quality-bot/src/util"
)

// Client provides access to CodeAPI endpoints
type Client struct {
	baseURL    string
	httpClient *http.Client
	retryConf  config.RetryConfig
}

// NewClient creates a new CodeAPI client
func NewClient(cfg config.CodeAPIConfig) *Client {
	return &Client{
		baseURL: cfg.URL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		retryConf: cfg.Retry,
	}
}

// ExecuteCypher executes a Cypher query against the code graph
func (c *Client) ExecuteCypher(ctx context.Context, repoName, query string) ([]map[string]any, error) {
	util.Debug("Executing Cypher query for repo: %s", repoName)

	// Replace $repo_name parameter with quoted literal since CodeAPI
	// doesn't support passing parameters separately
	resolvedQuery := strings.ReplaceAll(query, "$repo_name", fmt.Sprintf("'%s'", repoName))

	req := CypherRequest{
		RepoName: repoName,
		Query:    resolvedQuery,
	}

	var resp CypherResponse
	if err := c.post(ctx, "/codeapi/v1/cypher", req, &resp); err != nil {
		util.Error("Cypher query failed: %v", err)
		return nil, err
	}

	util.Debug("Cypher query returned %d results", len(resp.Results))
	return resp.Results, nil
}

// SearchSimilarCode finds semantically similar code
func (c *Client) SearchSimilarCode(ctx context.Context, req SimilarCodeRequest) (*SimilarCodeResponse, error) {
	util.Debug("Searching similar code for function: %s", req.FunctionID)

	var resp SimilarCodeResponse
	if err := c.post(ctx, "/api/v1/searchSimilarCode", req, &resp); err != nil {
		util.Error("Similar code search failed: %v", err)
		return nil, err
	}

	util.Debug("Found %d similar code matches", len(resp.Matches))
	return &resp, nil
}

// GetFunctions retrieves functions from a repository
func (c *Client) GetFunctions(ctx context.Context, repoName string, filePath string) (*FunctionsResponse, error) {
	req := FunctionsRequest{
		RepoName: repoName,
		FilePath: filePath,
	}

	var resp FunctionsResponse
	if err := c.post(ctx, "/codeapi/v1/functions", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetSnippet retrieves a code snippet from a file
func (c *Client) GetSnippet(ctx context.Context, repoName, filePath string, startLine, endLine int) (*SnippetResponse, error) {
	util.Debug("Fetching snippet from %s:%d-%d", filePath, startLine, endLine)

	req := SnippetRequest{
		RepoName:  repoName,
		FilePath:  filePath,
		StartLine: startLine,
		EndLine:   endLine,
	}

	var resp SnippetResponse
	if err := c.post(ctx, "/codeapi/v1/snippet", req, &resp); err != nil {
		util.Debug("Failed to fetch snippet: %v", err)
		return nil, err
	}

	util.Debug("Retrieved %d lines from snippet", resp.TotalLines)
	return &resp, nil
}

func (c *Client) post(ctx context.Context, path string, body any, result any) error {
	var lastErr error

	for attempt := 0; attempt <= c.retryConf.MaxAttempts; attempt++ {
		if attempt > 0 {
			delay := c.calculateBackoff(attempt)
			util.Warn("Retrying request to %s (attempt %d/%d) after %v", path, attempt+1, c.retryConf.MaxAttempts, delay)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		err := c.doPost(ctx, path, body, result)
		if err == nil {
			return nil
		}

		lastErr = err
		if !c.shouldRetry(err) {
			break
		}
	}

	return lastErr
}

func (c *Client) doPost(ctx context.Context, path string, body any, result any) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+path, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return &APIError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	return nil
}

func (c *Client) calculateBackoff(attempt int) time.Duration {
	delay := float64(c.retryConf.InitialDelay)
	for i := 0; i < attempt; i++ {
		delay *= c.retryConf.BackoffFactor
	}
	if delay > float64(c.retryConf.MaxDelay) {
		delay = float64(c.retryConf.MaxDelay)
	}
	return time.Duration(delay)
}

func (c *Client) shouldRetry(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		for _, code := range c.retryConf.RetryOnStatus {
			if apiErr.StatusCode == code {
				return true
			}
		}
	}
	return false
}

// APIError represents an error response from CodeAPI
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("CodeAPI error (status %d): %s", e.StatusCode, e.Body)
}
