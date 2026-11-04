package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxSoroForgeBytes = 4 << 20

// TrackedContract represents a contract entry returned by SoroForge's registry API.
type TrackedContract struct {
	ContractID string `json:"contract_id"`
	Name       string `json:"name,omitempty"`
	Network    string `json:"network"`
	WasmHash   string `json:"wasm_hash,omitempty"`
}

// SoroForgeClient queries tracked contracts from a SoroForge deployment registry.
type SoroForgeClient struct {
	baseURL string
	token   string
	http    *http.Client
}

// NewSoroForgeClient validates baseURL and constructs a read-only SoroForge client.
func NewSoroForgeClient(baseURL, token string, client *http.Client, timeout time.Duration) (*SoroForgeClient, error) {
	u, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("soroforge url %q must be an absolute http:// or https:// URL", baseURL)
	}
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	return &SoroForgeClient{
		baseURL: u.String(),
		token:   strings.TrimSpace(token),
		http:    client,
	}, nil
}

// ListTrackedContracts fetches all tracked contracts for network, following pagination cursors.
func (c *SoroForgeClient) ListTrackedContracts(ctx context.Context, network string) ([]TrackedContract, error) {
	var all []TrackedContract
	cursor := ""
	for page := 0; page < 100; page++ {
		endpoint := c.baseURL + "/api/contracts"
		q := url.Values{}
		if network != "" {
			q.Set("network", network)
		}
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		if enc := q.Encode(); enc != "" {
			endpoint += "?" + enc
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")
		if c.token != "" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}
		resp, err := c.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("soroforge: %w", err)
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxSoroForgeBytes))
		_ = resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("soroforge: reading response: %w", readErr)
		}
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return nil, errors.New("soroforge: authentication failed (check read-only token)")
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("soroforge: unexpected status %s", resp.Status)
		}
		var pageResp struct {
			Contracts  []TrackedContract `json:"contracts"`
			NextCursor string            `json:"next_cursor"`
		}
		if err := json.Unmarshal(body, &pageResp); err != nil {
			return nil, fmt.Errorf("soroforge: decoding response: %w", err)
		}
		all = append(all, pageResp.Contracts...)
		if pageResp.NextCursor == "" {
			break
		}
		cursor = pageResp.NextCursor
	}
	return all, nil
}

// CompareWasmHash checks whether the live on-chain WASM hash matches SoroForge's recorded hash.
func CompareWasmHash(recordedHash, liveHash string) (match bool, detail string) {
	rec := strings.ToLower(strings.TrimSpace(recordedHash))
	live := strings.ToLower(strings.TrimSpace(liveHash))
	if rec == "" {
		return true, "no recorded wasm hash in SoroForge"
	}
	if rec == live {
		return true, fmt.Sprintf("live wasm hash matches SoroForge (%s)", rec)
	}
	return false, fmt.Sprintf("wasm hash drift: SoroForge recorded %s, live network has %s", rec, live)
}
