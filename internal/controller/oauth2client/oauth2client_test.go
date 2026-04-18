package oauth2client

import (
	"context"
	"errors"
	"testing"
	"time"

	hydra "github.com/ory/hydra-client-go/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource/fake"

	"github.com/tjorri/crossplane-provider-hydra/apis/oauth2client/v1alpha1"
)

// mockHydra implements hydraClient for testing.
type mockHydra struct {
	getFn    func(ctx context.Context, id string) (*hydra.OAuth2Client, error)
	createFn func(ctx context.Context, client hydra.OAuth2Client) (*hydra.OAuth2Client, error)
	updateFn func(ctx context.Context, id string, client hydra.OAuth2Client) (*hydra.OAuth2Client, error)
	deleteFn func(ctx context.Context, id string) error
}

func (m *mockHydra) GetOAuth2Client(ctx context.Context, id string) (*hydra.OAuth2Client, error) {
	return m.getFn(ctx, id)
}

func (m *mockHydra) CreateOAuth2Client(ctx context.Context, client hydra.OAuth2Client) (*hydra.OAuth2Client, error) {
	return m.createFn(ctx, client)
}

func (m *mockHydra) UpdateOAuth2Client(ctx context.Context, id string, client hydra.OAuth2Client) (*hydra.OAuth2Client, error) {
	return m.updateFn(ctx, id, client)
}

func (m *mockHydra) DeleteOAuth2Client(ctx context.Context, id string) error {
	return m.deleteFn(ctx, id)
}

var _ hydraClient = &mockHydra{}

func newTestCR(name, externalName string) *v1alpha1.OAuth2Client {
	cr := &v1alpha1.OAuth2Client{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: v1alpha1.OAuth2ClientSpec{
			ForProvider: v1alpha1.OAuth2ClientParameters{
				ClientID:   ptr("test-client"),
				ClientName: ptr("Test Client"),
				GrantTypes: []string{"client_credentials"},
				Scope:      ptr("openid"),
			},
		},
	}
	if externalName != "" {
		meta.SetExternalName(cr, externalName)
	}
	return cr
}

func ptr[T any](v T) *T { return &v }

func TestSpecToHydra_AllFields(t *testing.T) {
	p := v1alpha1.OAuth2ClientParameters{
		ClientID:                          ptr("my-client"),
		ClientName:                        ptr("My Client"),
		ClientSecret:                      ptr("secret"),
		ClientURI:                         ptr("https://example.com"),
		GrantTypes:                        []string{"authorization_code", "refresh_token"},
		ResponseTypes:                     []string{"code"},
		RedirectURIs:                      []string{"https://example.com/callback"},
		Scope:                             ptr("openid profile"),
		Audience:                          []string{"api"},
		TokenEndpointAuthMethod:           ptr("client_secret_basic"),
		TokenEndpointAuthSigningAlg:       ptr("RS256"),
		Contacts:                          []string{"admin@example.com"},
		LogoURI:                           ptr("https://example.com/logo.png"),
		PolicyURI:                         ptr("https://example.com/policy"),
		TosURI:                            ptr("https://example.com/tos"),
		AllowedCORSOrigins:                []string{"https://example.com"},
		Owner:                             ptr("owner"),
		SubjectType:                       ptr("public"),
		SectorIdentifierURI:               ptr("https://example.com/sector"),
		SkipConsent:                       ptr(true),
		SkipLogoutConsent:                 ptr(false),
		AccessTokenStrategy:               ptr("jwt"),
		BackchannelLogoutURI:              ptr("https://example.com/logout"),
		BackchannelLogoutSessionRequired:  ptr(true),
		FrontchannelLogoutURI:             ptr("https://example.com/frontlogout"),
		FrontchannelLogoutSessionRequired: ptr(true),
		PostLogoutRedirectURIs:            []string{"https://example.com/post-logout"},
		RequestObjectSigningAlg:           ptr("RS256"),
		RequestURIs:                       []string{"https://example.com/request"},
		UserinfoSignedResponseAlg:         ptr("RS256"),
		JwksURI:                           ptr("https://example.com/.well-known/jwks.json"),
		AuthorizationCodeGrantAccessTokenLifespan:  ptr("30m"),
		AuthorizationCodeGrantIDTokenLifespan:      ptr("1h"),
		AuthorizationCodeGrantRefreshTokenLifespan: ptr("24h"),
		ClientCredentialsGrantAccessTokenLifespan:  ptr("10m"),
		ImplicitGrantAccessTokenLifespan:           ptr("15m"),
		ImplicitGrantIDTokenLifespan:               ptr("15m"),
		RefreshTokenGrantAccessTokenLifespan:       ptr("30m"),
		RefreshTokenGrantIDTokenLifespan:           ptr("1h"),
		RefreshTokenGrantRefreshTokenLifespan:      ptr("720h"),
		JwtBearerGrantAccessTokenLifespan:          ptr("5m"),
	}

	c := specToHydra(p)

	checks := []struct {
		name string
		got  any
		want any
	}{
		{"ClientId", c.ClientId, ptr("my-client")},
		{"ClientName", c.ClientName, ptr("My Client")},
		{"ClientSecret", c.ClientSecret, ptr("secret")},
		{"ClientUri", c.ClientUri, ptr("https://example.com")},
		{"GrantTypes", c.GrantTypes, []string{"authorization_code", "refresh_token"}},
		{"ResponseTypes", c.ResponseTypes, []string{"code"}},
		{"RedirectUris", c.RedirectUris, []string{"https://example.com/callback"}},
		{"Scope", c.Scope, ptr("openid profile")},
		{"Audience", c.Audience, []string{"api"}},
		{"TokenEndpointAuthMethod", c.TokenEndpointAuthMethod, ptr("client_secret_basic")},
		{"TokenEndpointAuthSigningAlg", c.TokenEndpointAuthSigningAlg, ptr("RS256")},
		{"Contacts", c.Contacts, []string{"admin@example.com"}},
		{"LogoUri", c.LogoUri, ptr("https://example.com/logo.png")},
		{"PolicyUri", c.PolicyUri, ptr("https://example.com/policy")},
		{"TosUri", c.TosUri, ptr("https://example.com/tos")},
		{"AllowedCorsOrigins", c.AllowedCorsOrigins, []string{"https://example.com"}},
		{"Owner", c.Owner, ptr("owner")},
		{"SubjectType", c.SubjectType, ptr("public")},
		{"SectorIdentifierUri", c.SectorIdentifierUri, ptr("https://example.com/sector")},
		{"SkipConsent", c.SkipConsent, ptr(true)},
		{"SkipLogoutConsent", c.SkipLogoutConsent, ptr(false)},
		{"AccessTokenStrategy", c.AccessTokenStrategy, ptr("jwt")},
		{"BackchannelLogoutUri", c.BackchannelLogoutUri, ptr("https://example.com/logout")},
		{"BackchannelLogoutSessionRequired", c.BackchannelLogoutSessionRequired, ptr(true)},
		{"FrontchannelLogoutUri", c.FrontchannelLogoutUri, ptr("https://example.com/frontlogout")},
		{"FrontchannelLogoutSessionRequired", c.FrontchannelLogoutSessionRequired, ptr(true)},
		{"PostLogoutRedirectUris", c.PostLogoutRedirectUris, []string{"https://example.com/post-logout"}},
		{"RequestObjectSigningAlg", c.RequestObjectSigningAlg, ptr("RS256")},
		{"RequestUris", c.RequestUris, []string{"https://example.com/request"}},
		{"UserinfoSignedResponseAlg", c.UserinfoSignedResponseAlg, ptr("RS256")},
		{"JwksUri", c.JwksUri, ptr("https://example.com/.well-known/jwks.json")},
		{"AuthorizationCodeGrantAccessTokenLifespan", c.AuthorizationCodeGrantAccessTokenLifespan, ptr("30m")},
		{"AuthorizationCodeGrantIdTokenLifespan", c.AuthorizationCodeGrantIdTokenLifespan, ptr("1h")},
		{"AuthorizationCodeGrantRefreshTokenLifespan", c.AuthorizationCodeGrantRefreshTokenLifespan, ptr("24h")},
		{"ClientCredentialsGrantAccessTokenLifespan", c.ClientCredentialsGrantAccessTokenLifespan, ptr("10m")},
		{"ImplicitGrantAccessTokenLifespan", c.ImplicitGrantAccessTokenLifespan, ptr("15m")},
		{"ImplicitGrantIdTokenLifespan", c.ImplicitGrantIdTokenLifespan, ptr("15m")},
		{"RefreshTokenGrantAccessTokenLifespan", c.RefreshTokenGrantAccessTokenLifespan, ptr("30m")},
		{"RefreshTokenGrantIdTokenLifespan", c.RefreshTokenGrantIdTokenLifespan, ptr("1h")},
		{"RefreshTokenGrantRefreshTokenLifespan", c.RefreshTokenGrantRefreshTokenLifespan, ptr("720h")},
		{"JwtBearerGrantAccessTokenLifespan", c.JwtBearerGrantAccessTokenLifespan, ptr("5m")},
	}

	for _, tc := range checks {
		t.Run(tc.name, func(t *testing.T) {
			switch got := tc.got.(type) {
			case *string:
				want := tc.want.(*string)
				if got == nil || want == nil {
					if got != want {
						t.Errorf("got %v, want %v", got, want)
					}
					return
				}
				if *got != *want {
					t.Errorf("got %q, want %q", *got, *want)
				}
			case *bool:
				want := tc.want.(*bool)
				if got == nil || want == nil {
					if got != want {
						t.Errorf("got %v, want %v", got, want)
					}
					return
				}
				if *got != *want {
					t.Errorf("got %v, want %v", *got, *want)
				}
			case []string:
				want := tc.want.([]string)
				if len(got) != len(want) {
					t.Errorf("got %v, want %v", got, want)
					return
				}
				for i := range got {
					if got[i] != want[i] {
						t.Errorf("index %d: got %q, want %q", i, got[i], want[i])
					}
				}
			}
		})
	}
}

func TestSpecToHydra_NilFields(t *testing.T) {
	p := v1alpha1.OAuth2ClientParameters{}
	c := specToHydra(p)

	if c.ClientId != nil {
		t.Errorf("expected nil ClientId, got %v", c.ClientId)
	}
	if c.ClientName != nil {
		t.Errorf("expected nil ClientName, got %v", c.ClientName)
	}
	if c.GrantTypes != nil {
		t.Errorf("expected nil GrantTypes, got %v", c.GrantTypes)
	}
	if c.Scope != nil {
		t.Errorf("expected nil Scope, got %v", c.Scope)
	}
}

func TestObservationFromHydra(t *testing.T) {
	now := time.Now()
	c := &hydra.OAuth2Client{
		ClientId:                ptr("test-id"),
		CreatedAt:               &now,
		UpdatedAt:               &now,
		RegistrationAccessToken: ptr("reg-token"),
		RegistrationClientUri:   ptr("https://example.com/register"),
	}

	obs := observationFromHydra(c)

	if obs.ClientID != "test-id" {
		t.Errorf("expected ClientID %q, got %q", "test-id", obs.ClientID)
	}
	if obs.CreatedAt == "" {
		t.Error("expected CreatedAt to be set")
	}
	if obs.UpdatedAt == "" {
		t.Error("expected UpdatedAt to be set")
	}
	if obs.RegistrationAccessToken != "reg-token" {
		t.Errorf("expected RegistrationAccessToken %q, got %q", "reg-token", obs.RegistrationAccessToken)
	}
	if obs.RegistrationClientURI != "https://example.com/register" {
		t.Errorf("expected RegistrationClientURI %q, got %q", "https://example.com/register", obs.RegistrationClientURI)
	}
}

func TestObservationFromHydra_NilFields(t *testing.T) {
	c := &hydra.OAuth2Client{}
	obs := observationFromHydra(c)

	if obs.ClientID != "" {
		t.Errorf("expected empty ClientID, got %q", obs.ClientID)
	}
	if obs.CreatedAt != "" {
		t.Errorf("expected empty CreatedAt, got %q", obs.CreatedAt)
	}
}

func TestIsUpToDate(t *testing.T) {
	spec := v1alpha1.OAuth2ClientParameters{
		ClientID:   ptr("test"),
		ClientName: ptr("Test"),
		GrantTypes: []string{"client_credentials"},
		Scope:      ptr("openid"),
	}

	t.Run("MatchingState", func(t *testing.T) {
		observed := &hydra.OAuth2Client{
			ClientId:                ptr("test"),
			ClientName:              ptr("Test"),
			GrantTypes:              []string{"client_credentials"},
			Scope:                   ptr("openid"),
			TokenEndpointAuthMethod: ptr("client_secret_basic"), // SDK default
		}
		if !isUpToDate(spec, observed) {
			t.Error("expected up to date")
		}
	})

	t.Run("DriftedField", func(t *testing.T) {
		observed := &hydra.OAuth2Client{
			ClientId:   ptr("test"),
			ClientName: ptr("Modified Name"),
			GrantTypes: []string{"client_credentials"},
			Scope:      ptr("openid"),
		}
		if isUpToDate(spec, observed) {
			t.Error("expected not up to date when client_name differs")
		}
	})

	t.Run("ServerSetFieldsIgnored", func(t *testing.T) {
		now := time.Now()
		observed := &hydra.OAuth2Client{
			ClientId:                ptr("test"),
			ClientName:              ptr("Test"),
			GrantTypes:              []string{"client_credentials"},
			Scope:                   ptr("openid"),
			TokenEndpointAuthMethod: ptr("client_secret_basic"), // SDK default
			CreatedAt:               &now,
			UpdatedAt:               &now,
			ClientSecret:            ptr("server-generated-secret"),
			ClientSecretExpiresAt:   ptr(int64(0)),
			RegistrationAccessToken: ptr("reg-token"),
			RegistrationClientUri:   ptr("https://example.com/register"),
		}
		if !isUpToDate(spec, observed) {
			t.Error("expected up to date — server-set fields should be ignored")
		}
	})
}

func TestStripServerFields(t *testing.T) {
	now := time.Now()
	c := hydra.OAuth2Client{
		ClientId:                ptr("keep"),
		ClientName:              ptr("keep"),
		ClientSecret:            ptr("strip"),
		ClientSecretExpiresAt:   ptr(int64(123)),
		CreatedAt:               &now,
		UpdatedAt:               &now,
		RegistrationAccessToken: ptr("strip"),
		RegistrationClientUri:   ptr("strip"),
	}

	stripped := stripServerFields(c)

	if stripped.ClientId == nil || *stripped.ClientId != "keep" {
		t.Error("ClientId should be preserved")
	}
	if stripped.ClientName == nil || *stripped.ClientName != "keep" {
		t.Error("ClientName should be preserved")
	}
	if stripped.ClientSecret != nil {
		t.Error("ClientSecret should be stripped")
	}
	if stripped.ClientSecretExpiresAt != nil {
		t.Error("ClientSecretExpiresAt should be stripped")
	}
	if stripped.CreatedAt != nil {
		t.Error("CreatedAt should be stripped")
	}
	if stripped.UpdatedAt != nil {
		t.Error("UpdatedAt should be stripped")
	}
	if stripped.RegistrationAccessToken != nil {
		t.Error("RegistrationAccessToken should be stripped")
	}
	if stripped.RegistrationClientUri != nil {
		t.Error("RegistrationClientUri should be stripped")
	}

	// Original should be unmodified.
	if c.ClientSecret == nil {
		t.Error("original ClientSecret should not be modified")
	}
}

// --- Controller method tests ------------------------------------------------

func TestObserve_NoExternalName(t *testing.T) {
	e := &external{hydra: &mockHydra{}}
	cr := newTestCR("test", "")

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.ResourceExists {
		t.Error("expected ResourceExists=false when no external name is set")
	}
}

func TestObserve_NotFound(t *testing.T) {
	e := &external{
		hydra: &mockHydra{
			getFn: func(_ context.Context, id string) (*hydra.OAuth2Client, error) {
				if id != "test-client" {
					t.Errorf("expected id %q, got %q", "test-client", id)
				}
				return nil, nil
			},
		},
	}
	cr := newTestCR("test", "test-client")

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.ResourceExists {
		t.Error("expected ResourceExists=false when Hydra returns nil")
	}
}

func TestObserve_Exists_UpToDate(t *testing.T) {
	e := &external{
		hydra: &mockHydra{
			getFn: func(_ context.Context, _ string) (*hydra.OAuth2Client, error) {
				return &hydra.OAuth2Client{
					ClientId:                ptr("test-client"),
					ClientName:              ptr("Test Client"),
					GrantTypes:              []string{"client_credentials"},
					Scope:                   ptr("openid"),
					TokenEndpointAuthMethod: ptr("client_secret_basic"),
				}, nil
			},
		},
	}
	cr := newTestCR("test", "test-client")

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !obs.ResourceExists {
		t.Error("expected ResourceExists=true")
	}
	if !obs.ResourceUpToDate {
		t.Error("expected ResourceUpToDate=true")
	}
	if cr.Status.AtProvider.ClientID != "test-client" {
		t.Errorf("expected atProvider.clientId %q, got %q", "test-client", cr.Status.AtProvider.ClientID)
	}
}

func TestObserve_Exists_Drifted(t *testing.T) {
	e := &external{
		hydra: &mockHydra{
			getFn: func(_ context.Context, _ string) (*hydra.OAuth2Client, error) {
				return &hydra.OAuth2Client{
					ClientId:                ptr("test-client"),
					ClientName:              ptr("Drifted Name"),
					GrantTypes:              []string{"client_credentials"},
					Scope:                   ptr("openid"),
					TokenEndpointAuthMethod: ptr("client_secret_basic"),
				}, nil
			},
		},
	}
	cr := newTestCR("test", "test-client")

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !obs.ResourceExists {
		t.Error("expected ResourceExists=true")
	}
	if obs.ResourceUpToDate {
		t.Error("expected ResourceUpToDate=false when Hydra state differs")
	}
}

func TestObserve_Error(t *testing.T) {
	e := &external{
		hydra: &mockHydra{
			getFn: func(_ context.Context, _ string) (*hydra.OAuth2Client, error) {
				return nil, errors.New("connection refused")
			},
		},
	}
	cr := newTestCR("test", "test-client")

	_, err := e.Observe(context.Background(), cr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestObserve_LateInitClientID(t *testing.T) {
	e := &external{
		hydra: &mockHydra{
			getFn: func(_ context.Context, _ string) (*hydra.OAuth2Client, error) {
				return &hydra.OAuth2Client{
					ClientId:                ptr("auto-generated-id"),
					TokenEndpointAuthMethod: ptr("client_secret_basic"),
				}, nil
			},
		},
	}
	cr := newTestCR("test", "auto-generated-id")
	cr.Spec.ForProvider.ClientID = nil

	_, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cr.Spec.ForProvider.ClientID == nil || *cr.Spec.ForProvider.ClientID != "auto-generated-id" {
		t.Error("expected ClientID to be late-initialized")
	}
}

func TestObserve_NotManaged(t *testing.T) {
	e := &external{}
	_, err := e.Observe(context.Background(), &fake.Managed{})
	if err == nil {
		t.Fatal("expected error for non-OAuth2Client resource")
	}
}

func TestCreate_Success(t *testing.T) {
	e := &external{
		publicURL: "http://hydra.example.com",
		hydra: &mockHydra{
			createFn: func(_ context.Context, c hydra.OAuth2Client) (*hydra.OAuth2Client, error) {
				return &hydra.OAuth2Client{
					ClientId:     c.ClientId,
					ClientSecret: ptr("generated-secret"),
				}, nil
			},
		},
	}
	cr := newTestCR("test", "")

	result, err := e.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// External name should be set.
	if meta.GetExternalName(cr) != "test-client" {
		t.Errorf("expected external name %q, got %q", "test-client", meta.GetExternalName(cr))
	}

	// Connection details should include client_id and client_secret.
	if string(result.ConnectionDetails["client_id"]) != "test-client" {
		t.Errorf("expected client_id in connection details")
	}
	if string(result.ConnectionDetails["client_secret"]) != "generated-secret" {
		t.Errorf("expected client_secret in connection details")
	}

	// Endpoint URLs should be derived from publicURL.
	if string(result.ConnectionDetails["token_endpoint"]) != "http://hydra.example.com/oauth2/token" {
		t.Errorf("unexpected token_endpoint: %s", result.ConnectionDetails["token_endpoint"])
	}
	if string(result.ConnectionDetails["authorization_endpoint"]) != "http://hydra.example.com/oauth2/auth" {
		t.Errorf("unexpected authorization_endpoint: %s", result.ConnectionDetails["authorization_endpoint"])
	}
	if string(result.ConnectionDetails["issuer_url"]) != "http://hydra.example.com" {
		t.Errorf("unexpected issuer_url: %s", result.ConnectionDetails["issuer_url"])
	}
}

func TestCreate_NoPublicURL(t *testing.T) {
	e := &external{
		publicURL: "",
		hydra: &mockHydra{
			createFn: func(_ context.Context, c hydra.OAuth2Client) (*hydra.OAuth2Client, error) {
				return &hydra.OAuth2Client{
					ClientId:     c.ClientId,
					ClientSecret: ptr("secret"),
				}, nil
			},
		},
	}
	cr := newTestCR("test", "")

	result, err := e.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := result.ConnectionDetails["token_endpoint"]; ok {
		t.Error("token_endpoint should not be set when publicURL is empty")
	}
}

func TestCreate_Error(t *testing.T) {
	e := &external{
		hydra: &mockHydra{
			createFn: func(_ context.Context, _ hydra.OAuth2Client) (*hydra.OAuth2Client, error) {
				return nil, errors.New("server error")
			},
		},
	}
	cr := newTestCR("test", "")

	_, err := e.Create(context.Background(), cr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCreate_NoSecret(t *testing.T) {
	e := &external{
		hydra: &mockHydra{
			createFn: func(_ context.Context, c hydra.OAuth2Client) (*hydra.OAuth2Client, error) {
				return &hydra.OAuth2Client{ClientId: c.ClientId}, nil
			},
		},
	}
	cr := newTestCR("test", "")

	result, err := e.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := result.ConnectionDetails["client_secret"]; ok {
		t.Error("client_secret should not be set when Hydra returns no secret")
	}
}

func TestUpdate_Success(t *testing.T) {
	var calledID string
	e := &external{
		hydra: &mockHydra{
			updateFn: func(_ context.Context, id string, _ hydra.OAuth2Client) (*hydra.OAuth2Client, error) {
				calledID = id
				return &hydra.OAuth2Client{}, nil
			},
		},
	}
	cr := newTestCR("test", "test-client")

	_, err := e.Update(context.Background(), cr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calledID != "test-client" {
		t.Errorf("expected update for %q, got %q", "test-client", calledID)
	}
}

func TestUpdate_Error(t *testing.T) {
	e := &external{
		hydra: &mockHydra{
			updateFn: func(_ context.Context, _ string, _ hydra.OAuth2Client) (*hydra.OAuth2Client, error) {
				return nil, errors.New("conflict")
			},
		},
	}
	cr := newTestCR("test", "test-client")

	_, err := e.Update(context.Background(), cr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDelete_Success(t *testing.T) {
	var calledID string
	e := &external{
		hydra: &mockHydra{
			deleteFn: func(_ context.Context, id string) error {
				calledID = id
				return nil
			},
		},
	}
	cr := newTestCR("test", "test-client")

	_, err := e.Delete(context.Background(), cr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calledID != "test-client" {
		t.Errorf("expected delete for %q, got %q", "test-client", calledID)
	}
}

func TestDelete_Error(t *testing.T) {
	e := &external{
		hydra: &mockHydra{
			deleteFn: func(_ context.Context, _ string) error {
				return errors.New("internal error")
			},
		},
	}
	cr := newTestCR("test", "test-client")

	_, err := e.Delete(context.Background(), cr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDisconnect(t *testing.T) {
	e := &external{}
	if err := e.Disconnect(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
