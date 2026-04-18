// Package v1alpha1 contains the v1alpha1 group OAuth2Client types.
// +kubebuilder:object:generate=true
// +groupName=hydra.crossplane.io
// +versionName=v1alpha1
package v1alpha1

import (
	"reflect"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// Package type metadata.
const (
	Group   = "hydra.crossplane.io"
	Version = "v1alpha1"
)

var (
	// SchemeGroupVersion is group version used to register these objects.
	SchemeGroupVersion = schema.GroupVersion{Group: Group, Version: Version}

	// SchemeBuilder is used to add go types to the GroupVersionResource scheme.
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}

	// OAuth2ClientKind is the kind of the OAuth2Client resource.
	OAuth2ClientKind = reflect.TypeOf(OAuth2Client{}).Name()

	// OAuth2ClientGroupKind is the group and kind of the OAuth2Client resource.
	OAuth2ClientGroupKind = schema.GroupKind{Group: Group, Kind: OAuth2ClientKind}.String()

	// OAuth2ClientKindAPIVersion is the kind and API version of the OAuth2Client resource.
	OAuth2ClientKindAPIVersion = OAuth2ClientKind + "." + Group + "/" + Version

	// OAuth2ClientGroupVersionKind is the group, version, and kind of the OAuth2Client resource.
	OAuth2ClientGroupVersionKind = SchemeGroupVersion.WithKind(OAuth2ClientKind)
)
