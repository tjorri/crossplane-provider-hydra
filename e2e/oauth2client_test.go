package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	hydra "github.com/ory/hydra-client-go/v2"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	hydraclient "github.com/tjorri/crossplane-provider-hydra/internal/clients"
)

// setupHydra starts an Ory Hydra container for testing and returns the admin URL and a cleanup function.
func setupHydra(ctx context.Context, t *testing.T) (adminURL string, cleanup func()) {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        "oryd/hydra:v2.2.0",
		ExposedPorts: []string{"4445/tcp"},
		Cmd:          []string{"serve", "all", "--dev"},
		Env: map[string]string{
			"DSN":              "memory",
			"URLS_SELF_ISSUER": "http://localhost:4444",
			"URLS_LOGIN":       "http://localhost:3000/login",
			"URLS_CONSENT":     "http://localhost:3000/consent",
			"SECRETS_SYSTEM":   "a]3M.I4;rcoO+e3nMgBi7xGBGotchgiK",
		},
		WaitingFor: wait.ForHTTP("/health/alive").
			WithPort("4445/tcp").
			WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start Hydra container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get container host: %v", err)
	}

	port, err := container.MappedPort(ctx, "4445")
	if err != nil {
		t.Fatalf("failed to get mapped port: %v", err)
	}

	adminURL = fmt.Sprintf("http://%s:%s", host, port.Port())

	cleanup = func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	}

	return adminURL, cleanup
}

// newDirectHydraClient creates a raw Hydra SDK client for test verification,
// bypassing the provider's client wrapper.
func newDirectHydraClient(adminURL string) *hydra.APIClient {
	cfg := hydra.NewConfiguration()
	cfg.Servers = hydra.ServerConfigurations{
		{URL: adminURL},
	}
	return hydra.NewAPIClient(cfg)
}

func TestOAuth2ClientCRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	ctx := context.Background()
	adminURL, cleanup := setupHydra(ctx, t)
	defer cleanup()

	client := hydraclient.NewHydraClient(adminURL, "")

	t.Run("Create", func(t *testing.T) {
		spec := hydra.OAuth2Client{
			ClientId:                hydra.PtrString("test-client-create"),
			ClientName:              hydra.PtrString("Test Client"),
			GrantTypes:              []string{"client_credentials"},
			ResponseTypes:           []string{"token"},
			Scope:                   hydra.PtrString("openid"),
			TokenEndpointAuthMethod: hydra.PtrString("client_secret_basic"),
		}

		created, err := client.CreateOAuth2Client(ctx, spec)
		if err != nil {
			t.Fatalf("Create() failed: %v", err)
		}
		if created.GetClientId() != "test-client-create" {
			t.Errorf("expected client_id %q, got %q", "test-client-create", created.GetClientId())
		}
		if created.GetClientName() != "Test Client" {
			t.Errorf("expected client_name %q, got %q", "Test Client", created.GetClientName())
		}
		// Secret should be returned on creation
		if created.ClientSecret == nil || *created.ClientSecret == "" {
			t.Error("expected client_secret to be set on creation")
		}
	})

	t.Run("Observe_Exists", func(t *testing.T) {
		observed, err := client.GetOAuth2Client(ctx, "test-client-create")
		if err != nil {
			t.Fatalf("GetOAuth2Client() failed: %v", err)
		}
		if observed == nil {
			t.Fatal("expected client to exist, got nil")
		}
		if observed.GetClientId() != "test-client-create" {
			t.Errorf("expected client_id %q, got %q", "test-client-create", observed.GetClientId())
		}
	})

	t.Run("Update", func(t *testing.T) {
		spec := hydra.OAuth2Client{
			ClientId:                hydra.PtrString("test-client-create"),
			ClientName:              hydra.PtrString("Updated Client"),
			GrantTypes:              []string{"client_credentials", "authorization_code"},
			ResponseTypes:           []string{"token", "code"},
			Scope:                   hydra.PtrString("openid profile"),
			TokenEndpointAuthMethod: hydra.PtrString("client_secret_basic"),
			RedirectUris:            []string{"http://localhost:8080/callback"},
		}

		updated, err := client.UpdateOAuth2Client(ctx, "test-client-create", spec)
		if err != nil {
			t.Fatalf("UpdateOAuth2Client() failed: %v", err)
		}
		if updated.GetClientName() != "Updated Client" {
			t.Errorf("expected client_name %q, got %q", "Updated Client", updated.GetClientName())
		}
		if updated.GetScope() != "openid profile" {
			t.Errorf("expected scope %q, got %q", "openid profile", updated.GetScope())
		}
	})

	t.Run("Observe_Updated", func(t *testing.T) {
		observed, err := client.GetOAuth2Client(ctx, "test-client-create")
		if err != nil {
			t.Fatalf("GetOAuth2Client() failed: %v", err)
		}
		if observed == nil {
			t.Fatal("expected client to exist")
		}
		if observed.GetClientName() != "Updated Client" {
			t.Errorf("expected client_name %q, got %q", "Updated Client", observed.GetClientName())
		}
	})

	t.Run("Delete", func(t *testing.T) {
		err := client.DeleteOAuth2Client(ctx, "test-client-create")
		if err != nil {
			t.Fatalf("DeleteOAuth2Client() failed: %v", err)
		}
	})

	t.Run("Observe_Deleted", func(t *testing.T) {
		observed, err := client.GetOAuth2Client(ctx, "test-client-create")
		if err != nil {
			t.Fatalf("GetOAuth2Client() failed: %v", err)
		}
		if observed != nil {
			t.Error("expected client to not exist after deletion")
		}
	})
}

func TestOAuth2ClientDriftDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	ctx := context.Background()
	adminURL, cleanup := setupHydra(ctx, t)
	defer cleanup()

	client := hydraclient.NewHydraClient(adminURL, "")
	directClient := newDirectHydraClient(adminURL)

	// Create a client via our wrapper.
	spec := hydra.OAuth2Client{
		ClientId:                hydra.PtrString("test-drift"),
		ClientName:              hydra.PtrString("Original Name"),
		GrantTypes:              []string{"client_credentials"},
		ResponseTypes:           []string{"token"},
		Scope:                   hydra.PtrString("openid"),
		TokenEndpointAuthMethod: hydra.PtrString("client_secret_basic"),
	}
	_, err := client.CreateOAuth2Client(ctx, spec)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	t.Run("ExternalModification_Detected", func(t *testing.T) {
		// Modify the client directly via Hydra admin API (simulating external change).
		drifted := hydra.OAuth2Client{
			ClientId:                hydra.PtrString("test-drift"),
			ClientName:              hydra.PtrString("Externally Modified"),
			GrantTypes:              []string{"client_credentials"},
			ResponseTypes:           []string{"token"},
			Scope:                   hydra.PtrString("openid offline"),
			TokenEndpointAuthMethod: hydra.PtrString("client_secret_basic"),
		}
		_, _, err := directClient.OAuth2API.SetOAuth2Client(ctx, "test-drift").
			OAuth2Client(drifted).Execute()
		if err != nil {
			t.Fatalf("external modification failed: %v", err)
		}

		// Observe via our wrapper — should see the drifted state.
		observed, err := client.GetOAuth2Client(ctx, "test-drift")
		if err != nil {
			t.Fatalf("GetOAuth2Client() failed: %v", err)
		}
		if observed.GetClientName() != "Externally Modified" {
			t.Errorf("expected externally modified name, got %q", observed.GetClientName())
		}
		if observed.GetScope() != "openid offline" {
			t.Errorf("expected drifted scope %q, got %q", "openid offline", observed.GetScope())
		}
	})

	t.Run("DriftCorrection", func(t *testing.T) {
		// Restore the original spec via update (simulating reconciliation).
		spec.ClientName = hydra.PtrString("Original Name")
		spec.Scope = hydra.PtrString("openid")
		_, err := client.UpdateOAuth2Client(ctx, "test-drift", spec)
		if err != nil {
			t.Fatalf("UpdateOAuth2Client() failed: %v", err)
		}

		// Verify it's back to the desired state.
		observed, err := client.GetOAuth2Client(ctx, "test-drift")
		if err != nil {
			t.Fatalf("GetOAuth2Client() failed: %v", err)
		}
		if observed.GetClientName() != "Original Name" {
			t.Errorf("expected restored name %q, got %q", "Original Name", observed.GetClientName())
		}
	})

	t.Run("ServerSetFields_Ignored", func(t *testing.T) {
		// Verify that server-set fields (createdAt, updatedAt) don't cause false drift.
		observed, err := client.GetOAuth2Client(ctx, "test-drift")
		if err != nil {
			t.Fatalf("GetOAuth2Client() failed: %v", err)
		}
		// createdAt and updatedAt should be set by the server.
		if observed.CreatedAt == nil {
			t.Error("expected createdAt to be set by server")
		}
		if observed.UpdatedAt == nil {
			t.Error("expected updatedAt to be set by server")
		}
	})

	// Cleanup.
	_ = client.DeleteOAuth2Client(ctx, "test-drift")
}

func TestOAuth2ClientEdgeCases(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	ctx := context.Background()
	adminURL, cleanup := setupHydra(ctx, t)
	defer cleanup()

	client := hydraclient.NewHydraClient(adminURL, "")

	t.Run("AutoGeneratedClientID", func(t *testing.T) {
		// Create without specifying client_id — Hydra should generate one.
		spec := hydra.OAuth2Client{
			ClientName:              hydra.PtrString("Auto ID Client"),
			GrantTypes:              []string{"client_credentials"},
			ResponseTypes:           []string{"token"},
			TokenEndpointAuthMethod: hydra.PtrString("client_secret_basic"),
		}

		created, err := client.CreateOAuth2Client(ctx, spec)
		if err != nil {
			t.Fatalf("Create() failed: %v", err)
		}
		if created.GetClientId() == "" {
			t.Error("expected auto-generated client_id, got empty string")
		}
		t.Logf("auto-generated client_id: %s", created.GetClientId())

		// Cleanup.
		_ = client.DeleteOAuth2Client(ctx, created.GetClientId())
	})

	t.Run("UserProvidedSecret", func(t *testing.T) {
		spec := hydra.OAuth2Client{
			ClientId:                hydra.PtrString("test-user-secret"),
			ClientName:              hydra.PtrString("User Secret Client"),
			ClientSecret:            hydra.PtrString("my-custom-secret-value"),
			GrantTypes:              []string{"client_credentials"},
			ResponseTypes:           []string{"token"},
			TokenEndpointAuthMethod: hydra.PtrString("client_secret_basic"),
		}

		created, err := client.CreateOAuth2Client(ctx, spec)
		if err != nil {
			t.Fatalf("Create() failed: %v", err)
		}
		// Hydra should accept and use the provided secret.
		if created.GetClientId() != "test-user-secret" {
			t.Errorf("expected client_id %q, got %q", "test-user-secret", created.GetClientId())
		}

		// Cleanup.
		_ = client.DeleteOAuth2Client(ctx, "test-user-secret")
	})

	t.Run("DeleteNonExistent_Idempotent", func(t *testing.T) {
		err := client.DeleteOAuth2Client(ctx, "non-existent-client-id")
		if err != nil {
			t.Errorf("expected idempotent delete of non-existent client, got error: %v", err)
		}
	})

	t.Run("ConnectionDetails", func(t *testing.T) {
		spec := hydra.OAuth2Client{
			ClientId:                hydra.PtrString("test-conn-details"),
			ClientName:              hydra.PtrString("Connection Details Client"),
			GrantTypes:              []string{"client_credentials"},
			ResponseTypes:           []string{"token"},
			TokenEndpointAuthMethod: hydra.PtrString("client_secret_basic"),
		}

		created, err := client.CreateOAuth2Client(ctx, spec)
		if err != nil {
			t.Fatalf("Create() failed: %v", err)
		}

		// Verify the client_id is returned.
		if created.GetClientId() != "test-conn-details" {
			t.Errorf("expected client_id in response")
		}
		// Verify client_secret is returned on creation.
		if created.ClientSecret == nil || *created.ClientSecret == "" {
			t.Error("expected client_secret to be returned on creation")
		}

		// Cleanup.
		_ = client.DeleteOAuth2Client(ctx, "test-conn-details")
	})
}

