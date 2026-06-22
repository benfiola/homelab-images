package internal

import (
	"context"
	"fmt"
	"net/http"

	"github.com/benfiola/homelab-images/pvc-restore/internal/reconciler"
	"github.com/benfiola/homelab-images/pvc-restore/internal/scheme"
	"github.com/benfiola/homelab-images/shared/pkg/logging"
	"github.com/go-logr/logr"
	"k8s.io/client-go/tools/clientcmd"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

type Opts struct {
	CacheStorageClass string
	HealthAddress     string
	Kubeconfig        string
	LeaderElection    bool
	MetricsAddress    string
}

type Controller struct {
	Manager controllerruntime.Manager
}

func New(opts *Opts) (*Controller, error) {
	healthAddress := opts.HealthAddress
	if healthAddress == "" {
		healthAddress = ":8081"
	}
	metricsAddress := opts.MetricsAddress
	if metricsAddress == "" {
		metricsAddress = ":8080"
	}

	kubeConfig, err := clientcmd.BuildConfigFromFlags("", opts.Kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
	}

	builtScheme, err := scheme.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build scheme: %w", err)
	}

	webhookServer := webhook.NewServer(webhook.Options{Port: 0})
	manager, err := controllerruntime.NewManager(kubeConfig, controllerruntime.Options{
		HealthProbeBindAddress: healthAddress,
		LeaderElection:         opts.LeaderElection,
		LeaderElectionID:       "pvc-restore.homelab-images.benfiola.com",
		Metrics:                server.Options{BindAddress: metricsAddress},
		WebhookServer:          webhookServer,
		Scheme:                 builtScheme,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create controller manager: %w", err)
	}

	client := manager.GetClient()
	reconcilers := []reconciler.Reconciler{
		&reconciler.PVCRestoreReconciler{CacheStorageClass: opts.CacheStorageClass, Client: client, Scheme: builtScheme},
		&reconciler.LockCleanupReconciler{Client: client, Scheme: builtScheme},
		&reconciler.PersistentVolumeClaimReconciler{Client: client, Scheme: builtScheme},
	}

	for index, rec := range reconcilers {
		err = rec.Register(manager)
		if err != nil {
			return nil, fmt.Errorf("failed to register reconciler %d: %w", index, err)
		}
	}

	return &Controller{
		Manager: manager,
	}, nil
}

func (c *Controller) Run(ctx context.Context) error {
	logger := logging.FromContext(ctx)

	logger.Info("setting controller-runtime logger")
	crLogger := logr.FromSlogHandler(logger.Handler()).WithName("controller-runtime")
	controllerruntime.SetLogger(crLogger)

	logger.Info("adding probes")
	err := c.Manager.AddHealthzCheck("ping", healthz.Ping)
	if err != nil {
		logger.Error("failed to setup liveness probe", "error", err)
	}

	readyz := func(req *http.Request) error {
		val := c.Manager.GetCache().WaitForCacheSync(req.Context())
		if !val {
			logger.Warn("readyz cache sync check failed")
			return fmt.Errorf("readyz cache sync check failed")
		}
		return nil
	}

	err = c.Manager.AddReadyzCheck("caches", readyz)
	if err != nil {
		logger.Error("failed to setup readiness probe", "error", err)
	}

	logger.Info("starting controller")
	err = c.Manager.Start(ctx)
	if err != nil {
		logger.Error("controller manager exited with error", "error", err)
		return err
	}

	return nil
}
