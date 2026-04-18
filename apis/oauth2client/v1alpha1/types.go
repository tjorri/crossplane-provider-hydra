package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
)

// OAuth2ClientParameters defines the configurable fields of an OAuth2 client in Ory Hydra.
// These map to the Hydra OAuth2Client model with full field parity.
type OAuth2ClientParameters struct {
	// ClientID is the OAuth 2.0 Client ID. If not set, Hydra will generate a UUID.
	// +optional
	ClientID *string `json:"clientId,omitempty"`

	// ClientName is the human-readable name of the client shown during authorization.
	// +optional
	ClientName *string `json:"clientName,omitempty"`

	// ClientSecret is the OAuth 2.0 Client Secret. If not set, Hydra will generate one.
	// The secret is only returned by Hydra on creation.
	// +optional
	ClientSecret *string `json:"clientSecret,omitempty"`

	// ClientURI is a URL of a web page providing information about the client.
	// +optional
	ClientURI *string `json:"clientUri,omitempty"`

	// GrantTypes lists the OAuth 2.0 grant types the client may use.
	// +optional
	GrantTypes []string `json:"grantTypes,omitempty"`

	// ResponseTypes lists the OAuth 2.0 response types the client may use.
	// +optional
	ResponseTypes []string `json:"responseTypes,omitempty"`

	// RedirectURIs lists the allowed redirect URIs for the client.
	// Must match exactly (no wildcard patterns).
	// +optional
	RedirectURIs []string `json:"redirectUris,omitempty"`

	// Scope is a space-delimited list of scope values the client can request.
	// +optional
	Scope *string `json:"scope,omitempty"`

	// Audience lists the allowed audience values for the client.
	// +optional
	Audience []string `json:"audience,omitempty"`

	// TokenEndpointAuthMethod is the authentication method for the token endpoint.
	// One of: client_secret_basic, client_secret_post, private_key_jwt, none.
	// +kubebuilder:validation:Enum=client_secret_basic;client_secret_post;private_key_jwt;none
	// +optional
	TokenEndpointAuthMethod *string `json:"tokenEndpointAuthMethod,omitempty"`

	// TokenEndpointAuthSigningAlg is the signing algorithm for token endpoint authentication.
	// +optional
	TokenEndpointAuthSigningAlg *string `json:"tokenEndpointAuthSigningAlg,omitempty"`

	// Contacts lists email addresses of people responsible for this client.
	// +optional
	Contacts []string `json:"contacts,omitempty"`

	// LogoURI is a URL pointing to the client's logo.
	// +optional
	LogoURI *string `json:"logoUri,omitempty"`

	// PolicyURI points to a human-readable privacy policy for the client.
	// +optional
	PolicyURI *string `json:"policyUri,omitempty"`

	// TosURI points to the client's terms of service.
	// +optional
	TosURI *string `json:"tosUri,omitempty"`

	// AllowedCORSOrigins lists the allowed CORS origins for this client.
	// +optional
	AllowedCORSOrigins []string `json:"allowedCorsOrigins,omitempty"`

	// Owner identifies the owner of the OAuth 2.0 client.
	// +optional
	Owner *string `json:"owner,omitempty"`

	// SubjectType defines the subject type requested for responses to this client.
	// +kubebuilder:validation:Enum=pairwise;public
	// +optional
	SubjectType *string `json:"subjectType,omitempty"`

	// SectorIdentifierURI is a URL using the https scheme to derive pairwise subject identifiers.
	// +optional
	SectorIdentifierURI *string `json:"sectorIdentifierUri,omitempty"`

	// SkipConsent skips the consent screen for this client if set to true.
	// +optional
	SkipConsent *bool `json:"skipConsent,omitempty"`

	// SkipLogoutConsent skips the logout consent screen for this client.
	// +optional
	SkipLogoutConsent *bool `json:"skipLogoutConsent,omitempty"`

	// AccessTokenStrategy defines the access token strategy: jwt or opaque.
	// +kubebuilder:validation:Enum=jwt;opaque
	// +optional
	AccessTokenStrategy *string `json:"accessTokenStrategy,omitempty"`

	// BackchannelLogoutURI is the URL for backchannel logout notification.
	// +optional
	BackchannelLogoutURI *string `json:"backchannelLogoutUri,omitempty"`

	// BackchannelLogoutSessionRequired indicates whether the RP requires a sid claim in the logout token.
	// +optional
	BackchannelLogoutSessionRequired *bool `json:"backchannelLogoutSessionRequired,omitempty"`

	// FrontchannelLogoutURI is the URL for frontchannel logout.
	// +optional
	FrontchannelLogoutURI *string `json:"frontchannelLogoutUri,omitempty"`

	// FrontchannelLogoutSessionRequired indicates whether the RP requires a sid claim.
	// +optional
	FrontchannelLogoutSessionRequired *bool `json:"frontchannelLogoutSessionRequired,omitempty"`

	// PostLogoutRedirectURIs lists the URLs the client may redirect to after logout.
	// +optional
	PostLogoutRedirectURIs []string `json:"postLogoutRedirectUris,omitempty"`

	// RequestObjectSigningAlg is the algorithm used to sign request objects.
	// +optional
	RequestObjectSigningAlg *string `json:"requestObjectSigningAlg,omitempty"`

	// RequestURIs lists pre-registered request_uri values for the client.
	// +optional
	RequestURIs []string `json:"requestUris,omitempty"`

	// UserinfoSignedResponseAlg is the algorithm used to sign userinfo responses.
	// +optional
	UserinfoSignedResponseAlg *string `json:"userinfoSignedResponseAlg,omitempty"`

	// IDTokenSignedResponseAlg is the algorithm used to sign ID tokens.
	// +optional
	IDTokenSignedResponseAlg *string `json:"idTokenSignedResponseAlg,omitempty"`

	// JwksURI is the URL for the client's JSON Web Key Set.
	// +optional
	JwksURI *string `json:"jwksUri,omitempty"`

	// Jwks is the client's JSON Web Key Set document value.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Jwks *runtime.RawExtension `json:"jwks,omitempty"`

	// Metadata is arbitrary metadata associated with the client.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Metadata *runtime.RawExtension `json:"metadata,omitempty"`

	// AuthorizationCodeGrantAccessTokenLifespan controls the access token lifespan
	// for the authorization code grant. Uses Go duration format (e.g., "30m", "1h").
	// +optional
	AuthorizationCodeGrantAccessTokenLifespan *string `json:"authorizationCodeGrantAccessTokenLifespan,omitempty"`

	// AuthorizationCodeGrantIDTokenLifespan controls the ID token lifespan
	// for the authorization code grant.
	// +optional
	AuthorizationCodeGrantIDTokenLifespan *string `json:"authorizationCodeGrantIdTokenLifespan,omitempty"`

	// AuthorizationCodeGrantRefreshTokenLifespan controls the refresh token lifespan
	// for the authorization code grant.
	// +optional
	AuthorizationCodeGrantRefreshTokenLifespan *string `json:"authorizationCodeGrantRefreshTokenLifespan,omitempty"`

	// ClientCredentialsGrantAccessTokenLifespan controls the access token lifespan
	// for the client credentials grant.
	// +optional
	ClientCredentialsGrantAccessTokenLifespan *string `json:"clientCredentialsGrantAccessTokenLifespan,omitempty"`

	// ImplicitGrantAccessTokenLifespan controls the access token lifespan
	// for the implicit grant.
	// +optional
	ImplicitGrantAccessTokenLifespan *string `json:"implicitGrantAccessTokenLifespan,omitempty"`

	// ImplicitGrantIDTokenLifespan controls the ID token lifespan
	// for the implicit grant.
	// +optional
	ImplicitGrantIDTokenLifespan *string `json:"implicitGrantIdTokenLifespan,omitempty"`

	// RefreshTokenGrantAccessTokenLifespan controls the access token lifespan
	// for the refresh token grant.
	// +optional
	RefreshTokenGrantAccessTokenLifespan *string `json:"refreshTokenGrantAccessTokenLifespan,omitempty"`

	// RefreshTokenGrantIDTokenLifespan controls the ID token lifespan
	// for the refresh token grant.
	// +optional
	RefreshTokenGrantIDTokenLifespan *string `json:"refreshTokenGrantIdTokenLifespan,omitempty"`

	// RefreshTokenGrantRefreshTokenLifespan controls the refresh token lifespan
	// for the refresh token grant.
	// +optional
	RefreshTokenGrantRefreshTokenLifespan *string `json:"refreshTokenGrantRefreshTokenLifespan,omitempty"`

	// JwtBearerGrantAccessTokenLifespan controls the access token lifespan
	// for the JWT bearer grant.
	// +optional
	JwtBearerGrantAccessTokenLifespan *string `json:"jwtBearerGrantAccessTokenLifespan,omitempty"`
}

// OAuth2ClientObservation defines the server-set fields observed from Hydra.
type OAuth2ClientObservation struct {
	// ClientID is the resolved client ID (especially relevant when auto-generated).
	// +optional
	ClientID string `json:"clientId,omitempty"`

	// CreatedAt is the time at which the client was created.
	// +optional
	CreatedAt string `json:"createdAt,omitempty"`

	// UpdatedAt is the time at which the client was last updated.
	// +optional
	UpdatedAt string `json:"updatedAt,omitempty"`

	// RegistrationAccessToken is the access token for the dynamic client registration endpoint.
	// +optional
	RegistrationAccessToken string `json:"registrationAccessToken,omitempty"`

	// RegistrationClientURI is the URL of the dynamic client registration endpoint.
	// +optional
	RegistrationClientURI string `json:"registrationClientUri,omitempty"`
}

// OAuth2ClientSpec defines the desired state of an OAuth2Client.
type OAuth2ClientSpec struct {
	xpv1.ResourceSpec `json:",inline"`

	// ForProvider contains the desired state of the OAuth2 client in Hydra.
	ForProvider OAuth2ClientParameters `json:"forProvider"`
}

// OAuth2ClientStatus defines the observed state of an OAuth2Client.
type OAuth2ClientStatus struct {
	xpv1.ResourceStatus `json:",inline"`

	// AtProvider contains the observed state of the OAuth2 client in Hydra.
	// +optional
	AtProvider OAuth2ClientObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,hydra}
// +kubebuilder:printcolumn:name="SYNCED",type=string,JSONPath=`.status.conditions[?(@.type=='Synced')].status`
// +kubebuilder:printcolumn:name="READY",type=string,JSONPath=`.status.conditions[?(@.type=='Ready')].status`
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type=string,JSONPath=`.metadata.annotations.crossplane\.io/external-name`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// OAuth2Client is the Schema for Ory Hydra OAuth2 clients.
type OAuth2Client struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OAuth2ClientSpec   `json:"spec"`
	Status OAuth2ClientStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OAuth2ClientList contains a list of OAuth2Client.
type OAuth2ClientList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OAuth2Client `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OAuth2Client{}, &OAuth2ClientList{})
}
