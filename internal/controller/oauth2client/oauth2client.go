package oauth2client

import (
	"context"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	hydra "github.com/ory/hydra-client-go/v2"

	"github.com/tjorri/crossplane-provider-hydra/apis/oauth2client/v1alpha1"
	configv1alpha1 "github.com/tjorri/crossplane-provider-hydra/apis/v1alpha1"
	hydraclient "github.com/tjorri/crossplane-provider-hydra/internal/clients"
)

const (
	errNotOAuth2Client = "managed resource is not an OAuth2Client"
	errGetPC           = "cannot get ProviderConfig"
	errGetSecret       = "cannot get credentials secret"
	errObserve         = "cannot observe OAuth2 client"
	errCreate          = "cannot create OAuth2 client"
	errUpdate          = "cannot update OAuth2 client"
	errDelete          = "cannot delete OAuth2 client"
)

// Setup adds a controller that reconciles OAuth2Client managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.OAuth2ClientGroupKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.OAuth2ClientGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithPollInterval(o.PollInterval),
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1alpha1.OAuth2Client{}).
		Complete(r)
}

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.OAuth2Client)
	if !ok {
		return nil, errors.New(errNotOAuth2Client)
	}

	pcRef := cr.Spec.ProviderConfigReference
	if pcRef == nil {
		return nil, errors.New("providerConfigRef is required")
	}

	pc := &configv1alpha1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: pcRef.Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	var bearerToken string
	if pc.Spec.Credentials != nil && pc.Spec.Credentials.Source == "Secret" && pc.Spec.Credentials.SecretRef != nil {
		secret := &corev1.Secret{}
		if err := c.kube.Get(ctx, types.NamespacedName{
			Namespace: pc.Spec.Credentials.SecretRef.Namespace,
			Name:      pc.Spec.Credentials.SecretRef.Name,
		}, secret); err != nil {
			return nil, errors.Wrap(err, errGetSecret)
		}
		bearerToken = string(secret.Data[pc.Spec.Credentials.SecretRef.Key])
	}

	hc := hydraclient.NewHydraClient(pc.Spec.AdminURL, bearerToken)

	return &external{
		kube:      c.kube,
		hydra:     hc,
		publicURL: pc.Spec.PublicURL,
	}, nil
}

type external struct {
	kube      client.Client
	hydra     *hydraclient.HydraClient
	publicURL string
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.OAuth2Client)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotOAuth2Client)
	}

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	observed, err := e.hydra.GetOAuth2Client(ctx, externalName)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errObserve)
	}
	if observed == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	cr.Status.AtProvider = observationFromHydra(observed)

	if cr.Spec.ForProvider.ClientID == nil && observed.ClientId != nil {
		cr.Spec.ForProvider.ClientID = observed.ClientId
	}

	upToDate := isUpToDate(cr.Spec.ForProvider, observed)
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.OAuth2Client)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotOAuth2Client)
	}

	hydraClient := specToHydra(cr.Spec.ForProvider)

	created, err := e.hydra.CreateOAuth2Client(ctx, hydraClient)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreate)
	}

	meta.SetExternalName(cr, created.GetClientId())

	connDetails := managed.ConnectionDetails{
		"client_id": []byte(created.GetClientId()),
	}
	if created.ClientSecret != nil && *created.ClientSecret != "" {
		connDetails["client_secret"] = []byte(*created.ClientSecret)
	}
	if e.publicURL != "" {
		connDetails["token_endpoint"] = []byte(strings.TrimRight(e.publicURL, "/") + "/oauth2/token")
		connDetails["authorization_endpoint"] = []byte(strings.TrimRight(e.publicURL, "/") + "/oauth2/auth")
		connDetails["issuer_url"] = []byte(e.publicURL)
	}

	return managed.ExternalCreation{
		ConnectionDetails: connDetails,
	}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.OAuth2Client)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotOAuth2Client)
	}

	externalName := meta.GetExternalName(cr)
	hydraClient := specToHydra(cr.Spec.ForProvider)

	_, err := e.hydra.UpdateOAuth2Client(ctx, externalName, hydraClient)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdate)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.OAuth2Client)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotOAuth2Client)
	}

	externalName := meta.GetExternalName(cr)
	return managed.ExternalDelete{}, errors.Wrap(e.hydra.DeleteOAuth2Client(ctx, externalName), errDelete)
}

func (e *external) Disconnect(_ context.Context) error {
	return nil
}

// specToHydra converts the CRD spec to a Hydra SDK OAuth2Client.
func specToHydra(p v1alpha1.OAuth2ClientParameters) hydra.OAuth2Client {
	c := *hydra.NewOAuth2Client()

	if p.ClientID != nil {
		c.ClientId = p.ClientID
	}
	if p.ClientName != nil {
		c.ClientName = p.ClientName
	}
	if p.ClientSecret != nil {
		c.ClientSecret = p.ClientSecret
	}
	if p.ClientURI != nil {
		c.ClientUri = p.ClientURI
	}
	if p.GrantTypes != nil {
		c.GrantTypes = p.GrantTypes
	}
	if p.ResponseTypes != nil {
		c.ResponseTypes = p.ResponseTypes
	}
	if p.RedirectURIs != nil {
		c.RedirectUris = p.RedirectURIs
	}
	if p.Scope != nil {
		c.Scope = p.Scope
	}
	if p.Audience != nil {
		c.Audience = p.Audience
	}
	if p.TokenEndpointAuthMethod != nil {
		c.TokenEndpointAuthMethod = p.TokenEndpointAuthMethod
	}
	if p.TokenEndpointAuthSigningAlg != nil {
		c.TokenEndpointAuthSigningAlg = p.TokenEndpointAuthSigningAlg
	}
	if p.Contacts != nil {
		c.Contacts = p.Contacts
	}
	if p.LogoURI != nil {
		c.LogoUri = p.LogoURI
	}
	if p.PolicyURI != nil {
		c.PolicyUri = p.PolicyURI
	}
	if p.TosURI != nil {
		c.TosUri = p.TosURI
	}
	if p.AllowedCORSOrigins != nil {
		c.AllowedCorsOrigins = p.AllowedCORSOrigins
	}
	if p.Owner != nil {
		c.Owner = p.Owner
	}
	if p.SubjectType != nil {
		c.SubjectType = p.SubjectType
	}
	if p.SectorIdentifierURI != nil {
		c.SectorIdentifierUri = p.SectorIdentifierURI
	}
	if p.SkipConsent != nil {
		c.SkipConsent = p.SkipConsent
	}
	if p.SkipLogoutConsent != nil {
		c.SkipLogoutConsent = p.SkipLogoutConsent
	}
	if p.AccessTokenStrategy != nil {
		c.AccessTokenStrategy = p.AccessTokenStrategy
	}
	if p.BackchannelLogoutURI != nil {
		c.BackchannelLogoutUri = p.BackchannelLogoutURI
	}
	if p.BackchannelLogoutSessionRequired != nil {
		c.BackchannelLogoutSessionRequired = p.BackchannelLogoutSessionRequired
	}
	if p.FrontchannelLogoutURI != nil {
		c.FrontchannelLogoutUri = p.FrontchannelLogoutURI
	}
	if p.FrontchannelLogoutSessionRequired != nil {
		c.FrontchannelLogoutSessionRequired = p.FrontchannelLogoutSessionRequired
	}
	if p.PostLogoutRedirectURIs != nil {
		c.PostLogoutRedirectUris = p.PostLogoutRedirectURIs
	}
	if p.RequestObjectSigningAlg != nil {
		c.RequestObjectSigningAlg = p.RequestObjectSigningAlg
	}
	if p.RequestURIs != nil {
		c.RequestUris = p.RequestURIs
	}
	if p.UserinfoSignedResponseAlg != nil {
		c.UserinfoSignedResponseAlg = p.UserinfoSignedResponseAlg
	}
	if p.JwksURI != nil {
		c.JwksUri = p.JwksURI
	}
	if p.AuthorizationCodeGrantAccessTokenLifespan != nil {
		c.AuthorizationCodeGrantAccessTokenLifespan = p.AuthorizationCodeGrantAccessTokenLifespan
	}
	if p.AuthorizationCodeGrantIDTokenLifespan != nil {
		c.AuthorizationCodeGrantIdTokenLifespan = p.AuthorizationCodeGrantIDTokenLifespan
	}
	if p.AuthorizationCodeGrantRefreshTokenLifespan != nil {
		c.AuthorizationCodeGrantRefreshTokenLifespan = p.AuthorizationCodeGrantRefreshTokenLifespan
	}
	if p.ClientCredentialsGrantAccessTokenLifespan != nil {
		c.ClientCredentialsGrantAccessTokenLifespan = p.ClientCredentialsGrantAccessTokenLifespan
	}
	if p.ImplicitGrantAccessTokenLifespan != nil {
		c.ImplicitGrantAccessTokenLifespan = p.ImplicitGrantAccessTokenLifespan
	}
	if p.ImplicitGrantIDTokenLifespan != nil {
		c.ImplicitGrantIdTokenLifespan = p.ImplicitGrantIDTokenLifespan
	}
	if p.RefreshTokenGrantAccessTokenLifespan != nil {
		c.RefreshTokenGrantAccessTokenLifespan = p.RefreshTokenGrantAccessTokenLifespan
	}
	if p.RefreshTokenGrantIDTokenLifespan != nil {
		c.RefreshTokenGrantIdTokenLifespan = p.RefreshTokenGrantIDTokenLifespan
	}
	if p.RefreshTokenGrantRefreshTokenLifespan != nil {
		c.RefreshTokenGrantRefreshTokenLifespan = p.RefreshTokenGrantRefreshTokenLifespan
	}
	if p.JwtBearerGrantAccessTokenLifespan != nil {
		c.JwtBearerGrantAccessTokenLifespan = p.JwtBearerGrantAccessTokenLifespan
	}

	return c
}

// observationFromHydra converts a Hydra SDK OAuth2Client to CRD observation.
func observationFromHydra(c *hydra.OAuth2Client) v1alpha1.OAuth2ClientObservation {
	obs := v1alpha1.OAuth2ClientObservation{}
	if c.ClientId != nil {
		obs.ClientID = *c.ClientId
	}
	if c.CreatedAt != nil {
		obs.CreatedAt = c.CreatedAt.String()
	}
	if c.UpdatedAt != nil {
		obs.UpdatedAt = c.UpdatedAt.String()
	}
	if c.RegistrationAccessToken != nil {
		obs.RegistrationAccessToken = *c.RegistrationAccessToken
	}
	if c.RegistrationClientUri != nil {
		obs.RegistrationClientURI = *c.RegistrationClientUri
	}
	return obs
}

// isUpToDate checks if the desired spec matches the observed Hydra state.
// Server-set fields are ignored.
func isUpToDate(desired v1alpha1.OAuth2ClientParameters, observed *hydra.OAuth2Client) bool {
	generated := specToHydra(desired)

	return cmp.Equal(
		stripServerFields(generated),
		stripServerFields(*observed),
	)
}

// stripServerFields zeroes out fields that Hydra sets and manages internally.
func stripServerFields(c hydra.OAuth2Client) hydra.OAuth2Client {
	c.ClientSecret = nil
	c.ClientSecretExpiresAt = nil
	c.CreatedAt = nil
	c.UpdatedAt = nil
	c.RegistrationAccessToken = nil
	c.RegistrationClientUri = nil
	return c
}

var _ managed.ExternalClient = &external{}
var _ managed.ExternalConnector = &connector{}
