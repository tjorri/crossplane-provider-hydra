// Package apis contains API Schema definitions for the provider.
package apis

import (
	"k8s.io/apimachinery/pkg/runtime"

	oauth2clientv1alpha1 "github.com/tjorri/crossplane-provider-hydra/apis/oauth2client/v1alpha1"
	configv1alpha1 "github.com/tjorri/crossplane-provider-hydra/apis/v1alpha1"
)

func init() {
	AddToSchemes = append(AddToSchemes,
		oauth2clientv1alpha1.SchemeBuilder.AddToScheme,
		configv1alpha1.SchemeBuilder.AddToScheme,
	)
}

// AddToSchemes may be used to add all resources defined in the project to a Scheme.
var AddToSchemes runtime.SchemeBuilder

// AddToScheme adds all resources to the given scheme.
func AddToScheme(s *runtime.Scheme) error {
	return AddToSchemes.AddToScheme(s)
}
