package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/alecthomas/kingpin/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"

	"github.com/tjorri/crossplane-provider-hydra/apis"
	hydracontroller "github.com/tjorri/crossplane-provider-hydra/internal/controller"
)

func main() {
	var (
		app              = kingpin.New(filepath.Base(os.Args[0]), "Crossplane provider for Ory Hydra.")
		debug            = app.Flag("debug", "Run with debug logging.").Short('d').Bool()
		syncInterval     = app.Flag("sync", "Controller manager sync period.").Default("1h").Duration()
		pollInterval     = app.Flag("poll", "Poll interval for external resource drift detection.").Default("1m").Duration()
		maxReconcileRate = app.Flag("max-reconcile-rate", "Max concurrent reconciles per controller.").Default("10").Int()
		leaderElection   = app.Flag("leader-election", "Use leader election.").Default("false").Envar("LEADER_ELECTION").Bool()
	)
	kingpin.MustParse(app.Parse(os.Args[1:]))

	zl := zap.New(zap.UseDevMode(*debug))
	log := logging.NewLogrLogger(zl)
	ctrl.SetLogger(zl)

	cfg := ctrl.GetConfigOrDie()

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Cache: cache.Options{
			SyncPeriod: syncInterval,
		},
		LeaderElection:   *leaderElection,
		LeaderElectionID: "crossplane-leader-election-provider-hydra",
		LeaseDuration:    func() *time.Duration { d := 60 * time.Second; return &d }(),
		RenewDeadline:    func() *time.Duration { d := 50 * time.Second; return &d }(),
	})
	kingpin.FatalIfError(err, "Cannot create controller manager")

	kingpin.FatalIfError(apis.AddToScheme(mgr.GetScheme()), "Cannot add APIs to scheme")

	o := controller.Options{
		Logger:                  log,
		MaxConcurrentReconciles: *maxReconcileRate,
		PollInterval:            *pollInterval,
		Features:                &feature.Flags{},
	}

	kingpin.FatalIfError(hydracontroller.Setup(mgr, o), "Cannot setup controllers")
	kingpin.FatalIfError(mgr.Start(ctrl.SetupSignalHandler()), "Cannot start controller manager")
}
