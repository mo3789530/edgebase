package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/edgebase/cluster-agent/internal/config"
	"github.com/edgebase/cluster-agent/internal/model"
	"github.com/google/uuid"
)

var ErrInventoryEndpointUnavailable = errors.New("inventory endpoint unavailable")

type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
	paths      config.EndpointPaths
}

func New(cfg config.Config) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: cfg.RequestTimeout},
		baseURL:    cfg.ControlPlaneBaseURL,
		token:      cfg.Token,
		paths:      cfg.Paths,
	}
}

func (c *Client) ReportHeartbeat(ctx context.Context, clusterID uuid.UUID, heartbeat model.Heartbeat) error {
	return c.postJSON(ctx, c.path(c.paths.Heartbeat, clusterID), heartbeat, nil)
}

func (c *Client) ReportInventory(ctx context.Context, clusterID uuid.UUID, inventory model.ClusterInventory) error {
	err := c.postJSON(ctx, c.path(c.paths.Inventory, clusterID), inventory, nil)
	if err == nil {
		return nil
	}
	if isHTTPStatus(err, http.StatusNotFound) || isHTTPStatus(err, http.StatusMethodNotAllowed) {
		return fmt.Errorf("%w: %v", ErrInventoryEndpointUnavailable, err)
	}
	return err
}

func (c *Client) FetchSyncPlan(ctx context.Context, clusterID uuid.UUID) (*model.SyncPlan, error) {
	var plan model.SyncPlan
	if err := c.getJSON(ctx, c.path(c.paths.Sync, clusterID), &plan); err != nil {
		return nil, err
	}
	return &plan, nil
}

func (c *Client) FetchGatewayRoutes(ctx context.Context, clusterID uuid.UUID) ([]model.GatewayRoute, error) {
	var routes []model.GatewayRoute
	if err := c.getJSON(ctx, c.path(c.paths.Gateway, clusterID), &routes); err != nil {
		return nil, err
	}
	return routes, nil
}

func (c *Client) ReportSyncAck(ctx context.Context, clusterID uuid.UUID, ack model.SyncAck) error {
	req := map[string]any{
		"sync_id": ack.SyncID,
		"result": map[string]any{
			"success":       ack.Success,
			"error_message": summarizeAckErrors(ack),
		},
	}
	return c.postJSON(ctx, c.path(c.paths.Ack, clusterID), req, nil)
}

func (c *Client) postJSON(ctx context.Context, path string, in any, out any) error {
	body, err := json.Marshal(in)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return readHTTPError(resp)
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

func (c *Client) getJSON(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return readHTTPError(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func (c *Client) path(template string, clusterID uuid.UUID) string {
	if strings.Contains(template, "%s") {
		return fmt.Sprintf(template, clusterID.String())
	}
	return template
}

type httpStatusError struct {
	status int
	body   string
}

func (e httpStatusError) Error() string {
	if e.body == "" {
		return fmt.Sprintf("http status %d", e.status)
	}
	return fmt.Sprintf("http status %d: %s", e.status, e.body)
}

func isHTTPStatus(err error, status int) bool {
	var httpErr httpStatusError
	return errors.As(err, &httpErr) && httpErr.status == status
}

func readHTTPError(resp *http.Response) error {
	const maxBody = 1024
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	return httpStatusError{status: resp.StatusCode, body: strings.TrimSpace(string(body))}
}

func summarizeAckErrors(ack model.SyncAck) string {
	if ack.Success {
		return ""
	}

	parts := make([]string, 0, len(ack.Results))
	for _, result := range ack.Results {
		if result.ErrorMessage == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s/%s: %s", result.ResourceType, result.ResourceName, result.ErrorMessage))
	}

	if len(parts) == 0 {
		return "apply failed"
	}

	msg := strings.Join(parts, "; ")
	if len(msg) > 1024 {
		return msg[:1024]
	}
	return msg
}

// Ensure the generated heartbeat always has a timestamp when omitted by caller.
func FillHeartbeatObservedAt(h model.Heartbeat) model.Heartbeat {
	if h.ObservedAt.IsZero() {
		h.ObservedAt = time.Now().UTC()
	}
	return h
}
