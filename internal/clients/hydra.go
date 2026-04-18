package clients

import (
	"context"
	"fmt"
	"io"
	"net/http"

	hydra "github.com/ory/hydra-client-go/v2"
)

// HydraClient wraps the Ory Hydra SDK for OAuth2 client management.
type HydraClient struct {
	api *hydra.APIClient
}

// NewHydraClient creates a new Hydra client configured for the given admin URL.
func NewHydraClient(adminURL string, bearerToken string) *HydraClient {
	cfg := hydra.NewConfiguration()
	cfg.Servers = hydra.ServerConfigurations{
		{URL: adminURL},
	}

	if bearerToken != "" {
		cfg.AddDefaultHeader("Authorization", "Bearer "+bearerToken)
	}

	return &HydraClient{
		api: hydra.NewAPIClient(cfg),
	}
}

// GetOAuth2Client retrieves an OAuth2 client by ID.
// Returns nil, nil if the client does not exist.
func (c *HydraClient) GetOAuth2Client(ctx context.Context, id string) (*hydra.OAuth2Client, error) {
	client, resp, err := c.api.OAuth2API.GetOAuth2Client(ctx, id).Execute()
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get OAuth2 client %q: %w", id, err)
	}
	return client, nil
}

// CreateOAuth2Client creates a new OAuth2 client.
func (c *HydraClient) CreateOAuth2Client(ctx context.Context, client hydra.OAuth2Client) (*hydra.OAuth2Client, error) {
	created, resp, err := c.api.OAuth2API.CreateOAuth2Client(ctx).OAuth2Client(client).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to create OAuth2 client: %w: %s", err, readBody(resp))
	}
	return created, nil
}

func readBody(resp *http.Response) string {
	if resp == nil || resp.Body == nil {
		return ""
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	return string(b)
}

// UpdateOAuth2Client replaces an existing OAuth2 client.
func (c *HydraClient) UpdateOAuth2Client(ctx context.Context, id string, client hydra.OAuth2Client) (*hydra.OAuth2Client, error) {
	updated, _, err := c.api.OAuth2API.SetOAuth2Client(ctx, id).OAuth2Client(client).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to update OAuth2 client %q: %w", id, err)
	}
	return updated, nil
}

// DeleteOAuth2Client deletes an OAuth2 client by ID. It is idempotent.
func (c *HydraClient) DeleteOAuth2Client(ctx context.Context, id string) error {
	resp, err := c.api.OAuth2API.DeleteOAuth2Client(ctx, id).Execute()
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil
		}
		return fmt.Errorf("failed to delete OAuth2 client %q: %w", id, err)
	}
	return nil
}
