package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
)

// ProviderConfigSpec defines the desired state of a ProviderConfig.
type ProviderConfigSpec struct {
	// AdminURL is the URL of the Hydra Admin API (e.g., http://hydra-admin:4445).
	// +kubebuilder:validation:Required
	AdminURL string `json:"adminUrl"`

	// PublicURL is the URL of the Hydra Public API (e.g., http://hydra-public:4444).
	// Used to derive connection detail endpoint URLs. If omitted, endpoint
	// URLs will not be included in connection details.
	// +optional
	PublicURL string `json:"publicUrl,omitempty"`

	// Credentials configures how the provider authenticates to the Hydra Admin API.
	// +optional
	Credentials *ProviderCredentials `json:"credentials,omitempty"`
}

// ProviderCredentials configures authentication for the Hydra Admin API.
type ProviderCredentials struct {
	// Source of the credentials.
	// +kubebuilder:validation:Enum=None;Secret
	// +kubebuilder:default=None
	Source string `json:"source"`

	// SecretRef references a Secret containing a bearer token.
	// Required when source is "Secret".
	// +optional
	SecretRef *xpv1.SecretKeySelector `json:"secretRef,omitempty"`
}

// ProviderConfigStatus represents the observed state of a ProviderConfig.
type ProviderConfigStatus struct {
	xpv1.ProviderConfigStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,provider,hydra}
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// ProviderConfig configures how the Hydra provider connects to an Ory Hydra instance.
type ProviderConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProviderConfigSpec   `json:"spec"`
	Status ProviderConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProviderConfigList contains a list of ProviderConfig.
type ProviderConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProviderConfig `json:"items"`
}

// ProviderConfigUsage tracks a usage of a ProviderConfig.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Cluster,categories={crossplane,provider,hydra}
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:printcolumn:name="CONFIG-NAME",type=string,JSONPath=`.providerConfigRef.name`
// +kubebuilder:printcolumn:name="RESOURCE-KIND",type=string,JSONPath=`.resourceRef.kind`
// +kubebuilder:printcolumn:name="RESOURCE-NAME",type=string,JSONPath=`.resourceRef.name`
type ProviderConfigUsage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	xpv1.ProviderConfigUsage `json:",inline"`
}

// +kubebuilder:object:root=true

// ProviderConfigUsageList contains a list of ProviderConfigUsage.
type ProviderConfigUsageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProviderConfigUsage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ProviderConfig{}, &ProviderConfigList{})
	SchemeBuilder.Register(&ProviderConfigUsage{}, &ProviderConfigUsageList{})
}
