package config

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/providerconfig"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	configv1alpha1 "github.com/tjorri/crossplane-provider-hydra/apis/v1alpha1"
)

// Setup adds a controller that reconciles ProviderConfig resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := providerconfig.ControllerName(configv1alpha1.SchemeGroupVersion.WithKind("ProviderConfig").GroupKind().String())

	r := providerconfig.NewReconciler(mgr,
		resource.ProviderConfigKinds{
			Config:    configv1alpha1.SchemeGroupVersion.WithKind("ProviderConfig"),
			Usage:     configv1alpha1.SchemeGroupVersion.WithKind("ProviderConfigUsage"),
			UsageList: configv1alpha1.SchemeGroupVersion.WithKind("ProviderConfigUsageList"),
		},
		providerconfig.WithLogger(o.Logger.WithValues("controller", name)),
		providerconfig.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&configv1alpha1.ProviderConfig{}).
		Complete(r)
}
