package controller

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"

	"github.com/tjorri/crossplane-provider-hydra/internal/controller/config"
	"github.com/tjorri/crossplane-provider-hydra/internal/controller/oauth2client"
)

// Setup creates all controllers with the supplied manager and options.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		config.Setup,
		oauth2client.Setup,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}
