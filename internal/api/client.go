package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/auth"
	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/models"
)

// Client is the authenticated HTTP client used by the agent.
type Client struct {
	baseURL string
	auth    *auth.TokenAuth
	client  *http.Client
}

// NewClient returns an API client for the given core URL and auth bundle.
func NewClient(baseURL string, authBundle *auth.TokenAuth) *Client {
	return &Client{
		baseURL: baseURL,
		auth:    authBundle,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// SetAuth updates the auth bundle used by the client.
func (c *Client) SetAuth(authBundle *auth.TokenAuth) {
	c.auth = authBundle
}

func (c *Client) request(method, path string, body io.Reader) (*http.Request, error) {
	u, err := url.JoinPath(c.baseURL, path)
	if err != nil {
		return nil, fmt.Errorf("join url: %w", err)
	}

	req, err := http.NewRequest(method, u, body)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	if c.auth != nil {
		authorization, timestamp, signature, err := c.auth.SignRequest()
		if err != nil {
			return nil, fmt.Errorf("sign request: %w", err)
		}
		req.Header.Set("Authorization", authorization)
		req.Header.Set("X-Agent-Timestamp", timestamp)
		req.Header.Set("X-Agent-Signature", signature)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "emir-agent/"+models.Version)

	return req, nil
}

func (c *Client) do(ctx context.Context, req *http.Request, out any) (*http.Response, error) {
	req = req.WithContext(ctx)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return resp, fmt.Errorf("http error %d: %s", resp.StatusCode, string(body))
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return resp, fmt.Errorf("decode response: %w", err)
		}
	}

	return resp, nil
}

// Pair registers the agent with the backend.
func (c *Client) Pair(ctx context.Context, req models.AgentPairRequest) (*models.AgentPairResponse, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal pair request: %w", err)
	}

	httpReq, err := c.request(http.MethodPost, "/api/agent/pair", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	var resp models.AgentPairResponse
	if _, err := c.do(ctx, httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Heartbeat sends a keep-alive to the backend.
func (c *Client) Heartbeat(ctx context.Context) (*models.AgentHeartbeatResponse, error) {
	httpReq, err := c.request(http.MethodPost, "/api/agent/heartbeat", nil)
	if err != nil {
		return nil, err
	}

	var resp models.AgentHeartbeatResponse
	if _, err := c.do(ctx, httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Inventory sends the full hardware/software inventory.
func (c *Client) Inventory(ctx context.Context, req models.AgentInventoryRequest) error {
	payload, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal inventory request: %w", err)
	}

	httpReq, err := c.request(http.MethodPost, "/api/agent/inventory", bytes.NewReader(payload))
	if err != nil {
		return err
	}

	_, err = c.do(ctx, httpReq, nil)
	return err
}

// Version checks the latest agent release.
func (c *Client) Version(ctx context.Context) (*models.AgentVersionResponse, error) {
	httpReq, err := c.request(http.MethodGet, "/api/agent/version", nil)
	if err != nil {
		return nil, err
	}

	var resp models.AgentVersionResponse
	if _, err := c.do(ctx, httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
